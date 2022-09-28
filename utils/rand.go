package utils

import "fmt"

// #include "rand.h"
import "C"

type RandReader struct {
}

func (r *RandReader) Read(out []byte) (n int, err error) { // implements io.Reader
	var x C.uint16_t
	var retry C.int = 1
	for i := 0; i < 64; i++ {
		res := C.rdrand_16(&x, retry)
		if res < 0 {
			return i, fmt.Errorf("cannot read random bytes")
		}
		out[i] = byte(x)
	}
	return len(out), nil
}
