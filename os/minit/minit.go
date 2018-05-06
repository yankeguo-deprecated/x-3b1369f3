package minit

import (
	"bytes"
	"encoding/binary"
	"io"
	"log"
	"os"

	"landzero.net/x/io/pty"
)

// Command command
type Command struct {
	Cmd []string // must not be empty
	Env []string
	Pty bool
}

type winsizeWriter struct {
	p   *os.File
	buf *bytes.Buffer
}

func (w *winsizeWriter) Write(b []byte) (l int, err error) {
	if l, err = w.buf.Write(b); err != nil {
		return
	}
	if w.buf.Len() >= 4 {
		bs := make([]byte, 4, 4)
		if _, err = w.buf.Read(bs); err != nil {
			return
		}
		wi, hi := binary.BigEndian.Uint16(bs[0:2]), binary.BigEndian.Uint16(bs[2:4])
		log.Println("Size:", wi, hi)
		err = pty.Setsize(w.p, &pty.Winsize{Cols: wi, Rows: hi})
	}
	return
}

// NewWinsizeWriter create a writer, decode bytes, and change windows size
func NewWinsizeWriter(p *os.File) io.Writer {
	return &winsizeWriter{p: p, buf: &bytes.Buffer{}}
}
