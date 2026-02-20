#include "contextsqueeze.h"

#include <cstdlib>
#include <cstring>
#include <new>

namespace {
constexpr const char* kVersion = "0.1.0";
}

extern "C" int csq_squeeze(csq_view in, csq_buf* out) {
  if (out == nullptr) {
    return 1;
  }

  out->data = nullptr;
  out->len = 0;

  if (in.len == 0) {
    return 0;
  }
  if (in.data == nullptr) {
    return 4;
  }

  try {
    void* raw = std::malloc(in.len);
    if (raw == nullptr) {
      return 2;
    }

    std::memcpy(raw, in.data, in.len);
    out->data = static_cast<char*>(raw);
    out->len = in.len;
    return 0;
  } catch (...) {
    return 3;
  }
}

extern "C" void csq_free(csq_buf* buf) {
  if (buf == nullptr) {
    return;
  }

  std::free(buf->data);
  buf->data = nullptr;
  buf->len = 0;
}

extern "C" const char* csq_version(void) { return kVersion; }
