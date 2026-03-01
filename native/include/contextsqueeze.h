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

typedef enum {
  CSQ_OK = 0,
  CSQ_ERR_INVALID_ARG = 1,
  CSQ_ERR_MALLOC = 2,
  CSQ_ERR_INTERNAL = 3,
  CSQ_ERR_INVALID_DATA = 4
} csq_error;

typedef void (*csq_progress_cb)(float percentage, void* user_data);

int csq_squeeze(csq_view in, csq_buf* out);
int csq_squeeze_ex(csq_view in, int aggressiveness, csq_buf* out);
int csq_squeeze_progress(csq_view in, int aggressiveness, csq_progress_cb cb, void* user_data, csq_buf* out);

void csq_free(csq_buf* buf);
const char* csq_version(void);
const char* csq_last_error(void);

#ifdef __cplusplus
}
#endif

#endif
