#include "contextsqueeze.h"

#include <algorithm>
#include <cctype>
#include <cmath>
#include <cstdint>
#include <cstdlib>
#include <cstring>
#include <string>
#include <unordered_map>
#include <unordered_set>
#include <vector>

namespace {

struct Span {
  size_t start;
  size_t end;
};

static bool is_ascii_alnum(unsigned char c) {
  return (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9');
}

static unsigned char ascii_lower(unsigned char c) {
  if (c >= 'A' && c <= 'Z') {
    return static_cast<unsigned char>(c + ('a' - 'A'));
  }
  return c;
}

static std::vector<Span> split_blocks(const char* data, size_t len) {
  std::vector<Span> blocks;
  size_t start = 0;
  size_t i = 0;
  while (i + 1 < len) {
    if (data[i] == '\n' && data[i + 1] == '\n') {
      blocks.push_back({start, i});
      i += 2;
      start = i;
      continue;
    }
    i++;
  }
  if (start <= len) {
    blocks.push_back({start, len});
  }
  return blocks;
}

static uint64_t fnv1a64(const char* data, size_t len) {
  uint64_t h = 1469598103934665603ULL;
  for (size_t i = 0; i < len; ++i) {
    h ^= static_cast<unsigned char>(data[i]);
    h *= 1099511628211ULL;
  }
  return h;
}

static bool is_abbrev_before(const char* data, size_t dot_pos) {
  static const char* abbrs[] = {"e.g.", "i.e.", "mr.", "dr.", "vs.", "etc."};
  for (const char* ab : abbrs) {
    size_t n = std::strlen(ab);
    if (dot_pos + 1 >= n) {
      size_t start = dot_pos + 1 - n;
      bool ok = true;
      for (size_t i = 0; i < n; ++i) {
        if (ascii_lower(static_cast<unsigned char>(data[start + i])) != static_cast<unsigned char>(ab[i])) {
          ok = false;
          break;
        }
      }
      if (ok) {
        return true;
      }
    }
  }
  return false;
}

static std::vector<Span> segment_sentences(const char* data, size_t len) {
  std::vector<Span> spans;
  size_t start = 0;
  for (size_t i = 0; i < len; ++i) {
    bool split = false;
    char c = data[i];
    if ((c == '.' || c == '?' || c == '!') && !is_abbrev_before(data, i)) {
      split = true;
    }
    if (i + 1 < len && data[i] == '\n' && data[i + 1] == '\n') {
      split = true;
      i += 1;
    }
    if (split) {
      size_t end = i + 1;
      if (end > start) {
        spans.push_back({start, end});
      }
      start = end;
    }
  }
  if (start < len) {
    spans.push_back({start, len});
  }
  return spans;
}

static std::vector<std::string> tokenize_sentence(const char* data, size_t start, size_t end) {
  std::vector<std::string> out;
  size_t i = start;
  while (i < end) {
    while (i < end) {
      unsigned char c = ascii_lower(static_cast<unsigned char>(data[i]));
      if (is_ascii_alnum(c)) {
        break;
      }
      i++;
    }
    size_t j = i;
    while (j < end) {
      unsigned char c = ascii_lower(static_cast<unsigned char>(data[j]));
      if (!is_ascii_alnum(c)) {
        break;
      }
      j++;
    }
    if (j > i) {
      out.emplace_back(data + i, data + j);
      for (char& ch : out.back()) {
        ch = static_cast<char>(ascii_lower(static_cast<unsigned char>(ch)));
      }
    }
    i = j;
  }
  return out;
}

static bool is_stopword(const std::string& s) {
  static const std::unordered_set<std::string> stopwords = {
      "the", "a",   "an",  "and", "or",   "to",  "of", "in",  "on",  "for", "with", "is",
      "are", "was", "were", "be",  "as",   "at",  "by", "it",  "that", "this", "from", "but",
      "we",  "you", "they", "he",  "she",  "i",   "not", "can", "will", "if",   "then", "than"};
  return stopwords.find(s) != stopwords.end();
}

static double cosine_tf(const std::unordered_map<std::string, int>& a,
                        const std::unordered_map<std::string, int>& b,
                        double norm_a,
                        double norm_b) {
  if (norm_a == 0.0 || norm_b == 0.0) {
    return 0.0;
  }
  const auto* small = &a;
  const auto* large = &b;
  if (a.size() > b.size()) {
    small = &b;
    large = &a;
  }
  double dot = 0.0;
  for (const auto& kv : *small) {
    auto it = large->find(kv.first);
    if (it != large->end()) {
      dot += static_cast<double>(kv.second) * static_cast<double>(it->second);
    }
  }
  return dot / (norm_a * norm_b);
}

static bool is_anchor(const char* data, Span s) {
  std::string text(data + s.start, data + s.end);
  if (text.find("```") != std::string::npos) {
    return true;
  }
  if (text.find("http://") != std::string::npos || text.find("https://") != std::string::npos) {
    return true;
  }
  int digits = 0;
  int letters = 0;
  int upper = 0;
  size_t i = 0;
  while (i < text.size() && std::isspace(static_cast<unsigned char>(text[i])) != 0) {
    i++;
  }
  if (i < text.size() && text[i] == '#') {
    return true;
  }
  for (char ch : text) {
    unsigned char uc = static_cast<unsigned char>(ch);
    if (std::isdigit(uc) != 0) {
      digits++;
    }
    if (std::isalpha(uc) != 0) {
      letters++;
      if (std::isupper(uc) != 0) {
        upper++;
      }
    }
  }
  if (digits >= 4) {
    return true;
  }
  if (letters > 0 && static_cast<double>(upper) / static_cast<double>(letters) >= 0.7) {
    return true;
  }
  return false;
}

static double drop_ratio_for_aggr(int aggr) {
  static const double table[] = {0.00, 0.05, 0.10, 0.15, 0.20, 0.25, 0.30, 0.35, 0.40, 0.45};
  if (aggr < 0) {
    aggr = 0;
  }
  if (aggr > 9) {
    aggr = 9;
  }
  return table[aggr];
}

static double dup_threshold_for_aggr(int aggr) {
  if (aggr <= 3) {
    return 0.95;
  }
  if (aggr <= 6) {
    return 0.90;
  }
  return 0.85;
}

}  // namespace

int csq_squeeze(csq_view in, csq_buf* out) { return csq_squeeze_ex(in, 0, out); }

int csq_squeeze_ex(csq_view in, int aggressiveness, csq_buf* out) {
  if (out == nullptr) {
    return 2;
  }
  out->data = nullptr;
  out->len = 0;
  if (in.len == 0) {
    return 0;
  }
  if (in.data == nullptr) {
    return 3;
  }
  try {
    if (aggressiveness <= 0) {
      char* mem = static_cast<char*>(std::malloc(in.len));
      if (mem == nullptr) {
        return 1;
      }
      std::memcpy(mem, in.data, in.len);
      out->data = mem;
      out->len = in.len;
      return 0;
    }

    std::vector<Span> blocks = split_blocks(in.data, in.len);
    std::vector<bool> remove_block(blocks.size(), false);
    std::unordered_map<uint64_t, size_t> seen_block;
    const size_t min_block_len = 120;
    for (size_t i = 0; i < blocks.size(); ++i) {
      size_t len = blocks[i].end - blocks[i].start;
      if (len == 0) {
        continue;
      }
      uint64_t h = fnv1a64(in.data + blocks[i].start, len);
      auto it = seen_block.find(h);
      if (len >= min_block_len) {
        if (it == seen_block.end()) {
          seen_block[h] = i;
        } else {
          remove_block[i] = true;
        }
      }
      if (len >= min_block_len) {
        bool uniq[256] = {false};
        int uniq_count = 0;
        for (size_t j = blocks[i].start; j < blocks[i].end; ++j) {
          unsigned char c = static_cast<unsigned char>(in.data[j]);
          if (!uniq[c]) {
            uniq[c] = true;
            uniq_count++;
          }
        }
        double ratio = static_cast<double>(uniq_count) / static_cast<double>(len);
        if (ratio < 0.12) {
          remove_block[i] = true;
        }
      }
    }

    std::vector<Span> sentences = segment_sentences(in.data, in.len);
    size_t n = sentences.size();
    std::vector<bool> keep(n, true);
    std::vector<bool> anchor(n, false);

    for (size_t i = 0; i < n; ++i) {
      for (size_t b = 0; b < blocks.size(); ++b) {
        if (!remove_block[b]) {
          continue;
        }
        if (sentences[i].start >= blocks[b].start && sentences[i].end <= blocks[b].end) {
          keep[i] = false;
          break;
        }
      }
      anchor[i] = is_anchor(in.data, sentences[i]);
    }

    std::vector<std::unordered_map<std::string, int>> tf(n);
    std::vector<double> norms(n, 0.0);
    std::unordered_map<std::string, int> df;

    for (size_t i = 0; i < n; ++i) {
      if (!keep[i]) {
        continue;
      }
      std::unordered_set<std::string> seen_tokens;
      auto tokens = tokenize_sentence(in.data, sentences[i].start, sentences[i].end);
      for (const auto& t : tokens) {
        if (is_stopword(t)) {
          continue;
        }
        tf[i][t]++;
        seen_tokens.insert(t);
      }
      for (const auto& t : seen_tokens) {
        df[t]++;
      }
      double sumsq = 0.0;
      for (const auto& kv : tf[i]) {
        sumsq += static_cast<double>(kv.second * kv.second);
      }
      norms[i] = std::sqrt(sumsq);
    }

    std::unordered_map<uint64_t, std::vector<size_t>> buckets;
    double dup_threshold = dup_threshold_for_aggr(aggressiveness);
    for (size_t i = 0; i < n; ++i) {
      if (!keep[i]) {
        continue;
      }
      std::vector<std::pair<std::string, int>> items(tf[i].begin(), tf[i].end());
      std::sort(items.begin(), items.end(), [](const auto& a, const auto& b) {
        if (a.second != b.second) {
          return a.second > b.second;
        }
        return a.first < b.first;
      });
      uint64_t sig = static_cast<uint64_t>((sentences[i].end - sentences[i].start) / 16);
      size_t top_k = std::min<size_t>(items.size(), 3);
      for (size_t k = 0; k < top_k; ++k) {
        sig ^= fnv1a64(items[k].first.data(), items[k].first.size());
      }
      auto& cand = buckets[sig];
      bool dup = false;
      for (size_t prev : cand) {
        double c = cosine_tf(tf[i], tf[prev], norms[i], norms[prev]);
        if (c >= dup_threshold) {
          dup = true;
          break;
        }
      }
      if (dup && !anchor[i]) {
        keep[i] = false;
      } else {
        cand.push_back(i);
      }
    }

    struct ScoreItem {
      size_t idx;
      double score;
    };
    std::vector<ScoreItem> scores;
    for (size_t i = 0; i < n; ++i) {
      if (!keep[i] || anchor[i]) {
        continue;
      }
      double score = 0.0;
      for (const auto& kv : tf[i]) {
        double idf = std::log(1.0 + static_cast<double>(n) / (1.0 + static_cast<double>(df[kv.first])));
        score += static_cast<double>(kv.second) * idf;
      }
      size_t slen = sentences[i].end - sentences[i].start;
      if (slen < 25) {
        bool has_rare = false;
        for (const auto& kv : tf[i]) {
          if (df[kv.first] <= 1) {
            has_rare = true;
            break;
          }
        }
        if (!has_rare) {
          score *= 0.4;
        }
      }
      scores.push_back({i, score});
    }

    size_t keep_count = 0;
    for (bool k : keep) {
      if (k) {
        keep_count++;
      }
    }
    size_t target_drop = static_cast<size_t>(std::floor(drop_ratio_for_aggr(aggressiveness) * static_cast<double>(keep_count)));
    std::sort(scores.begin(), scores.end(), [](const ScoreItem& a, const ScoreItem& b) {
      if (a.score != b.score) {
        return a.score < b.score;
      }
      return a.idx < b.idx;
    });
    for (size_t i = 0; i < target_drop && i < scores.size(); ++i) {
      keep[scores[i].idx] = false;
    }

    std::vector<Span> kept;
    for (size_t i = 0; i < n; ++i) {
      if (keep[i]) {
        kept.push_back(sentences[i]);
      }
    }
    std::sort(kept.begin(), kept.end(), [](Span a, Span b) { return a.start < b.start; });

    std::string out_s;
    out_s.reserve(in.len);
    size_t prev_end = 0;
    for (const auto& s : kept) {
      if (s.end <= s.start || s.start >= in.len) {
        continue;
      }
      size_t start = s.start;
      size_t end = std::min(s.end, in.len);
      if (start < prev_end) {
        start = prev_end;
      }
      if (end > start) {
        out_s.append(in.data + start, in.data + end);
        prev_end = end;
      }
    }

    char* mem = nullptr;
    if (!out_s.empty()) {
      mem = static_cast<char*>(std::malloc(out_s.size()));
      if (mem == nullptr) {
        return 1;
      }
      std::memcpy(mem, out_s.data(), out_s.size());
    }
    out->data = mem;
    out->len = out_s.size();
    return 0;
  } catch (...) {
    return 9;
  }
}

void csq_free(csq_buf* buf) {
  if (buf == nullptr) {
    return;
  }
  if (buf->data != nullptr) {
    std::free(buf->data);
  }
  buf->data = nullptr;
  buf->len = 0;
}

const char* csq_version(void) { return "0.2.0"; }
