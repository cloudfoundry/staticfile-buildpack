package main

import (
	"errors"
	"log"
	"os"

	"github.com/cloudfoundry/libbuildpack/shims"
)

func main() {
	if len(os.Args) != 6 {
		log.Fatal(errors.New("incorrect number of arguments"))
	}

	finalizer := shims.Finalizer{DepsIndex: os.Args[4], ProfileDir: os.Args[5]}
	if err := finalizer.Finalize(); err != nil {
		log.Fatal(err)
	}
}
