#include "contextsqueeze.h"

#include <chrono>
#include <cstring>
#include <iostream>
#include <string>
#include <vector>

namespace {

bool expect(bool cond, const char* msg) {
  if (!cond) {
    std::cerr << "FAIL: " << msg << std::endl;
  }
  return cond;
}

std::string squeeze_ex(const std::string& in, int aggr) {
  csq_view view{in.data(), in.size()};
  csq_buf out{nullptr, 0};
  int rc = csq_squeeze_ex(view, aggr, &out);
  if (rc != 0) {
    return "";
  }
  std::string res;
  if (out.data != nullptr && out.len > 0) {
    res.assign(out.data, out.len);
  }
  csq_free(&out);
  return res;
}

bool test_version() { return expect(std::string(csq_version()) == "0.2.0", "version should be 0.2.0"); }

bool test_segmentation_abbrev_and_newlines() {
  const std::string in = "Dr. Smith went home. This is next!\n\nNew para starts here. e.g. not split.";
  const std::string out = squeeze_ex(in, 3);
  bool ok = true;
  ok &= expect(!out.empty(), "output should not be empty");
  ok &= expect(out.find("Dr. Smith") != std::string::npos, "abbreviation sentence preserved");
  ok &= expect(out.find("New para starts here") != std::string::npos, "double newline split handled");
  return ok;
}

bool test_boilerplate_removes_repeats_keeps_first() {
  const std::string block =
      "NOTICE: Legal boilerplate legal boilerplate legal boilerplate legal boilerplate legal "
      "boilerplate legal boilerplate legal boilerplate legal boilerplate legal boilerplate.";
  const std::string in = block + "\n\n" + block + "\n\n" + "Unique sentence remains.";
  const std::string out = squeeze_ex(in, 6);
  bool ok = true;
  size_t first = out.find("NOTICE:");
  ok &= expect(first != std::string::npos, "first occurrence should remain");
  size_t second = out.find("NOTICE:", first + 1);
  ok &= expect(second == std::string::npos, "repeated block should be removed");
  return ok;
}

bool test_duplicate_removal() {
  const std::string in = "The quick brown fox jumps over the lazy dog. "
                         "The quick brown fox jumps over the lazy dog! "
                         "A different sentence remains.";
  const std::string out = squeeze_ex(in, 7);
  bool ok = true;
  ok &= expect(out.find("A different sentence remains") != std::string::npos, "distinct sentence kept");
  const size_t first = out.find("quick brown fox");
  ok &= expect(first != std::string::npos, "first duplicate sentence kept");
  const size_t second = out.find("quick brown fox", first + 1);
  ok &= expect(second == std::string::npos, "later duplicate sentence dropped");
  return ok;
}

bool test_anchors_preserved() {
  const std::string in = "# TITLE\n"
                         "https://example.com must stay.\n"
                         "```code```\n"
                         "ref id 1234 must stay.\n"
                         "Low info. Low info. Low info.";
  const std::string out = squeeze_ex(in, 9);
  bool ok = true;
  ok &= expect(out.find("# TITLE") != std::string::npos, "heading anchor kept");
  ok &= expect(out.find("https://example.com") != std::string::npos, "URL anchor kept");
  ok &= expect(out.find("```code```") != std::string::npos, "code fence anchor kept");
  ok &= expect(out.find("1234") != std::string::npos, "numeric anchor kept");
  return ok;
}

bool test_determinism() {
  const std::string in = "Alpha beta gamma. Alpha beta gamma. # KEEP\nhttps://x.y\n";
  const std::string out1 = squeeze_ex(in, 6);
  const std::string out2 = squeeze_ex(in, 6);
  return expect(out1 == out2, "same input should produce identical output");
}

bool test_perf_sanity() {
  std::string in;
  in.reserve(2000 * 80);
  for (int i = 0; i < 2000; ++i) {
    in += "Sentence repeated content number ";
    in += std::to_string(i % 20);
    in += ". ";
  }
  auto start = std::chrono::steady_clock::now();
  const std::string out = squeeze_ex(in, 6);
  const auto elapsed =
      std::chrono::duration_cast<std::chrono::milliseconds>(std::chrono::steady_clock::now() - start)
          .count();
  bool ok = true;
  ok &= expect(!out.empty(), "perf output should not be empty");
  ok &= expect(elapsed < 1500, "2000 sentence synthetic doc should process quickly");
  return ok;
}

}  // namespace

int main() {
  bool ok = true;
  ok &= test_version();
  ok &= test_segmentation_abbrev_and_newlines();
  ok &= test_boilerplate_removes_repeats_keeps_first();
  ok &= test_duplicate_removal();
  ok &= test_anchors_preserved();
  ok &= test_determinism();
  ok &= test_perf_sanity();

  if (!ok) {
    return 1;
  }
  std::cout << "All native tests passed" << std::endl;
  return 0;
}
