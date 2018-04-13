package com

// CloneBytes clone a copy of []byte
func CloneBytes(s []byte) []byte {
	out := make([]byte, len(s), len(s))
	copy(out, s)
	return out
}
