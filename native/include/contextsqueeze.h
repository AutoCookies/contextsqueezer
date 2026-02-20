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

int csq_squeeze(csq_view in, csq_buf* out);
void csq_free(csq_buf* buf);
const char* csq_version(void);

#ifdef __cplusplus
}
#endif

#endif
