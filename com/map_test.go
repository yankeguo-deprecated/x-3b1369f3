package com

import "testing"

func TestNewMap(t *testing.T) {
	m := NewMap("hello", 1, "world", 2, "what", "3")
	if m["hello"] != 1 {
		t.Error("NewMap failed 1")
	}
	if m["world"] != 2 {
		t.Error("NewMap failed 2")
	}
	if m["what"] != "3" {
		t.Error("NewMap failed 3")
	}

}
