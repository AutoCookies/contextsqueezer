#include "metrics.h"

namespace {
csq_metrics_snapshot g_metrics = {0, 0, 0, 0};
}

extern "C" void csq_metrics_reset(void) { g_metrics = {0, 0, 0, 0}; }
extern "C" void csq_metrics_add_tokens(uint64_t n) { g_metrics.tokens_parsed += n; }
extern "C" void csq_metrics_add_sentences(uint64_t n) { g_metrics.sentences_total += n; }
extern "C" void csq_metrics_add_candidates(uint64_t n) { g_metrics.similarity_candidates_checked += n; }
extern "C" void csq_metrics_add_pairs(uint64_t n) { g_metrics.similarity_pairs_compared += n; }
extern "C" csq_metrics_snapshot csq_metrics_get(void) { return g_metrics; }
