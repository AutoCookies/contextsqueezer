package api

/*
#cgo CFLAGS: -I${SRCDIR}/../../native/include
#cgo linux LDFLAGS: -L${SRCDIR}/../../build/native/lib -lcontextsqueeze -Wl,-rpath,${SRCDIR}/../../build/native/lib
#cgo darwin LDFLAGS: -L${SRCDIR}/../../build/native/lib -lcontextsqueeze -Wl,-rpath,${SRCDIR}/../../build/native/lib
#include <stdlib.h>
#include "contextsqueeze.h"
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

func csqSqueeze(in []byte) ([]byte, error) {
	var view C.csq_view
	if len(in) > 0 {
		view.data = (*C.char)(unsafe.Pointer(&in[0]))
		view.len = C.size_t(len(in))
	}

	var out C.csq_buf
	status := C.csq_squeeze(view, &out)
	if status != 0 {
		return nil, errors.New("native csq_squeeze returned non-zero")
	}
	defer C.csq_free(&out)

	if out.data == nil || out.len == 0 {
		return []byte{}, nil
	}

	if out.len > C.size_t(math.MaxInt32) {
		return nil, errors.New("native output too large")
	}

	result := C.GoBytes(unsafe.Pointer(out.data), C.int(out.len))
	return result, nil
}
