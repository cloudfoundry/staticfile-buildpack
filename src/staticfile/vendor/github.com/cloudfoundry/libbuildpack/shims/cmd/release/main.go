package main

import (
	"github.com/cloudfoundry/libbuildpack/shims"
	"log"
	"os"
)

func main() {
	buildDir := os.Args[1]

	if err := shims.Release(buildDir, os.Stdout); err != nil {
		log.Fatal(err)
	}
}
