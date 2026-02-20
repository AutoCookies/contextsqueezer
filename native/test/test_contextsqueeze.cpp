#include "contextsqueeze.h"

#include <cstring>
#include <iostream>
#include <string>
#include <vector>

namespace {

bool check_equal(const std::vector<char>& in, const csq_buf& out) {
  if (in.size() != out.len) {
    return false;
  }
  if (in.empty()) {
    return true;
  }
  return std::memcmp(in.data(), out.data, out.len) == 0;
}

int test_version_non_empty() {
  const char* v = csq_version();
  if (v == nullptr || std::strlen(v) == 0) {
    std::cerr << "version is empty\n";
    return 1;
  }
  return 0;
}

int test_squeeze_empty() {
  csq_view in{nullptr, 0};
  csq_buf out{nullptr, 0};
  if (csq_squeeze(in, &out) != 0) {
    std::cerr << "empty squeeze failed\n";
    return 1;
  }
  if (out.data != nullptr || out.len != 0) {
    std::cerr << "empty squeeze output invalid\n";
    return 1;
  }
  csq_free(&out);
  return 0;
}

int test_squeeze_text() {
  const std::string text = "hello context squeezer";
  const std::vector<char> in(text.begin(), text.end());
  csq_view view{in.data(), in.size()};
  csq_buf out{nullptr, 0};

  if (csq_squeeze(view, &out) != 0) {
    std::cerr << "text squeeze failed\n";
    return 1;
  }
  const bool ok = check_equal(in, out);
  csq_free(&out);
  if (!ok) {
    std::cerr << "text squeeze mismatch\n";
    return 1;
  }
  return 0;
}

int test_squeeze_binary() {
  const std::vector<char> in = {'a', '\0', 'b', '\n', '\0', 'z'};
  csq_view view{in.data(), in.size()};
  csq_buf out{nullptr, 0};

  if (csq_squeeze(view, &out) != 0) {
    std::cerr << "binary squeeze failed\n";
    return 1;
  }
  const bool ok = check_equal(in, out);
  csq_free(&out);
  if (!ok) {
    std::cerr << "binary squeeze mismatch\n";
    return 1;
  }
  return 0;
}

int test_free_null_safe() {
  csq_free(nullptr);
  csq_buf out{nullptr, 0};
  csq_free(&out);
  if (out.data != nullptr || out.len != 0) {
    std::cerr << "free on empty changed state unexpectedly\n";
    return 1;
  }
  return 0;
}

}  // namespace

int main() {
  int rc = 0;
  rc |= test_version_non_empty();
  rc |= test_squeeze_empty();
  rc |= test_squeeze_text();
  rc |= test_squeeze_binary();
  rc |= test_free_null_safe();
  return rc;
}
