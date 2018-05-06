package main

import (
	"encoding/binary"
	"encoding/gob"
	"flag"
	"io"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"golang.org/x/crypto/ssh/terminal"
	"landzero.net/x/io/pty"
	"landzero.net/x/io/stdcopy"
	"landzero.net/x/os/minit"
)

var sock string

var envwl = []string{
	"TERM",
}

func buildEnv() []string {
	ret := []string{}
	for _, l := range os.Environ() {
		for _, wl := range envwl {
			if strings.HasPrefix(l, wl+"=") {
				ret = append(ret, l)
			}
		}
	}
	return ret
}

func main() {
	flag.StringVar(&sock, "H", "/var/run/minit.sock", "socket file to connect")
	flag.Parse()
	if len(sock) == 0 {
		printHelp()
		os.Exit(1)
	}
	var err error
	var addr *net.UnixAddr
	if addr, err = net.ResolveUnixAddr("unix", sock); err != nil {
		panic(err)
	}
	var c *net.UnixConn
	if c, err = net.DialUnix("unix", nil, addr); err != nil {
		panic(err)
	}
	// write command
	cmd := minit.Command{
		Cmd: []string{"bash", "-il"},
		Pty: true,
		Env: buildEnv(),
	}
	gob.NewEncoder(c).Encode(cmd)
	// stream stdin
	go func() {
		io.Copy(stdcopy.NewStdWriter(c, stdcopy.Stdout), os.Stdin)
		c.CloseWrite()
	}()
	// stream winsize
	ch := make(chan os.Signal, 1)
	sw := stdcopy.NewStdWriter(c, stdcopy.Stderr)
	signal.Notify(ch, syscall.SIGWINCH)
	go func() {
		for range ch {
			var wi int
			var hi int
			var err error
			if hi, wi, err = pty.Getsize(os.Stdin); err != nil {
				continue
			}
			bs := make([]byte, 4, 4)
			binary.BigEndian.PutUint16(bs, uint16(wi))
			binary.BigEndian.PutUint16(bs[2:4], uint16(hi))
			sw.Write(bs)
		}
	}()
	ch <- syscall.SIGWINCH // Initial resize.
	oldState, err := terminal.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		panic(err)
	}
	defer func() { _ = terminal.Restore(int(os.Stdin.Fd()), oldState) }() // Best effort.
	// stream stdout
	io.Copy(os.Stdout, c)
	c.Close()
}

func printHelp() {
	println("Minit Test")
	println("  by Yanke Guo <guoyk.cn@gmail.com>")
	println("Usage:")
	flag.PrintDefaults()
}
