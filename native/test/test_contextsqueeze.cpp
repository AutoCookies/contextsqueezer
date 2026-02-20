#include "contextsqueeze.h"

#include <cstring>
#include <iostream>
#include <vector>

namespace {

bool expect(bool cond, const char* msg) {
  if (!cond) {
    std::cerr << "FAIL: " << msg << std::endl;
  }
  return cond;
}

bool test_version_non_empty() {
  const char* v = csq_version();
  return expect(v != nullptr && std::strlen(v) > 0, "csq_version should be non-empty");
}

bool test_squeeze_empty() {
  csq_buf out{nullptr, 0};
  csq_view in{nullptr, 0};
  const int rc = csq_squeeze(in, &out);
  bool ok = true;
  ok &= expect(rc == 0, "csq_squeeze empty rc == 0");
  ok &= expect(out.data == nullptr, "empty output data should be null");
  ok &= expect(out.len == 0, "empty output len should be 0");
  csq_free(&out);
  return ok;
}

bool test_squeeze_text() {
  const char text[] = "hello";
  csq_view in{text, 5};
  csq_buf out{nullptr, 0};
  const int rc = csq_squeeze(in, &out);
  bool ok = true;
  ok &= expect(rc == 0, "csq_squeeze text rc == 0");
  ok &= expect(out.len == 5, "text output len should match");
  ok &= expect(out.data != nullptr, "text output data should not be null");
  ok &= expect(std::memcmp(out.data, text, 5) == 0, "text output bytes should match input");
  csq_free(&out);
  return ok;
}

bool test_squeeze_binary() {
  const std::vector<char> input = {'a', '\0', 'b', '\0', static_cast<char>(0xff), 'c'};
  csq_view in{input.data(), input.size()};
  csq_buf out{nullptr, 0};
  const int rc = csq_squeeze(in, &out);
  bool ok = true;
  ok &= expect(rc == 0, "csq_squeeze binary rc == 0");
  ok &= expect(out.len == input.size(), "binary output len should match");
  ok &= expect(out.data != nullptr, "binary output data should not be null");
  ok &= expect(std::memcmp(out.data, input.data(), input.size()) == 0,
               "binary output bytes should match input");
  csq_free(&out);
  return ok;
}

bool test_free_null_safe() {
  csq_free(nullptr);
  csq_buf out{nullptr, 0};
  csq_free(&out);
  return expect(out.data == nullptr && out.len == 0, "csq_free null/empty safe");
}

}  // namespace

int main() {
  bool ok = true;
  ok &= test_version_non_empty();
  ok &= test_squeeze_empty();
  ok &= test_squeeze_text();
  ok &= test_squeeze_binary();
  ok &= test_free_null_safe();

  if (!ok) {
    return 1;
  }
  std::cout << "All native tests passed" << std::endl;
  return 0;
}
