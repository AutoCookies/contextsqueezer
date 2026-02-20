#include "contextsqueeze.h"

#include <algorithm>
#include <array>
#include <cmath>
#include <cstdint>
#include <cstdlib>
#include <cstring>
#include <string>
#include <unordered_map>
#include <unordered_set>
#include <utility>
#include <vector>

namespace {

constexpr const char* kVersion = "0.2.0";
constexpr size_t kMinBlockLen = 120;

struct Span {
  size_t start;
  size_t end;
};

struct SentenceData {
  Span span;
  bool anchor = false;
  bool removed = false;
  std::unordered_map<std::string, int> tf;
  std::vector<std::pair<std::string, int>> top_tokens;
  double score = 0.0;
};

bool is_ascii_alnum(unsigned char c) {
  return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9');
}

char ascii_lower(unsigned char c) {
  if (c >= 'A' && c <= 'Z') {
    return static_cast<char>(c - 'A' + 'a');
  }
  return static_cast<char>(c);
}

uint64_t fnv1a_64(const char* data, size_t len) {
  uint64_t hash = 1469598103934665603ULL;
  for (size_t i = 0; i < len; ++i) {
    hash ^= static_cast<unsigned char>(data[i]);
    hash *= 1099511628211ULL;
  }
  return hash;
}

bool is_abbrev(const std::string& token) {
  static const std::unordered_set<std::string> kAbbrev = {
      "e.g", "i.e", "mr", "mrs", "ms", "dr", "vs", "etc", "prof", "sr", "jr"};
  return kAbbrev.find(token) != kAbbrev.end();
}

std::vector<Span> segment_sentences(const char* data, size_t len) {
  std::vector<Span> spans;
  size_t start = 0;
  size_t i = 0;
  while (i < len) {
    if (i + 1 < len && data[i] == '\n' && data[i + 1] == '\n') {
      if (start < i) {
        spans.push_back({start, i});
      }
      i += 2;
      start = i;
      continue;
    }

    if (data[i] == '.' || data[i] == '?' || data[i] == '!') {
      size_t t_end = i;
      size_t t_start = i;
      while (t_start > start && is_ascii_alnum(static_cast<unsigned char>(data[t_start - 1]))) {
        --t_start;
      }
      std::string token;
      token.reserve(t_end - t_start + 1);
      for (size_t j = t_start; j < t_end; ++j) {
        token.push_back(ascii_lower(static_cast<unsigned char>(data[j])));
      }
      if (!is_abbrev(token)) {
        size_t end = i + 1;
        while (end < len && (data[end] == ' ' || data[end] == '\t' || data[end] == '\n' || data[end] == '\r')) {
          if (end + 1 < len && data[end] == '\n' && data[end + 1] == '\n') {
            break;
          }
          ++end;
        }
        if (start < end) {
          spans.push_back({start, end});
        }
        start = end;
        i = end;
        continue;
      }
    }
    ++i;
  }
  if (start < len) {
    spans.push_back({start, len});
  }
  return spans;
}

void mark_span(std::vector<unsigned char>* keep_mask, Span span) {
  for (size_t i = span.start; i < span.end && i < keep_mask->size(); ++i) {
    (*keep_mask)[i] = 0;
  }
}

std::vector<Span> paragraph_spans(const char* data, size_t len) {
  std::vector<Span> out;
  size_t start = 0;
  size_t i = 0;
  while (i < len) {
    if (i + 1 < len && data[i] == '\n' && data[i + 1] == '\n') {
      if (start < i) {
        out.push_back({start, i});
      }
      i += 2;
      start = i;
      continue;
    }
    ++i;
  }
  if (start < len) {
    out.push_back({start, len});
  }
  return out;
}

double unique_ratio(const char* data, Span s) {
  std::array<bool, 256> seen{};
  size_t unique = 0;
  for (size_t i = s.start; i < s.end; ++i) {
    unsigned char c = static_cast<unsigned char>(data[i]);
    if (!seen[c]) {
      seen[c] = true;
      ++unique;
    }
  }
  const size_t n = s.end - s.start;
  return n == 0 ? 1.0 : static_cast<double>(unique) / static_cast<double>(n);
}

bool contains_subspan(const char* data, Span s, const char* needle) {
  const size_t n = s.end - s.start;
  const size_t m = std::strlen(needle);
  if (m == 0 || m > n) {
    return false;
  }
  for (size_t i = s.start; i + m <= s.end; ++i) {
    if (std::memcmp(data + i, needle, m) == 0) {
      return true;
    }
  }
  return false;
}

bool is_numeric_heavy(const char* data, Span s) {
  int digits = 0;
  for (size_t i = s.start; i < s.end; ++i) {
    if (data[i] >= '0' && data[i] <= '9') {
      ++digits;
    }
  }
  return digits >= 4;
}

bool is_heading_like(const char* data, Span s) {
  size_t i = s.start;
  while (i < s.end && (data[i] == ' ' || data[i] == '\t')) {
    ++i;
  }
  if (i < s.end && data[i] == '#') {
    return true;
  }

  int alpha = 0;
  int upper = 0;
  for (size_t j = s.start; j < s.end; ++j) {
    unsigned char c = static_cast<unsigned char>(data[j]);
    if (c >= 'A' && c <= 'Z') {
      ++alpha;
      ++upper;
    } else if (c >= 'a' && c <= 'z') {
      ++alpha;
    }
  }
  return alpha >= 4 && static_cast<double>(upper) / static_cast<double>(alpha) >= 0.8;
}

bool is_anchor(const char* data, Span s) {
  return contains_subspan(data, s, "```") || contains_subspan(data, s, "http://") ||
         contains_subspan(data, s, "https://") || is_numeric_heavy(data, s) || is_heading_like(data, s);
}

void tokenize_sentence(const char* data, Span s, std::unordered_map<std::string, int>* out_tf) {
  static const std::unordered_set<std::string> kStopwords = {
      "a",   "an",   "and", "are", "as",   "at",  "be",  "by", "for", "from", "has",
      "he",  "in",   "is",  "it",  "its",  "of",  "on",  "or", "that", "the", "to",
      "was", "were", "will", "with", "this", "they", "we", "you", "i",   "but"};

  std::string token;
  for (size_t i = s.start; i < s.end; ++i) {
    unsigned char c = static_cast<unsigned char>(data[i]);
    if (is_ascii_alnum(c)) {
      token.push_back(ascii_lower(c));
    } else {
      if (!token.empty() && kStopwords.find(token) == kStopwords.end()) {
        ++(*out_tf)[token];
      }
      token.clear();
    }
  }
  if (!token.empty() && kStopwords.find(token) == kStopwords.end()) {
    ++(*out_tf)[token];
  }
}

std::vector<std::pair<std::string, int>> top_k_tokens(const std::unordered_map<std::string, int>& tf, size_t k) {
  std::vector<std::pair<std::string, int>> tokens(tf.begin(), tf.end());
  std::sort(tokens.begin(), tokens.end(), [](const auto& a, const auto& b) {
    if (a.second != b.second) {
      return a.second > b.second;
    }
    return a.first < b.first;
  });
  if (tokens.size() > k) {
    tokens.resize(k);
  }
  return tokens;
}

double cosine_similarity(const std::unordered_map<std::string, int>& a,
                         const std::unordered_map<std::string, int>& b) {
  if (a.empty() || b.empty()) {
    return 0.0;
  }
  double dot = 0.0;
  double na = 0.0;
  double nb = 0.0;
  for (const auto& it : a) {
    na += static_cast<double>(it.second) * static_cast<double>(it.second);
    auto jt = b.find(it.first);
    if (jt != b.end()) {
      dot += static_cast<double>(it.second) * static_cast<double>(jt->second);
    }
  }
  for (const auto& it : b) {
    nb += static_cast<double>(it.second) * static_cast<double>(it.second);
  }
  if (na == 0.0 || nb == 0.0) {
    return 0.0;
  }
  return dot / (std::sqrt(na) * std::sqrt(nb));
}

size_t aggr_drop_percent(int aggr) {
  static const size_t map[10] = {0, 5, 10, 15, 20, 25, 30, 35, 40, 45};
  if (aggr < 0) {
    return 0;
  }
  if (aggr > 9) {
    return map[9];
  }
  return map[aggr];
}

double dup_threshold(int aggr) {
  if (aggr <= 3) {
    return 0.95;
  }
  if (aggr <= 6) {
    return 0.90;
  }
  return 0.85;
}

std::vector<char> compress_impl(const char* data, size_t len, int aggressiveness) {
  if (aggressiveness <= 0 || len == 0) {
    return std::vector<char>(data, data + len);
  }

  std::vector<unsigned char> keep_mask(len, 1);

  auto pspans = paragraph_spans(data, len);
  std::unordered_map<uint64_t, int> seen_blocks;
  for (const auto& ps : pspans) {
    size_t plen = ps.end - ps.start;
    if (plen >= kMinBlockLen) {
      const uint64_t h = fnv1a_64(data + ps.start, plen);
      int seen = ++seen_blocks[h];
      if (seen >= 2) {
        mark_span(&keep_mask, ps);
        continue;
      }
      if (aggressiveness >= 8 && unique_ratio(data, ps) < 0.03) {
        mark_span(&keep_mask, ps);
      }
    }
  }

  auto spans = segment_sentences(data, len);
  std::vector<SentenceData> sentences;
  sentences.reserve(spans.size());
  for (const auto& s : spans) {
    SentenceData sd;
    sd.span = s;
    sd.anchor = is_anchor(data, s);
    sd.removed = false;
    tokenize_sentence(data, s, &sd.tf);
    sd.top_tokens = top_k_tokens(sd.tf, 3);
    sentences.push_back(std::move(sd));
  }

  std::unordered_map<std::string, std::vector<size_t>> buckets;
  const double threshold = dup_threshold(aggressiveness);
  for (size_t i = 0; i < sentences.size(); ++i) {
    auto& s = sentences[i];
    if (s.anchor) {
      continue;
    }
    if (s.tf.empty()) {
      continue;
    }
    std::string key = std::to_string((s.span.end - s.span.start) / 20);
    for (const auto& tk : s.top_tokens) {
      key.push_back('|');
      key += tk.first;
    }

    bool duplicate = false;
    auto it = buckets.find(key);
    if (it != buckets.end()) {
      for (size_t prior_idx : it->second) {
        if (sentences[prior_idx].removed) {
          continue;
        }
        const double sim = cosine_similarity(s.tf, sentences[prior_idx].tf);
        if (sim >= threshold) {
          duplicate = true;
          break;
        }
      }
    }

    if (duplicate) {
      s.removed = true;
      mark_span(&keep_mask, s.span);
    } else {
      buckets[key].push_back(i);
    }
  }

  std::unordered_map<std::string, int> df;
  size_t active_n = 0;
  for (const auto& s : sentences) {
    if (s.removed) {
      continue;
    }
    ++active_n;
    for (const auto& kv : s.tf) {
      ++df[kv.first];
    }
  }

  std::vector<std::pair<double, size_t>> removable;
  removable.reserve(sentences.size());
  for (size_t i = 0; i < sentences.size(); ++i) {
    auto& s = sentences[i];
    if (s.removed) {
      continue;
    }
    double score = 0.0;
    bool has_rare = false;
    for (const auto& kv : s.tf) {
      const int d = df[kv.first];
      const double idf = std::log(1.0 + static_cast<double>(active_n) / (1.0 + static_cast<double>(d)));
      score += static_cast<double>(kv.second) * idf;
      if (idf > 1.2) {
        has_rare = true;
      }
    }
    if ((s.span.end - s.span.start) < 25 && !has_rare) {
      score *= 0.4;
    }
    s.score = score;
    if (!s.anchor) {
      removable.push_back({score, i});
    }
  }

  const size_t drop_target = removable.empty() ? 0 : (removable.size() * aggr_drop_percent(aggressiveness)) / 100;
  std::sort(removable.begin(), removable.end(), [](const auto& a, const auto& b) {
    if (a.first != b.first) {
      return a.first < b.first;
    }
    return a.second < b.second;
  });

  for (size_t i = 0; i < drop_target && i < removable.size(); ++i) {
    size_t idx = removable[i].second;
    sentences[idx].removed = true;
    mark_span(&keep_mask, sentences[idx].span);
  }

  std::vector<char> out;
  out.reserve(len);
  for (size_t i = 0; i < len; ++i) {
    if (keep_mask[i]) {
      out.push_back(data[i]);
    }
  }
  return out;
}

}  // namespace

int csq_squeeze(csq_view in, csq_buf* out) { return csq_squeeze_ex(in, 0, out); }

int csq_squeeze_ex(csq_view in, int aggressiveness, csq_buf* out) {
  if (out == nullptr) {
    return 1;
  }
  out->data = nullptr;
  out->len = 0;

  try {
    if (aggressiveness < 0) {
      aggressiveness = 0;
    }
    if (aggressiveness > 9) {
      aggressiveness = 9;
    }

    if (in.len == 0) {
      return 0;
    }

    std::vector<char> result = compress_impl(in.data, in.len, aggressiveness);
    if (result.empty()) {
      return 0;
    }

    char* data_out = static_cast<char*>(std::malloc(result.size()));
    if (data_out == nullptr) {
      return 2;
    }
    std::memcpy(data_out, result.data(), result.size());
    out->data = data_out;
    out->len = result.size();
    return 0;
  } catch (...) {
    return 3;
  }
}

void csq_free(csq_buf* buf) {
  if (buf == nullptr) {
    return;
  }
  if (buf->data != nullptr) {
    std::free(buf->data);
    buf->data = nullptr;
  }
  buf->len = 0;
}

const char* csq_version(void) { return kVersion; }
