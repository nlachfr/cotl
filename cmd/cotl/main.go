package main

import (
	"log"

	"github.com/nlachfr/cotl/internal/cmd"
)

func main() {
	if err := cmd.BuildCommand().Execute(); err != nil {
		log.Fatal(err)
	}
}
