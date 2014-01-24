/*******************************************************************************
* Copyright (C) Zenoss, Inc. 2014, all rights reserved.
*******************************************************************************/

package circular

import (
	"testing"
)

// TODO: Write a performance benchmark to show improvements to impl.

func TestBuffer(t *testing.T) {

	const circularBufferSize = 5
	b := NewBuffer(circularBufferSize)

	testbytes := []byte{99, 1, 2, 3, 4, 5, 6, 7, 8, 9}

	if n, err := b.Write(testbytes); err != nil {
		t.Logf("Unexpected error when writing to circular buffer: %s", err)
		t.FailNow()
	} else {
		if n != len(testbytes) {
			t.Logf("expected %d bytes written, only %d were written", len(testbytes), n)
			t.FailNow()
		}
	}

	results := make([]byte, circularBufferSize)

	if n, err := b.Read(results); err != nil {
		t.Logf("Unexpected error when reading from circular buffer: %s", err)
		t.FailNow()
	} else {
		if n != circularBufferSize {
			t.Logf("expected %d bytes read, only %d were read", circularBufferSize, n)
			t.Logf("buffer: %v", b)
			t.Logf("results: %v", results)
			t.FailNow()
		}
	}
}
