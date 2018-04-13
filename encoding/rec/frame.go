/**
 * frame.go
 * Copyright (c) 2018 Yanke Guo
 *
 * This software is released under the MIT License.
 * https://opensource.org/licenses/MIT
 */

package rec

import (
	"encoding/binary"
)

const (
	// FrameStdout frame type - stdout
	FrameStdout = byte(1)
	// FrameStderr frame type - stderr
	FrameStderr = byte(2)
	// FrameWindowSize frame type - window size
	FrameWindowSize = byte(3)
)

// Frame a single frame in rec file
type Frame struct {
	Time    uint32 // timestamp, in ms
	Type    byte   // frame type
	Payload []byte // payload data
}

// Encode encode frame to bytes sequence
func (f Frame) Encode() []byte {
	/* TIMESTAMP (4 bytes) + TYPE (1 byte) + PAYLOAD_LEN (4 bytes) + PAYLOAD */
	l := 4 + 1 + 4 + len(f.Payload)
	out := make([]byte, l, l)
	binary.BigEndian.PutUint32(out, f.Time)
	out[4] = f.Type
	binary.BigEndian.PutUint32(out[5:], uint32(len(f.Payload)))
	copy(out[9:], f.Payload)
	return out
}

// DecodeWindowSize decode window size from Payload
func (f Frame) DecodeWindowSize() (w uint32, h uint32) {
	if len(f.Payload) < 8 {
		return
	}
	w = binary.BigEndian.Uint32(f.Payload)
	h = binary.BigEndian.Uint32(f.Payload[4:])
	return
}
