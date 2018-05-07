package main

import (
	"flag"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"golang.org/x/crypto/ssh/terminal"
	"landzero.net/x/io/pty"
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
	flag.StringVar(&sock, "H", "unix:///var/run/minit.sock", "socket file to connect")
	flag.Parse()
	if len(sock) == 0 {
		printHelp()
		os.Exit(1)
	}

	// write command
	cmd := minit.Command{
		Cmd: []string{"bash", "-il"},
		Pty: true,
		Env: buildEnv(),
	}
	var c minit.Conn
	var err error
	if c, err = minit.DialURL(sock, cmd); err != nil {
		panic(err)
	}
	// stream winsize
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGWINCH)
	go func() {
		for range ch {
			if hi, wi, err := pty.Getsize(os.Stdin); err == nil {
				c.SetWinsize(uint16(wi), uint16(hi))
			}
		}
	}()
	ch <- syscall.SIGWINCH // Initial resize.
	oldState, err := terminal.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		panic(err)
	}
	defer func() { _ = terminal.Restore(int(os.Stdin.Fd()), oldState) }() // Best effort.
	// stream stdout
	// stream stdin
	go c.ReadFrom(os.Stdin)
	c.DemuxTo(os.Stdout, os.Stderr)
	c.Close()
}

func printHelp() {
	println("Minit Test")
	println("  by Yanke Guo <guoyk.cn@gmail.com>")
	println("Usage:")
	flag.PrintDefaults()
}
