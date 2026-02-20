package api

/*
#cgo CFLAGS: -I${SRCDIR}/../../native/include
#cgo linux LDFLAGS: -L${SRCDIR}/../../build/native/lib -lcontextsqueeze -Wl,-rpath,${SRCDIR}/../../build/native/lib
#cgo darwin LDFLAGS: -L${SRCDIR}/../../build/native/lib -lcontextsqueeze -Wl,-rpath,${SRCDIR}/../../build/native/lib
#include <stdlib.h>
#include "contextsqueeze.h"
#include "metrics.h"
*/
import "C"

import (
	"errors"
	"math"
	"unsafe"
)

func csqVersion() string {
	return C.GoString(C.csq_version())
}

func csqLastMetrics() NativeMetrics {
	m := C.csq_metrics_get()
	return NativeMetrics{
		TokensParsed:         uint64(m.tokens_parsed),
		SentencesTotal:       uint64(m.sentences_total),
		SimilarityCandidates: uint64(m.similarity_candidates_checked),
		SimilarityPairs:      uint64(m.similarity_pairs_compared),
	}
}

func csqSqueeze(in []byte, aggr int) ([]byte, error) {
	var view C.csq_view
	if len(in) > 0 {
		view.data = (*C.char)(unsafe.Pointer(&in[0]))
		view.len = C.size_t(len(in))
	}

	var out C.csq_buf
	status := C.csq_squeeze_ex(view, C.int(aggr), &out)
	if status != 0 {
		status = C.csq_squeeze(view, &out)
	}
	if status != 0 {
		return nil, errors.New("native squeeze returned non-zero")
	}
	defer C.csq_free(&out)

	if out.data == nil || out.len == 0 {
		return []byte{}, nil
	}
	if out.len > C.size_t(math.MaxInt32) {
		return nil, errors.New("native output too large")
	}
	return C.GoBytes(unsafe.Pointer(out.data), C.int(out.len)), nil
}
