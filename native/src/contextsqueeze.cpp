#include "contextsqueeze.h"

#include <cstdlib>
#include <cstring>

namespace {
constexpr const char* kVersion = "0.1.0";
}

int csq_squeeze(csq_view in, csq_buf* out) {
  if (out == nullptr) {
    return 1;
  }

  out->data = nullptr;
  out->len = 0;

  try {
    if (in.len == 0) {
      return 0;
    }

    char* data = static_cast<char*>(std::malloc(in.len));
    if (data == nullptr) {
      return 2;
    }

    if (in.data != nullptr) {
      std::memcpy(data, in.data, in.len);
    }
    out->data = data;
    out->len = in.len;
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
