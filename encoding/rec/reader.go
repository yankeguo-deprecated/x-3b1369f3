/**
 * reader.go
 * Copyright (c) 2018 Yanke Guo
 *
 * This software is released under the MIT License.
 * https://opensource.org/licenses/MIT
 */

package rec

import (
	"encoding/binary"
	"io"
)

// FrameReader frame reader for rec file stream
type FrameReader interface {
	/**
	 * ReadFrame
	 * read a frame from underlaying io.Reader
	 */
	ReadFrame(f *Frame) error
}

type frameReader struct {
	r io.Reader
}

func (r *frameReader) ReadFrame(f *Frame) (err error) {
	// head cache, 4 + 1 + 4
	h := make([]byte, 9, 9)
	_, err = r.r.Read(h)
	if err != nil {
		return
	}
	f.Time = binary.BigEndian.Uint32(h)
	f.Type = h[4]
	l := binary.BigEndian.Uint32(h[5:])
	if l > 0 {
		f.Payload = make([]byte, l, l)
		_, err = r.r.Read(f.Payload)
	} else {
		f.Payload = make([]byte, 0, 0)
	}
	return
}

// NewFrameReader create a new frame reader
func NewFrameReader(r io.Reader) FrameReader {
	return &frameReader{r: r}
}
