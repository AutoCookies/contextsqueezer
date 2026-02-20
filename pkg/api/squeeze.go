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
	"fmt"
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

func normalizeAgg(opt Options) int {
	a := opt.Aggressiveness
	if a == 0 {
		switch opt.Profile {
		case "local":
			a = 6
		case "api":
			a = 4
		}
	}
	if a < 0 {
		a = 0
	}
	if a > 9 {
		a = 9
	}
	return a
}

func SqueezeBytes(in []byte, opt Options) ([]byte, error) {
	if len(in) == 0 {
		return []byte{}, nil
	}

	view := C.csq_view{data: (*C.char)(unsafe.Pointer(&in[0])), len: C.size_t(len(in))}
	var out C.csq_buf
	rc := C.csq_squeeze_ex(view, C.int(normalizeAgg(opt)), &out)
	if rc != 0 {
		rc2 := C.csq_squeeze(view, &out)
		if rc2 != 0 {
			return nil, fmt.Errorf("contextsqueeze native error: %d", int(rc))
		}
	}
	defer C.csq_free(&out)

	if out.len == 0 || out.data == nil {
		return []byte{}, nil
	}

	buf := C.GoBytes(unsafe.Pointer(out.data), C.int(out.len))
	return buf, nil
}
