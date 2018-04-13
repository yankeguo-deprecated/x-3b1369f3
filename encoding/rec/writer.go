/**
 * writer.go
 * Copyright (c) 2018 Yanke Guo
 *
 * This software is released under the MIT License.
 * https://opensource.org/licenses/MIT
 */

package rec

import (
	"encoding/binary"
	"errors"
	"io"
	"sync"
	"time"
)

var (
	// ErrNotActivated Writer is not activated
	ErrNotActivated = errors.New("Writer is not activated")
	// ErrUnknownFrameType Writer cannot recognize frame type, FrameWriter won't return this error
	ErrUnknownFrameType = errors.New("unknown frame type")
)

// FrameWriter rec file frame writer
type FrameWriter interface {
	/**
	 * WriteFrame
	 * write a frame to internal io.Writer
	 */
	WriteFrame(f Frame) error
	/**
	 * Close
	 * close internal io.Writer if it's a io.WriteCloser
	 */
	io.Closer
}

type frameWriter struct {
	w io.Writer
}

func (fw *frameWriter) Close() error {
	if c, ok := fw.w.(io.Closer); ok {
		return c.Close()
	}
	return nil
}
func (fw *frameWriter) WriteFrame(f Frame) (err error) {
	_, err = fw.w.Write(f.Encode())
	return
}

// NewFrameWriter create a new frame writer on io.Writer
func NewFrameWriter(w io.Writer) FrameWriter {
	return &frameWriter{w: w}
}

// Writer rec file writer
type Writer interface {
	/**
	 * FrameWriter
	 * return the internal frame writer
	 */
	FrameWriter() FrameWriter
	/**
	 * Activate
	 * activate the Writer, mark current time as the initial time for frame writing
	 */
	Activate()
	/**
	 * IsActivated()
	 */
	IsActivated() bool
	/**
	 * WriteStdout
	 * write a frame with stdout content, returns ErrNotActivated if Activate() is not invoked
	 */
	WriteStdout(p []byte) error
	/**
	 * WriteStderr
	 * write a frame with stderr content, returns ErrNotActivated if Activate() is not invoked
	 */
	WriteStderr(p []byte) error
	/**
	 * WriteWindowSize
	 * write a frame with window size, returns ErrNotActivated if Activate() is not invoked
	 */
	WriteWindowSize(w, h uint32) error // writes windowsize
	/**
	 * Stdout
	 * io.Writer wrapper for function WriteStdout()
	 */
	Stdout() io.Writer
	/**
	 * Stderr
	 * io.Writer wrapper for function WriteStderr()
	 */
	Stderr() io.Writer
	/**
	 * Close
	 * close the internal FrameWriter and clear activate flag
	 */
	io.Closer
}

type stdoutWriter struct {
	w Writer
}

func (sew *stdoutWriter) Write(p []byte) (n int, err error) {
	err = sew.w.WriteStdout(p)
	if err == nil {
		n = len(p)
	}
	return
}

type stderrWriter struct {
	w Writer
}

func (sew *stderrWriter) Write(p []byte) (n int, err error) {
	err = sew.w.WriteStderr(p)
	if err == nil {
		n = len(p)
	}
	return
}

type writer struct {
	f      *Frame
	sq     uint32
	fw     FrameWriter
	t0     time.Time
	active bool
	mtx    *sync.Mutex
}

func (w *writer) timestamp() uint32 {
	return uint32(time.Now().Sub(w.t0) / time.Millisecond)
}

func (w *writer) writeFrame(f Frame) (err error) {
	// if squeeze not enabled, just write
	if w.sq == 0 {
		return w.fw.WriteFrame(f)
	}
	// lock/unlock w.f
	w.mtx.Lock()
	defer w.mtx.Unlock()
	// if already cached
	if w.f != nil {
		// if same type and time is ok
		if w.f.Type == f.Type && f.Time-w.f.Time < w.sq {
			// append
			switch f.Type {
			case FrameStdout, FrameStderr:
				{
					// append payload
					o := make([]byte, len(w.f.Payload)+len(f.Payload), len(w.f.Payload)+len(f.Payload))
					copy(o, w.f.Payload)
					copy(o[len(w.f.Payload):], f.Payload)
					w.f.Payload = o
				}
			case FrameWindowSize:
				{
					// change payload
					w.f.Payload = f.Payload
				}
			default:
				return ErrUnknownFrameType
			}
		} else {
			// write cached frame
			err = w.fw.WriteFrame(*w.f)
			// cache frame
			w.f = &f
		}
	} else {
		// cache frame
		w.f = &f
	}
	return
}

func (w *writer) flushFrame() (err error) {
	// if squeeze not enabled, just ignore
	if w.sq == 0 {
		return
	}
	// lock/unlock w.f
	w.mtx.Lock()
	defer w.mtx.Unlock()
	// write w.f if not nil
	if w.f != nil {
		err = w.fw.WriteFrame(*w.f)
		w.f = nil
	}
	return
}

func (w *writer) FrameWriter() FrameWriter {
	return w.fw
}

func (w *writer) Activate() {
	w.active = true
	w.t0 = time.Now()
}

func (w *writer) IsActivated() bool {
	return w.active
}

func (w *writer) WriteStdout(p []byte) error {
	// clone payload, cause frame may be cached for later use
	o := make([]byte, len(p), len(p))
	copy(o, p)
	return w.writeFrame(Frame{
		Time:    w.timestamp(),
		Type:    FrameStdout,
		Payload: o,
	})
}

func (w *writer) WriteStderr(p []byte) error {
	// clone payload, cause frame may be cached for later use
	o := make([]byte, len(p), len(p))
	copy(o, p)
	return w.writeFrame(Frame{
		Time:    w.timestamp(),
		Type:    FrameStderr,
		Payload: o,
	})
}

func (w *writer) WriteWindowSize(width, height uint32) error {
	o := make([]byte, 8, 8)
	binary.BigEndian.PutUint32(o, width)
	binary.BigEndian.PutUint32(o[4:], height)
	return w.writeFrame(Frame{
		Time:    w.timestamp(),
		Type:    FrameWindowSize,
		Payload: o,
	})
}

func (w *writer) Stdout() io.Writer {
	return &stdoutWriter{w: w}
}

func (w *writer) Stderr() io.Writer {
	return &stderrWriter{w: w}
}

func (w *writer) Close() error {
	// flush frame
	w.flushFrame()
	// deactive
	w.active = false
	w.t0 = time.Time{}
	// close underlaying FrameWriter
	return w.fw.Close()
}

// WriterOption writer option
type WriterOption struct {
	/**
	 * SqueezeFrame
	 * number of milliseconds, frames with same type and time difference smaller than this value
	 * will be squeezed into one frame, 0 means no squeezing
	 */
	SqueezeFrame uint32
}

// NewWriter create a new writer
func NewWriter(w io.Writer, options ...WriterOption) Writer {
	var opt WriterOption
	if len(options) > 0 {
		opt = options[0]
	}
	return &writer{
		fw:  NewFrameWriter(w),
		sq:  opt.SqueezeFrame,
		mtx: &sync.Mutex{},
	}
}
