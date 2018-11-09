package main

import (
	"github.com/cloudfoundry/libbuildpack/shims"
	"log"
	"os"
)

func main() {
	depsDir := os.Args[3]
	depsIndex := os.Args[4]
	profileDir := os.Args[5]

	err := shims.Finalize(depsDir, depsIndex, profileDir)
	if err != nil {
		log.Fatal(err)
	}
}
