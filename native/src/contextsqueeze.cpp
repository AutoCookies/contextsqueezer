#include "contextsqueeze.h"
#include "metrics.h"

#include <algorithm>
#include <array>
#include <cmath>
#include <cstddef>
#include <cstdint>
#include <cstdlib>
#include <cstring>
#include <string>
#include <string_view>
#include <unordered_map>
#include <unordered_set>
#include <utility>
#include <vector>

namespace {

constexpr const char* kVersion = "1.0.0";

struct Span {
  size_t start;
  size_t end;
};

struct SentenceInfo {
  Span span;
  std::unordered_map<std::string, int> tf;
  std::vector<std::string> uniq_tokens;
  bool anchor{false};
  double score{0.0};
  bool drop{false};
};

bool is_ascii_alpha_num(unsigned char c) {
  return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9');
}

unsigned char lower_ascii(unsigned char c) {
  if (c >= 'A' && c <= 'Z') return static_cast<unsigned char>(c + ('a' - 'A'));
  return c;
}

uint64_t fnv1a(std::string_view s) {
  uint64_t h = 1469598103934665603ULL;
  for (unsigned char c : s) {
    h ^= c;
    h *= 1099511628211ULL;
  }
  return h;
}

std::string trim_ascii(std::string_view s) {
  size_t b = 0;
  while (b < s.size() && (s[b] == ' ' || s[b] == '\t' || s[b] == '\n' || s[b] == '\r')) ++b;
  size_t e = s.size();
  while (e > b && (s[e - 1] == ' ' || s[e - 1] == '\t' || s[e - 1] == '\n' || s[e - 1] == '\r')) --e;
  return std::string(s.substr(b, e - b));
}

std::unordered_set<std::string> stopwords() {
  return {"the", "and", "or", "a", "an", "is", "are", "to", "of", "in", "for", "on", "with",
          "as", "at", "by", "be", "this", "that", "it", "from", "was", "were", "will", "can", "if"};
}

bool has_double_newline(const std::string& s, size_t i) {
  return i + 1 < s.size() && s[i] == '\n' && s[i + 1] == '\n';
}

bool is_url_token(std::string_view t) {
  return t.find("http://") != std::string_view::npos || t.find("https://") != std::string_view::npos;
}

bool is_abbrev_before(const std::string& s, size_t punct_idx) {
  static const std::unordered_set<std::string> kAbbrev = {"e.g.", "i.e.", "mr.", "dr.", "vs.", "etc.", "ms.", "mrs.", "prof."};
  if (s[punct_idx] != '.') return false;
  size_t end = punct_idx + 1;
  size_t start = punct_idx;
  while (start > 0 && ((s[start - 1] >= 'a' && s[start - 1] <= 'z') || (s[start - 1] >= 'A' && s[start - 1] <= 'Z'))) --start;
  if (end - start < 2 || end - start > 6) return false;
  std::string t;
  for (size_t i = start; i < end; ++i) t.push_back(static_cast<char>(lower_ascii(static_cast<unsigned char>(s[i]))));
  return kAbbrev.find(t) != kAbbrev.end();
}

std::vector<Span> segment_sentences(const std::string& s) {
  std::vector<Span> spans;
  if (s.empty()) return spans;
  size_t start = 0;
  for (size_t i = 0; i < s.size(); ++i) {
    if (has_double_newline(s, i)) {
      if (i > start) spans.push_back({start, i});
      spans.push_back({i, i + 2});
      start = i + 2;
      ++i;
      continue;
    }

    char c = s[i];
    if ((c == '.' || c == '?' || c == '!') && !is_abbrev_before(s, i)) {
      if (c == '.') {
        size_t tstart = i;
        while (tstart > start && s[tstart - 1] != ' ' && s[tstart - 1] != '\n' && s[tstart - 1] != '\t' && s[tstart - 1] != '\r') --tstart;
        std::string_view tok(s.data() + tstart, i - tstart + 1);
        if (is_url_token(tok)) continue;
      }
      size_t end = i + 1;
      while (end < s.size() && (s[end] == ' ' || s[end] == '\t' || s[end] == '\r' || (s[end] == '\n' && !has_double_newline(s, end)))) ++end;
      spans.push_back({start, end});
      start = end;
      i = end > 0 ? end - 1 : end;
    }
  }
  if (start < s.size()) spans.push_back({start, s.size()});
  return spans;
}

bool is_anchor(std::string_view sv) {
  if (sv.find("```") != std::string_view::npos) return true;
  if (sv.find("http://") != std::string_view::npos || sv.find("https://") != std::string_view::npos) return true;
  int digits = 0;
  for (unsigned char c : sv) if (c >= '0' && c <= '9') ++digits;
  if (digits >= 4) return true;

  std::string trimmed = trim_ascii(sv);
  if (!trimmed.empty() && trimmed[0] == '#') return true;

  int alpha_words = 0;
  int all_caps = 0;
  size_t i = 0;
  while (i < trimmed.size()) {
    while (i < trimmed.size() && !is_ascii_alpha_num(static_cast<unsigned char>(trimmed[i]))) ++i;
    size_t j = i;
    int alpha = 0;
    int caps = 0;
    while (j < trimmed.size() && is_ascii_alpha_num(static_cast<unsigned char>(trimmed[j]))) {
      if (trimmed[j] >= 'A' && trimmed[j] <= 'Z') {
        ++alpha;
        ++caps;
      } else if (trimmed[j] >= 'a' && trimmed[j] <= 'z') {
        ++alpha;
      }
      ++j;
    }
    if (alpha > 1) {
      ++alpha_words;
      if (alpha == caps) ++all_caps;
    }
    i = j;
  }
  return alpha_words > 0 && (static_cast<double>(all_caps) / static_cast<double>(alpha_words)) >= 0.6;
}

std::vector<std::string> tokenize(std::string_view sv, const std::unordered_set<std::string>& sw) {
  std::vector<std::string> out;
  std::string cur;
  for (unsigned char c : sv) {
    if (is_ascii_alpha_num(c)) {
      cur.push_back(static_cast<char>(lower_ascii(c)));
    } else if (!cur.empty()) {
      if (sw.find(cur) == sw.end()) out.push_back(cur);
      cur.clear();
    }
  }
  if (!cur.empty() && sw.find(cur) == sw.end()) out.push_back(cur);
  return out;
}

double cosine_tf(const std::unordered_map<std::string, int>& a, const std::unordered_map<std::string, int>& b) {
  if (a.empty() || b.empty()) return 0.0;
  double dot = 0.0;
  for (const auto& kv : a) {
    auto it = b.find(kv.first);
    if (it != b.end()) dot += static_cast<double>(kv.second * it->second);
  }
  double na = 0.0;
  for (const auto& kv : a) na += static_cast<double>(kv.second * kv.second);
  double nb = 0.0;
  for (const auto& kv : b) nb += static_cast<double>(kv.second * kv.second);
  if (na == 0.0 || nb == 0.0) return 0.0;
  return dot / (std::sqrt(na) * std::sqrt(nb));
}

double dup_threshold(int aggr) {
  if (aggr <= 3) return 0.95;
  if (aggr <= 6) return 0.90;
  return 0.85;
}

double drop_ratio(int aggr) {
  static const std::array<double, 10> k = {0.0, 0.05, 0.10, 0.15, 0.20, 0.25, 0.30, 0.35, 0.40, 0.45};
  if (aggr < 0) return 0.0;
  if (aggr > 9) aggr = 9;
  return k[static_cast<size_t>(aggr)];
}

std::string squeeze_impl(std::string input, int aggr) {
  if (aggr <= 0 || input.empty()) return input;

  std::vector<Span> blocks;
  size_t pstart = 0;
  for (size_t i = 0; i < input.size();) {
    if (has_double_newline(input, i)) {
      blocks.push_back({pstart, i});
      blocks.push_back({i, i + 2});
      i += 2;
      pstart = i;
    } else {
      ++i;
    }
  }
  if (pstart <= input.size()) blocks.push_back({pstart, input.size()});

  std::vector<bool> block_drop(blocks.size(), false);
  std::unordered_map<uint64_t, size_t> first_seen;
  for (size_t i = 0; i < blocks.size(); ++i) {
    const Span& b = blocks[i];
    if (b.end <= b.start) continue;
    std::string_view sv(input.data() + b.start, b.end - b.start);
    if (sv == "\n\n") continue;
    if (sv.size() >= 120) {
      uint64_t h = fnv1a(sv);
      if (first_seen.find(h) == first_seen.end()) {
        first_seen[h] = i;
      } else {
        block_drop[i] = true;
      }
    }

    if (sv.size() >= 300) {
      std::array<bool, 256> seen{};
      size_t uniq = 0;
      for (unsigned char c : sv) {
        if (!seen[c]) {
          seen[c] = true;
          ++uniq;
        }
      }
      if (static_cast<double>(uniq) / static_cast<double>(sv.size()) < 0.08) block_drop[i] = true;
    }
  }

  std::string filtered;
  filtered.reserve(input.size());
  for (size_t i = 0; i < blocks.size(); ++i) {
    if (!block_drop[i]) {
      const Span& b = blocks[i];
      filtered.append(input.data() + b.start, b.end - b.start);
    }
  }

  auto spans = segment_sentences(filtered);
  if (spans.empty()) return filtered;

  const auto sw = stopwords();
  std::vector<SentenceInfo> sentences;
  csq_metrics_add_sentences(static_cast<uint64_t>(spans.size()));
  for (const auto& sp : spans) {
    std::string_view sv(filtered.data() + sp.start, sp.end - sp.start);
    SentenceInfo info;
    info.span = sp;
    info.anchor = is_anchor(sv);
    auto tokens = tokenize(sv, sw);
    csq_metrics_add_tokens(static_cast<uint64_t>(tokens.size()));
    for (const auto& t : tokens) info.tf[t] += 1;
    for (const auto& kv : info.tf) info.uniq_tokens.push_back(kv.first);
    std::sort(info.uniq_tokens.begin(), info.uniq_tokens.end());
    sentences.push_back(std::move(info));
  }

  auto token_signature = [](const SentenceInfo& s) {
    std::vector<std::string> v;
    for (const auto& t : s.uniq_tokens) v.push_back(t.substr(0, std::min<size_t>(4, t.size())));
    std::sort(v.begin(), v.end());
    std::string sig;
    for (size_t i = 0; i < v.size() && i < 3; ++i) {
      if (i > 0) sig += "|";
      sig += v[i];
    }
    return sig;
  };

  std::unordered_map<std::string, std::vector<size_t>> buckets;
  for (size_t i = 0; i < sentences.size(); ++i) {
    if (sentences[i].anchor) continue;
    std::string key = std::to_string((sentences[i].span.end - sentences[i].span.start) / 20) + "|" + token_signature(sentences[i]);
    auto& cand = buckets[key];
    bool dup = false;
    size_t begin = cand.size() > 64 ? cand.size() - 64 : 0;
    size_t checked = cand.size() - begin;
    csq_metrics_add_candidates(static_cast<uint64_t>(checked));
    for (size_t j = begin; j < cand.size(); ++j) {
      size_t prev = cand[j];
      csq_metrics_add_pairs(1);
      if (cosine_tf(sentences[prev].tf, sentences[i].tf) >= dup_threshold(aggr)) {
        dup = true;
        break;
      }
    }
    if (dup) {
      sentences[i].drop = true;
    } else {
      cand.push_back(i);
    }
  }

  std::unordered_map<std::string, int> df;
  int n = 0;
  for (const auto& s : sentences) {
    if (s.drop) continue;
    ++n;
    for (const auto& kv : s.tf) df[kv.first] += 1;
  }

  for (auto& s : sentences) {
    if (s.drop) continue;
    for (const auto& kv : s.tf) {
      double idf = std::log(1.0 + static_cast<double>(n) / (1.0 + static_cast<double>(df[kv.first])));
      s.score += static_cast<double>(kv.second) * idf;
    }
    size_t slen = s.span.end - s.span.start;
    if (slen < 25) {
      bool rare = false;
      for (const auto& kv : s.tf) {
        double idf = std::log(1.0 + static_cast<double>(n) / (1.0 + static_cast<double>(df[kv.first])));
        if (idf > 1.5) {
          rare = true;
          break;
        }
      }
      if (!rare) s.score *= 0.3;
    }
  }

  std::vector<std::pair<double, size_t>> candidates;
  for (size_t i = 0; i < sentences.size(); ++i) {
    if (!sentences[i].drop && !sentences[i].anchor) candidates.push_back({sentences[i].score, i});
  }

  size_t to_drop = static_cast<size_t>(std::floor(drop_ratio(aggr) * static_cast<double>(sentences.size())));
  if (to_drop > candidates.size()) to_drop = candidates.size();
  std::stable_sort(candidates.begin(), candidates.end(), [](const auto& a, const auto& b) {
    if (a.first == b.first) return a.second < b.second;
    return a.first < b.first;
  });
  for (size_t i = 0; i < to_drop; ++i) sentences[candidates[i].second].drop = true;

  std::string out;
  out.reserve(filtered.size());
  for (const auto& s : sentences) {
    if (!s.drop) out.append(filtered.data() + s.span.start, s.span.end - s.span.start);
  }
  return out;
}

int copy_to_cbuf(const std::string& s, csq_buf* out) {
  out->data = nullptr;
  out->len = 0;
  if (s.empty()) return 0;
  void* raw = std::malloc(s.size());
  if (raw == nullptr) return 2;
  std::memcpy(raw, s.data(), s.size());
  out->data = static_cast<char*>(raw);
  out->len = s.size();
  return 0;
}

}  // namespace

extern "C" int csq_squeeze(csq_view in, csq_buf* out) { return csq_squeeze_ex(in, 0, out); }

extern "C" int csq_squeeze_ex(csq_view in, int aggressiveness, csq_buf* out) {
  if (out == nullptr) return 1;
  out->data = nullptr;
  out->len = 0;
  if (in.len == 0) return 0;
  if (in.data == nullptr) return 4;

  try {
    csq_metrics_reset();
    if (aggressiveness < 0) aggressiveness = 0;
    if (aggressiveness > 9) aggressiveness = 9;
    std::string input(in.data, in.len);
    return copy_to_cbuf(squeeze_impl(std::move(input), aggressiveness), out);
  } catch (...) {
    return 3;
  }
}

extern "C" void csq_free(csq_buf* buf) {
  if (buf == nullptr) return;
  std::free(buf->data);
  buf->data = nullptr;
  buf->len = 0;
}

extern "C" const char* csq_version(void) { return kVersion; }
