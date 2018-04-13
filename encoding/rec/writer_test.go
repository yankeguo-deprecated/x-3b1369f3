/**
 * writer_test.go
 * Copyright (c) 2018 Yanke Guo
 *
 * This software is released under the MIT License.
 * https://opensource.org/licenses/MIT
 */

package rec

import (
	"bytes"
	"io"
	"testing"
	"time"
)

func TestWriter(t *testing.T) {
	b := &bytes.Buffer{}
	w := NewWriter(b)
	w.Activate()
	// not trigger frame compacting
	w.WriteStdout([]byte{0x01, 0x02, 0x03, 0x04})
	<-time.NewTimer(time.Millisecond * 10).C
	w.WriteStderr([]byte{0x04, 0x03, 0x02, 0x01})
	<-time.NewTimer(time.Millisecond * 10).C
	w.Stdout().Write([]byte{0x01, 0x02, 0x03, 0x04})
	<-time.NewTimer(time.Millisecond * 20).C
	w.Stderr().Write([]byte{0x03, 0x04, 0x05, 0x01})
	<-time.NewTimer(time.Millisecond * 30).C
	w.WriteWindowSize(20, 50)
	w.Close()
	o := b.Bytes()
	if len(o) != (13*4 + 17) {
		t.Errorf("invalid length: %d", len(o))
	}
}

func TestWriterWithSqueeze(t *testing.T) {
	b := &bytes.Buffer{}
	w := NewWriter(b, WriterOption{
		SqueezeFrame: 100,
	})
	w.Activate()
	w.WriteStdout([]byte("hello,"))
	<-time.NewTimer(time.Millisecond * 10).C
	w.WriteStdout([]byte(" world"))
	<-time.NewTimer(time.Millisecond * 300).C
	w.WriteStdout([]byte("bong"))
	<-time.NewTimer(time.Millisecond * 10).C
	w.WriteStdout([]byte("pong"))
	<-time.NewTimer(time.Millisecond * 10).C
	w.WriteStderr([]byte("abc"))
	<-time.NewTimer(time.Millisecond * 10).C
	w.WriteWindowSize(100, 80)
	<-time.NewTimer(time.Millisecond * 10).C
	w.WriteWindowSize(80, 50)
	w.Close()

	r := NewFrameReader(bytes.NewReader(b.Bytes()))
	f := Frame{}
	// 1
	r.ReadFrame(&f)
	if f.Type != FrameStdout || string(f.Payload) != "hello, world" {
		t.Errorf("frame bad 1: %s", string(f.Payload))
	}
	// 2
	r.ReadFrame(&f)
	if f.Type != FrameStdout || string(f.Payload) != "bongpong" {
		t.Errorf("frame bad 2: %s", string(f.Payload))
	}
	// 3
	r.ReadFrame(&f)
	if f.Type != FrameStderr || string(f.Payload) != "abc" {
		t.Errorf("frame bad 3: %s", string(f.Payload))
	}
	// 4
	r.ReadFrame(&f)
	wi, hi := f.DecodeWindowSize()
	if f.Type != FrameWindowSize || wi != 80 || hi != 50 {
		t.Errorf("frame bad 4: %d x %d", wi, hi)
	}
	err := r.ReadFrame(&f)
	if err != io.EOF {
		t.Errorf("bad frame")
	}
}
