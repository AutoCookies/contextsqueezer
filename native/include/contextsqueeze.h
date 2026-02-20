#ifndef CONTEXTSQUEEZE_H
#define CONTEXTSQUEEZE_H

#include <stddef.h>

#ifdef __cplusplus
extern "C" {
#endif

typedef struct {
  const char* data;
  size_t len;
} csq_view;

typedef struct {
  char* data;
  size_t len;
} csq_buf;

// Identity transform for Phase 0
// Returns 0 on success, non-zero error code on failure.
int csq_squeeze(csq_view in, csq_buf* out);

// Frees buffer allocated by csq_squeeze
void csq_free(csq_buf* buf);

// Returns a static version string like "0.1.0"
const char* csq_version(void);

#ifdef __cplusplus
}
#endif

#endif
