package com

import "testing"

func TestCloneBytes(t *testing.T) {
	s := []byte{1, 2, 3, 4}
	o := CloneBytes(s)
	if len(o) != 4 {
		t.Error("clone failed")
	}
	if o[2] != 3 {
		t.Error("value not right")
	}
	s[1] = 3
	if o[1] != 2 {
		t.Error("underlaying un refered")
	}
}
