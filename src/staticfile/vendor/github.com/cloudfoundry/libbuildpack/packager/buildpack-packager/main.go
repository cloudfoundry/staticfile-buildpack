package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/cloudfoundry/libbuildpack/packager"
)

func main() {
	var cached bool
	var version, cacheDir string

	flag.StringVar(&version, "version", "", "version to build as")
	flag.BoolVar(&cached, "cached", false, "include dependencies")
	flag.StringVar(&cacheDir, "cachedir", packager.CacheDir, "cache dir")
	flag.Parse()

	if version == "" {
		v, err := ioutil.ReadFile("VERSION")
		if err != nil {
			log.Fatalf("error: Could not read VERSION file: %v", err)
		}
		version = string(v)
	}

	zipFile, err := packager.Package(".", cacheDir, version, cached)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	buildpackType := "uncached"
	if cached {
		buildpackType = "cached"
	}

	stat, err := os.Stat(zipFile)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	fmt.Printf("%s buildpack created and saved as %s with a size of %dMB\n", buildpackType, zipFile, stat.Size()/1024/1024)
}
