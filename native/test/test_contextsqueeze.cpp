#include "contextsqueeze.h"

#include <chrono>
#include <cstring>
#include <iostream>
#include <string>
#include <vector>

namespace {

bool check_equal(const std::vector<char>& in, const csq_buf& out) {
  if (in.size() != out.len) return false;
  if (in.empty()) return true;
  return std::memcmp(in.data(), out.data, out.len) == 0;
}

std::string squeeze_ex(const std::string& in, int aggr) {
  csq_view view{in.data(), in.size()};
  csq_buf out{nullptr, 0};
  if (csq_squeeze_ex(view, aggr, &out) != 0) return "";
  std::string result(out.data, out.len);
  csq_free(&out);
  return result;
}

int test_version_non_empty() {
  const char* v = csq_version();
  return (v != nullptr && std::strlen(v) > 0) ? 0 : 1;
}

int test_squeeze_identity_binary() {
  const std::vector<char> in = {'a', '\0', 'b', '\n', '\0', 'z'};
  csq_view view{in.data(), in.size()};
  csq_buf out{nullptr, 0};
  if (csq_squeeze(view, &out) != 0) return 1;
  bool ok = check_equal(in, out);
  csq_free(&out);
  return ok ? 0 : 1;
}

int test_segmentation_abbrev_and_newlines() {
  const std::string in = "Dr. A met Mr. B.\nStill same paragraph.\n\nNew section starts here! i.e. keep sentence.";
  const std::string out = squeeze_ex(in, 6);
  if (out.find("Dr. A met Mr. B.") == std::string::npos) return 1;
  if (out.find("New section starts here!") == std::string::npos) return 1;
  return 0;
}

int test_boilerplate_preserve_first_drop_repeats() {
  const std::string blk = "DISCLAIMER: This text is repeated boilerplate for legal reasons and should be removed on repeats. "
                          "DISCLAIMER: This text is repeated boilerplate for legal reasons and should be removed on repeats.\n";
  const std::string in = blk + "\n" + "Unique content here.\n\n" + blk;
  const std::string out = squeeze_ex(in, 7);
  size_t first = out.find("DISCLAIMER");
  if (first == std::string::npos) return 1;
  size_t second = out.find("DISCLAIMER", first + 1);
  if (second != std::string::npos) return 1;
  return 0;
}

int test_duplicate_removal() {
  const std::string in =
      "The cache layer reduces latency for requests. "
      "The cache layer reduces latency for requests! "
      "Caching reduces latency for requests in services. "
      "Independent sentence remains.";
  const std::string out = squeeze_ex(in, 1);
  if (out.find("Independent sentence remains.") == std::string::npos) return 1;
  size_t first = out.find("The cache layer reduces latency for requests.");
  if (first == std::string::npos) return 1;
  size_t second = out.find("The cache layer reduces latency for requests.", first + 1);
  if (second != std::string::npos) return 1;
  return 0;
}

int test_pruning_respects_anchors() {
  const std::string in =
      "# HEADER TITLE\n"
      "short fluff. short fluff. short fluff.\n"
      "Visit https://example.com/docs for details.\n"
      "```\ncode block\n```\n"
      "Build 20240101 release 1234 metrics.\n"
      "tiny note. tiny note. tiny note. tiny note.\n";
  const std::string out = squeeze_ex(in, 9);
  if (out.find("# HEADER TITLE") == std::string::npos) return 1;
  if (out.find("https://example.com/docs") == std::string::npos) return 1;
  if (out.find("```") == std::string::npos) return 1;
  if (out.find("20240101") == std::string::npos) return 1;
  return 0;
}

int test_determinism() {
  const std::string in =
      "Alpha sentence with detail. Alpha sentence with detail. Beta sentence with unique token xyz123.";
  const std::string o1 = squeeze_ex(in, 6);
  const std::string o2 = squeeze_ex(in, 6);
  return o1 == o2 ? 0 : 1;
}

int test_performance_sanity() {
  std::string in;
  in.reserve(250000);
  for (int i = 0; i < 2000; ++i) {
    in += "Sentence " + std::to_string(i % 100) + " with repeated platform detail and architecture note. ";
  }
  auto t0 = std::chrono::steady_clock::now();
  std::string out = squeeze_ex(in, 6);
  auto t1 = std::chrono::steady_clock::now();
  auto ms = std::chrono::duration_cast<std::chrono::milliseconds>(t1 - t0).count();
  return ms < 8000 ? 0 : 1;
}

}  // namespace

int main() {
  int rc = 0;
  auto run = [&](const char* name, int (*fn)()) {
    int r = fn();
    if (r != 0) std::cerr << "failed: " << name << "\n";
    rc |= r;
  };
  run("version", test_version_non_empty);
  run("identity", test_squeeze_identity_binary);
  run("segment", test_segmentation_abbrev_and_newlines);
  run("boilerplate", test_boilerplate_preserve_first_drop_repeats);
  run("duplicate", test_duplicate_removal);
  run("anchors", test_pruning_respects_anchors);
  run("determinism", test_determinism);
  run("performance", test_performance_sanity);
  if (rc != 0) std::cerr << "native tests failed\n";
  return rc;
}
