package mshuf

import (
	"encoding/binary"
	"math/rand"
	"testing"
)

func TestMatrix_ShuffleIdentity(t *testing.T) {
	m := NewMatrix()
	for i := 0; i < MatrixSize; i++ {
		m.IdentityRowAt(i)
	}
	for i := 0; i < 100; i++ {
		b := make([]byte, 8, 8)
		rand.Read(b)
		n := binary.BigEndian.Uint64(b)
		r := m.Shuffle(n)
		if r != n {
			t.Errorf("Shuffle failed, expect %v = %v", n, r)
		}
	}
}
