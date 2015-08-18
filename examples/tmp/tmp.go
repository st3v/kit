package main

import (
	"os"

	"github.com/go-kit/kit/log"
)

func main() {
	var logger log.Logger
	logger = log.NewLogfmtLogger(os.Stderr)
	logger = log.NewContext(logger).With("caller", log.DefaultCaller)

	_ = logger.Log("msg", "hello")
}
