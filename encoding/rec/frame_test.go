/**
 * frame_Test.go
 * Copyright (c) 2018 Yanke Guo
 *
 * This software is released under the MIT License.
 * https://opensource.org/licenses/MIT
 */

package rec

import (
	"bytes"
	"testing"
)

func TestFrameEncode(t *testing.T) {
	f := Frame{
		Time:    123,
		Type:    FrameStdout,
		Payload: []byte{0xff, 0x23, 0x1d, 0x3c},
	}
	o := f.Encode()
	v := []byte{0x00, 0x00, 0x00, 0x7b, 0x01, 0x00, 0x00, 0x00, 0x04, 0xff, 0x23, 0x1d, 0x3c}
	if bytes.Compare(o, v) != 0 {
		t.Errorf("invalid encode result")
	}
}
