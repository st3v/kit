package main

import (
	"os"

	"github.com/go-kit/kit/log"
)

func main() {
	var a log.Logger
	a = log.NewLogfmtLogger(os.Stderr)
	a = log.NewContext(a).With("caller", log.DefaultCaller)
	_ = a.Log("msg", "a")

	b1 := log.NewLogfmtLogger(os.Stderr)
	b2 := log.NewContext(b1).With("caller", log.DefaultCaller)
	_ = b2.Log("msg", "b")
}
