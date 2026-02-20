#ifndef CONTEXTSQUEEZE_METRICS_H
#define CONTEXTSQUEEZE_METRICS_H

#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

typedef struct {
  uint64_t tokens_parsed;
  uint64_t sentences_total;
  uint64_t similarity_candidates_checked;
  uint64_t similarity_pairs_compared;
} csq_metrics_snapshot;

void csq_metrics_reset(void);
void csq_metrics_add_tokens(uint64_t n);
void csq_metrics_add_sentences(uint64_t n);
void csq_metrics_add_candidates(uint64_t n);
void csq_metrics_add_pairs(uint64_t n);
csq_metrics_snapshot csq_metrics_get(void);

#ifdef __cplusplus
}
#endif

#endif
