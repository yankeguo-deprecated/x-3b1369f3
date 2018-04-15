package ioext

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
)

const (
	// LazyFileWriterMaxTry max time a lazy file writer tries to create the file
	LazyFileWriterMaxTry = 3
)

var (
	// ErrTooManyCreationFailure too many creation failure occured
	ErrTooManyCreationFailure = errors.New("LazyFileWriter: too many creation failure")
)

type lazyFilerWriter struct {
	filename string
	w        io.WriteCloser
	f        int
	mtx      *sync.Mutex
}

func (lfw *lazyFilerWriter) ensureFile() (w io.WriteCloser, err error) {
	if lfw.f > LazyFileWriterMaxTry {
		err = ErrTooManyCreationFailure
		return
	}
	lfw.mtx.Lock()
	defer lfw.mtx.Unlock()
	if lfw.w != nil {
		w = lfw.w
		return
	}
	// ensure directory
	d := filepath.Dir(lfw.filename)
	if err = os.MkdirAll(d, os.FileMode(0750)); err != nil {
		return
	}
	var f *os.File
	if f, err = os.OpenFile(lfw.filename, os.O_WRONLY|os.O_CREATE|os.O_APPEND, os.FileMode(0640)); err != nil {
		lfw.f++
		return
	}
	lfw.f = 0
	lfw.w = f
	w = f
	return
}

func (lfw *lazyFilerWriter) Write(p []byte) (n int, err error) {
	var w io.WriteCloser
	if w, err = lfw.ensureFile(); err != nil {
		return
	}
	return w.Write(p)
}

func (lfw *lazyFilerWriter) Close() error {
	lfw.mtx.Lock()
	defer lfw.mtx.Unlock()
	if lfw.w != nil {
		return lfw.Close()
	}
	return nil
}

// NewLazyFileWriter lazy file writer, create file on first write
func NewLazyFileWriter(filename string) io.WriteCloser {
	return &lazyFilerWriter{filename: filename, mtx: &sync.Mutex{}}
}
