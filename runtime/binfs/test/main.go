package main

import (
	"fmt"
	"io/ioutil"

	"landzero.net/x/runtime/binfs"
)

func main() {
	f, err := binfs.Open("/other/other2/hello.txt")
	if err != nil {
		panic(err)
	}
	s, err := ioutil.ReadAll(f)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(s))
	binfs.Walk(func(n *binfs.Node) {
		var s string
		if n.FileInfo().IsDir() {
			s = "D"
		} else {
			s = "F"
		}
		fmt.Println("Node:", s, n.Path)
	})
}
