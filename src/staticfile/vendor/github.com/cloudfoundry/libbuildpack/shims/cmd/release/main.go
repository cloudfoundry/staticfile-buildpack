package main

import (
	"errors"
	"log"
	"os"
	"path/filepath"

	"github.com/cloudfoundry/libbuildpack/shims"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatal(errors.New("incorrect number of arguments"))
	}

	metadataPath := filepath.Join(os.Args[1], ".cloudfoundry", "metadata.toml")

	releaser := shims.Releaser{MetadataPath: metadataPath, Writer: os.Stdout}
	if err := releaser.Release(); err != nil {
		log.Fatal(err)
	}
}
