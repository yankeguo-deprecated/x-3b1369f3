package minit

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync/atomic"

	"landzero.net/x/io/ioext"
	"landzero.net/x/io/pty"
	"landzero.net/x/io/stdcopy"
)

var (
	// ErrEmptyCommand empty command
	ErrEmptyCommand = errors.New("empty command")
)

type winsizeWriter struct {
	p   *os.File
	buf *bytes.Buffer
}

func (w *winsizeWriter) Write(b []byte) (l int, err error) {
	w.buf.Write(b) // Buffer returns no error
	if w.buf.Len() >= 4 {
		bs := make([]byte, 4, 4)
		w.buf.Read(bs) // Buffer returns no error
		pty.Setsize(w.p, &pty.Winsize{
			Cols: binary.BigEndian.Uint16(bs[0:2]),
			Rows: binary.BigEndian.Uint16(bs[2:4]),
		}) // ignore error
	}
	l = len(b)
	return
}

// newWinsizeWriter create a writer, decode bytes, and change windows size
func newWinsizeWriter(p *os.File) io.Writer {
	return &winsizeWriter{p: p, buf: &bytes.Buffer{}}
}

// Serve serve on a net.Listener and blocks
func Serve(l net.Listener) (err error) {
	var id uint64
	for {
		var c net.Conn
		if c, err = l.Accept(); err != nil {
			break
		}
		go NewServerConn(c, atomic.AddUint64(&id, 1)).Handle()
	}
	return
}

// ServerConn server side connection
type ServerConn interface {
	// Handle the connection and blocks
	Handle() error
}

type serverConn struct {
	nc net.Conn
	id uint64
}

func (sc *serverConn) Handle() (err error) {
	name := fmt.Sprintf("[conn-%d]", sc.id)
	defer log.Println(name, "disconnected")
	defer sc.nc.Close()
	log.Println(name, "connected")
	// decode command
	var cmd Command
	if err = gob.NewDecoder(sc.nc).Decode(&cmd); err != nil {
		log.Println(name, "failed to decode command", err)
		return
	}
	// check command
	if len(cmd.Cmd) == 0 {
		err = ErrEmptyCommand
		log.Println(name, err.Error())
		return
	}
	if cmd.Env == nil {
		cmd.Env = []string{}
	}
	log.Println(name, "Cmd:", strings.Join(cmd.Cmd, ","), "Env:", strings.Join(cmd.Env, ","))
	// exec
	ecmd := exec.Command(cmd.Cmd[0], cmd.Cmd[1:]...)
	ecmd.Env = append(os.Environ(), cmd.Env...)
	// rebuild stream
	if cmd.Pty {
		var p *os.File
		if p, err = pty.Start(ecmd); err != nil {
			log.Println(name, "failed to allocate pty", err)
			return
		}
		defer p.Close()
		go func() {
			stdcopy.StdCopy(ioext.NewSilentWriter(p), newWinsizeWriter(p), sc.nc) // ignore write error
			if proc := ecmd.Process; proc != nil {
				proc.Kill() // kill after disconnect
			}
		}()
		go io.Copy(stdcopy.NewStdWriter(sc.nc, stdcopy.Stdout), p)
	} else {
		stdir, stdiw := io.Pipe()
		ecmd.Stdin = stdir
		ecmd.Stdout = stdcopy.NewStdWriter(sc.nc, stdcopy.Stdout)
		ecmd.Stderr = stdcopy.NewStdWriter(sc.nc, stdcopy.Stderr)
		if err = ecmd.Start(); err != nil {
			log.Println(name, "failed to start", err)
			return
		}
		go func() {
			stdcopy.StdCopy(ioext.NewSilentWriter(stdiw), ioutil.Discard, sc.nc) // ignore write error
			if proc := ecmd.Process; proc != nil {
				proc.Kill() // kill after disconnect
			}
		}()
	}
	if err = ecmd.Wait(); err != nil {
		log.Println(name, "command failed", err)
		return
	}
	return
}

// NewServerConn create a server connection
func NewServerConn(nc net.Conn, id uint64) ServerConn {
	return &serverConn{nc: nc, id: id}
}
