package main

import (
	"encoding/gob"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"

	"landzero.net/x/io/pty"
	"landzero.net/x/io/stdcopy"
	"landzero.net/x/os/minit"
)

var sock string
var connid uint64

func main() {
	// parse flags
	flag.StringVar(&sock, "L", "/var/run/minit/minit.sock", "socket file to listen")
	flag.Parse()
	if len(sock) == 0 {
		printHelp()
		os.Exit(1)
	}
	// try remove existing sock file
	os.Remove(sock)
	// try create parrent directory
	os.MkdirAll(filepath.Dir(sock), os.FileMode(0755))
	// listen sock file
	var err error
	var l net.Listener
	if l, err = net.Listen("unix", sock); err != nil {
		log.Println("Failed to listen", sock)
		return
	}
	log.Println("Listening on", sock)
	// the listen loop
	for {
		var c net.Conn
		if c, err = l.Accept(); err != nil {
			break
		} else {
			go handleConnection(c)
		}
	}
}

func handleConnection(c net.Conn) {
	// connection id
	id := fmt.Sprintf("[conn-%d]", atomic.AddUint64(&connid, 1))
	// defer to close
	defer log.Println(id, "closed")
	defer c.Close()
	//
	var cmd minit.Command
	var err error
	// decode Command
	dec := gob.NewDecoder(c)
	if err = dec.Decode(&cmd); err != nil {
		log.Println(id, "failed to decode command:", err)
		return
	}
	// execute command
	if len(cmd.Cmd) == 0 {
		log.Println(id, "empty command")
		return
	}
	log.Println(id, "command:", strings.Join(cmd.Cmd, ","), "env:", strings.Join(cmd.Env, ","))
	// exec
	ecmd := exec.Command(cmd.Cmd[0], cmd.Cmd[1:]...)
	if len(cmd.Env) > 0 {
		ecmd.Env = cmd.Env
	} else {
		ecmd.Env = []string{}
	}
	if cmd.Pty {
		var p *os.File
		if p, err = pty.Start(ecmd); err != nil {
			log.Println(id, "failed to allocate pty", err)
			return
		}
		log.Println(id, "pty allocated")
		defer p.Close()
		go stdcopy.StdCopy(p, minit.NewWinsizeWriter(p), c)
		go io.Copy(c, p)
	} else {
		ecmd.Stdin = c
		ecmd.Stdout = stdcopy.NewStdWriter(c, stdcopy.Stdout)
		ecmd.Stderr = stdcopy.NewStdWriter(c, stdcopy.Stderr)
		if err = ecmd.Start(); err != nil {
			log.Println(id, "failed to start", err)
			return
		}
	}

	if err = ecmd.Wait(); err != nil {
		log.Println(id, "command failed", err)
		return
	}
}

func printHelp() {
	println("Minit")
	println("  by Yanke Guo <guoyk.cn@gmail.com>")
	println("Usage:")
	flag.PrintDefaults()
}
