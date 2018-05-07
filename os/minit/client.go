package minit

import (
	"encoding/binary"
	"encoding/gob"
	"io"
	"io/ioutil"
	"net"
	"net/url"
	"strings"

	"landzero.net/x/io/stdcopy"
)

// Conn minit connection
type Conn interface {
	// SetWinsize set windows size
	SetWinsize(cols, rows uint16) (err error)
	// ReadFrom read stdin from a io.Reader
	ReadFrom(stdin io.Reader) (n int64, err error)
	// WriteTo2 write stdout/stderr to two io.Writer
	DemuxTo(stdout, stderr io.Writer) (n int64, err error)
	// Close close the underlaying net.Conn
	Close() error
}

type conn struct {
	nc    net.Conn
	stdin io.Writer
	stdws io.Writer
}

func (c *conn) SetWinsize(cols, rows uint16) (err error) {
	buf := make([]byte, 4, 4)
	binary.BigEndian.PutUint16(buf[0:2], cols)
	binary.BigEndian.PutUint16(buf[2:4], rows)
	_, err = c.stdws.Write(buf)
	return
}

func (c *conn) ReadFrom(stdin io.Reader) (n int64, err error) {
	return io.Copy(c.stdin, stdin)
}

func (c *conn) DemuxTo(stdout, stderr io.Writer) (n int64, err error) {
	if stdout == nil {
		stdout = ioutil.Discard
	}
	if stderr == nil {
		stderr = ioutil.Discard
	}
	return stdcopy.StdCopy(stdout, stderr, c.nc)
}

func (c *conn) Close() error {
	return c.nc.Close()
}

// DialURL dial a uri, supports tcp://host:ip and unix:///path/to/socket.sock
func DialURL(u string, cmd Command) (conn Conn, err error) {
	var ul *url.URL
	if ul, err = url.Parse(u); err != nil {
		return
	}
	var network string
	var host string
	if strings.ToLower(ul.Scheme) == "tcp" {
		network = "tcp"
		host = ul.Host
	} else if strings.ToLower(ul.Scheme) == "unix" {
		network = "unix"
		host = ul.Path
	} else {
		err = ErrURLSchemeNotSupported
		return
	}
	return Dial(network, host, cmd)
}

// Dial dial a new minit connection
func Dial(network, address string, cmd Command) (c Conn, err error) {
	// dial network
	var nc net.Conn
	if nc, err = net.Dial(network, address); err != nil {
		return
	}
	// send command
	if err = gob.NewEncoder(nc).Encode(cmd); err != nil {
		return
	}
	c = &conn{
		nc:    nc,
		stdin: stdcopy.NewStdWriter(nc, stdcopy.Stdout), // type stdout is used for stdin
		stdws: stdcopy.NewStdWriter(nc, stdcopy.Stderr), // type stderr is used for window size
	}
	return
}
