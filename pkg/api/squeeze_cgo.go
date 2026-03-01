package api

/*
#cgo CFLAGS: -I${SRCDIR}/../../native/include
#cgo linux LDFLAGS: -L${SRCDIR}/../../build/native/lib -lcontextsqueeze -Wl,-rpath,${SRCDIR}/../../build/native/lib
#cgo darwin LDFLAGS: -L${SRCDIR}/../../build/native/lib -lcontextsqueeze -Wl,-rpath,${SRCDIR}/../../build/native/lib
#include <stdlib.h>
#include "contextsqueeze.h"
#include "metrics.h"

extern void goProgressCallback(float pct, void* userData);
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

func csqLastError() string {
	return C.GoString(C.csq_last_error())
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

//export goProgressCallback
func goProgressCallback(pct C.float, userData unsafe.Pointer) {
	cb := (*func(float32))(userData)
	if cb != nil {
		(*cb)(float32(pct))
	}
}

func csqSqueeze(in []byte, aggr int, cb *func(float32)) ([]byte, error) {
	var view C.csq_view
	if len(in) > 0 {
		view.data = (*C.char)(unsafe.Pointer(&in[0]))
		view.len = C.size_t(len(in))
	}

	var out C.csq_buf
	var status C.int
	if cb != nil {
		status = C.csq_squeeze_progress(view, C.int(aggr), (C.csq_progress_cb)(C.goProgressCallback), unsafe.Pointer(cb), &out)
	} else {
		status = C.csq_squeeze_ex(view, C.int(aggr), &out)
	}

	if status != 0 {
		errStr := csqLastError()
		if errStr == "" {
			errStr = "native squeeze returned non-zero"
		}
		return nil, errors.New(errStr)
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
