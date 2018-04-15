package ioext

import "io"

type silentWriter struct {
	w io.Writer
}

func (sw *silentWriter) Write(p []byte) (n int, err error) {
	sw.w.Write(p)
	n = len(p)
	return
}

func (sw *silentWriter) Close() error {
	if c, ok := sw.w.(io.Closer); ok {
		c.Close()
	}
	return nil
}

// NewSilentWriter returns a new io.WriteCloser which never returns error
func NewSilentWriter(w io.Writer) io.WriteCloser {
	return &silentWriter{w: w}
}
