package main

import (
	"os"

	"go-counter/counter"
)

func main() {
	os.Exit(counter.CLI(os.Args[1:]))
}
