package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/cloudfoundry/libbuildpack/packager"
)

func main() {
	var cached, summary bool
	var version, cacheDir string

	flag.StringVar(&version, "version", "", "version to build as")
	flag.BoolVar(&cached, "cached", false, "include dependencies")
	flag.BoolVar(&summary, "summary", false, "list dependencies")
	flag.StringVar(&cacheDir, "cachedir", packager.CacheDir, "cache dir")
	flag.Parse()

	if summary {
		if summary, err := packager.Summary("."); err != nil {
			log.Fatalf("error: %v", err)
		} else {
			fmt.Println(summary)
		}
		os.Exit(0)
	}

	if version == "" {
		v, err := ioutil.ReadFile("VERSION")
		if err != nil {
			log.Fatalf("error: Could not read VERSION file: %v", err)
		}
		version = strings.TrimSpace(string(v))
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
