package main

import (
	"github.com/JoshuaGabriel/goup/cmd"
	"os"
)

func main() {
	if err := cmd.Main(os.Args); err != nil {
		os.Exit(1)
	}
}
