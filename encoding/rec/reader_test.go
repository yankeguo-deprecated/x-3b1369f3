/**
 * reader_test.go
 * Copyright (c) 2018 Yanke Guo
 *
 * This software is released under the MIT License.
 * https://opensource.org/licenses/MIT
 */

package rec

import (
	"bytes"
	"testing"
	"time"
)

func TestReader(t *testing.T) {
	buf := &bytes.Buffer{}
	w := NewWriter(buf)
	w.Activate()
	<-time.NewTimer(time.Millisecond * 10).C
	w.WriteStdout([]byte("Hello World"))
	<-time.NewTimer(time.Millisecond * 12).C
	w.WriteStderr([]byte("Bong"))
	<-time.NewTimer(time.Millisecond * 13).C
	w.WriteWindowSize(100, 80)

	f := Frame{}
	r := NewFrameReader(bytes.NewReader(buf.Bytes()))
	r.ReadFrame(&f)
	if f.Type != FrameStdout {
		t.Errorf("invalid type %d", f.Type)
	}
	if string(f.Payload) != "Hello World" {
		t.Errorf("invalid content %s", string(f.Payload))
	}
	r.ReadFrame(&f)
	if f.Type != FrameStderr {
		t.Errorf("invalid type %d", f.Type)
	}
	if string(f.Payload) != "Bong" {
		t.Errorf("invalid content %s", string(f.Payload))
	}
	r.ReadFrame(&f)
	if f.Type != FrameWindowSize {
		t.Errorf("invalid type %d", f.Type)
	}
	wi, hi := f.DecodeWindowSize()
	if wi != 100 || hi != 80 {
		t.Errorf("invalid window size %d x %d", wi, hi)
	}
}
