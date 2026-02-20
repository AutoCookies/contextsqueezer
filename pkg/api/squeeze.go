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
	"unsafe"
)

type Options struct {
	Aggressiveness int
	MaxTokens      int
	Profile        string
}

func Version() string {
	return C.GoString(C.csq_version())
}

func SqueezeBytes(in []byte, opt Options) ([]byte, error) {
	_ = opt

	var inPtr unsafe.Pointer
	if len(in) > 0 {
		inPtr = unsafe.Pointer(&in[0])
	}

	inView := C.csq_view{data: (*C.char)(inPtr), len: C.size_t(len(in))}
	out := C.csq_buf{}
	ret := C.csq_squeeze(inView, &out)
	if ret != 0 {
		return nil, errors.New("csq_squeeze failed")
	}
	defer C.csq_free(&out)

	if out.data == nil || out.len == 0 {
		return []byte{}, nil
	}

	buf := C.GoBytes(unsafe.Pointer(out.data), C.int(out.len))
	return buf, nil
}
