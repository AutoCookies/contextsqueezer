#include "contextsqueeze.h"

#include <chrono>
#include <cstring>
#include <iostream>
#include <string>
#include <vector>

static int failures = 0;

static void expect(bool cond, const char* msg) {
  if (!cond) {
    std::cerr << "FAIL: " << msg << "\n";
    failures++;
  }
}

static std::vector<char> squeeze(const std::vector<char>& in, int aggr = 0) {
  csq_view view{in.data(), in.size()};
  csq_buf out{nullptr, 0};
  int rc = csq_squeeze_ex(view, aggr, &out);
  expect(rc == 0, "squeeze should succeed");
  std::vector<char> got;
  if (out.len > 0) {
    got.assign(out.data, out.data + out.len);
  }
  csq_free(&out);
  return got;
}

int main() {
  expect(csq_version() != nullptr, "version ptr not null");
  expect(std::strlen(csq_version()) > 0, "version non-empty");

  std::vector<char> empty;
  auto out_empty = squeeze(empty);
  expect(out_empty.empty(), "empty remains empty");

  std::string text = "hello world";
  auto out_text = squeeze(std::vector<char>(text.begin(), text.end()));
  expect(std::string(out_text.begin(), out_text.end()) == text, "text identity at aggr0");

  std::vector<char> binary = {'a', '\0', 'b', '\n', 'c'};
  auto out_bin = squeeze(binary);
  expect(out_bin == binary, "binary identity with null bytes");

  csq_free(nullptr);
  csq_buf nil{nullptr, 0};
  csq_free(&nil);
  expect(nil.data == nullptr && nil.len == 0, "free null-safe");

  std::string tricky = "Dr. A met Mr. B.\nSingle line still same sentence!\n\nNew block here. etc. done?";
  auto seg = squeeze(std::vector<char>(tricky.begin(), tricky.end()), 1);
  expect(!seg.empty(), "segmentation path should produce output");

  std::string boiler =
      "Intro.\n\n"
      "DISCLAIMER DISCLAIMER DISCLAIMER DISCLAIMER DISCLAIMER DISCLAIMER DISCLAIMER DISCLAIMER DISCLAIMER DISCLAIMER "
      "DISCLAIMER DISCLAIMER DISCLAIMER DISCLAIMER DISCLAIMER DISCLAIMER DISCLAIMER DISCLAIMER DISCLAIMER DISCLAIMER.\n\n"
      "Body text unique.\n\n"
      "DISCLAIMER DISCLAIMER DISCLAIMER DISCLAIMER DISCLAIMER DISCLAIMER DISCLAIMER DISCLAIMER DISCLAIMER DISCLAIMER "
      "DISCLAIMER DISCLAIMER DISCLAIMER DISCLAIMER DISCLAIMER DISCLAIMER DISCLAIMER DISCLAIMER DISCLAIMER DISCLAIMER.";
  auto boiled = squeeze(std::vector<char>(boiler.begin(), boiler.end()), 8);
  std::string boiled_s(boiled.begin(), boiled.end());
  expect(boiled_s.find("Body text unique") != std::string::npos, "boilerplate preserves body");

  std::string dup = "Alpha beta gamma are present. Alpha beta gamma are present! Keep last sentence.";
  auto duped = squeeze(std::vector<char>(dup.begin(), dup.end()), 9);
  std::string duped_s(duped.begin(), duped.end());
  expect(duped_s.find("Keep last sentence") != std::string::npos, "duplicate path keep remainder");

  std::string anchors = "```code```\nhttp://example.com\nOrder 1234 arrives.\nLOW INFO.\n";
  auto anch = squeeze(std::vector<char>(anchors.begin(), anchors.end()), 9);
  std::string anch_s(anch.begin(), anch.end());
  expect(anch_s.find("```code```") != std::string::npos, "code fence anchor kept");
  expect(anch_s.find("http://example.com") != std::string::npos, "url anchor kept");
  expect(anch_s.find("1234") != std::string::npos, "numeric anchor kept");

  std::string det = "One two three. One two three. Four five six.";
  auto d1 = squeeze(std::vector<char>(det.begin(), det.end()), 6);
  auto d2 = squeeze(std::vector<char>(det.begin(), det.end()), 6);
  expect(d1 == d2, "deterministic output");

  std::string big;
  for (int i = 0; i < 2000; ++i) {
    big += "Sentence number " + std::to_string(i % 200) + " has repeated structure and payload. ";
  }
  auto start = std::chrono::steady_clock::now();
  auto perf = squeeze(std::vector<char>(big.begin(), big.end()), 6);
  auto end = std::chrono::steady_clock::now();
  (void)perf;
  auto ms = std::chrono::duration_cast<std::chrono::milliseconds>(end - start).count();
  expect(ms < 1500, "performance sanity under 1.5s");

  if (failures > 0) {
    return 1;
  }
  std::cout << "All native tests passed\n";
  return 0;
}
