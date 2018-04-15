package mshuf

import (
	"encoding/binary"
	"io"
	"math/rand"
)

// Matrix a matrix for mshuf
type Matrix []byte

// MatrixSize = 16
const MatrixSize = 16

// MatrixLength the length of matrix
const MatrixLength = MatrixSize * MatrixSize

// NewMatrix create a new empty matrix
func NewMatrix() Matrix {
	return make(Matrix, MatrixLength, MatrixLength)
}

// IdentityRowAt set identity sequence at row n
func (m Matrix) IdentityRowAt(n int) {
	for i := 0; i < MatrixSize; i++ {
		m[n*MatrixSize+i] = byte(i)
	}
}

// RandomRowAt set random sequence at row n
func (m Matrix) RandomRowAt(r io.Reader, n int) error {
	// seeds
	s := make([]byte, 8, 8)
	if _, err := r.Read(s); err != nil {
		return err
	}
	// make
	randSequence(int64(binary.BigEndian.Uint64(s)), m[n*MatrixSize:(n+1)*MatrixSize])
	return nil
}

// Shuffle shuffle a uint64
func (m Matrix) Shuffle(n uint64) uint64 {
	b := make([]byte, 8, 8)
	binary.BigEndian.PutUint64(b, n)
	for i := 0; i < 8; i++ {
		d := b[i]
		b[i] = m[i*2*MatrixSize+int(d>>4)]<<4 + m[(i*2+1)*MatrixSize+int(d&0x0f)]
	}
	return binary.BigEndian.Uint64(b)
}

// RandSequence create a rand sequence from 0 to f
func randSequence(seed int64, seq []byte) {
	// fill sequence
	for i := 0; i < len(seq); i++ {
		seq[i] = byte(i)
	}
	// Fisher-Yates shuffle
	rnd := rand.New(rand.NewSource(seed))
	for i := len(seq) - 1; i > 0; i = i - 1 {
		j := rnd.Intn(i)
		k := seq[i]
		seq[i] = seq[j]
		seq[j] = k
	}
	return
}
