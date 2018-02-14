package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/cloudfoundry/libbuildpack/packager"
	"github.com/google/subcommands"
)

type summaryCmd struct {
}

func (*summaryCmd) Name() string             { return "summary" }
func (*summaryCmd) Synopsis() string         { return "Print out list of dependencies of this buildpack" }
func (*summaryCmd) SetFlags(f *flag.FlagSet) {}
func (*summaryCmd) Usage() string {
	return `summary:
  When run in a directory that is structured as a buildpack, prints a list of depedencies of that buildpack.
  (i.e. what would be downloaded to build a cached zipfile)
`
}
func (s *summaryCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	summary, err := packager.Summary(".")
	if err != nil {
		log.Printf("error reading dependencies from manifest: %v", err)
		return subcommands.ExitFailure
	}
	fmt.Println(summary)
	return subcommands.ExitSuccess
}

type buildCmd struct {
	cached   bool
	version  string
	cacheDir string
}

func (*buildCmd) Name() string     { return "build" }
func (*buildCmd) Synopsis() string { return "Create a buildpack zipfile from the current directory" }
func (*buildCmd) Usage() string {
	return `build [-cached] [-version <version>] [-cachedir <path to cachedir>]:
  When run in a directory that is structured as a buildpack, creates a zip file.

`
}
func (b *buildCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&b.version, "version", "", "version to build as")
	f.BoolVar(&b.cached, "cached", false, "include dependencies")
	f.StringVar(&b.cacheDir, "cachedir", packager.CacheDir, "cache dir")
}
func (b *buildCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if b.version == "" {
		v, err := ioutil.ReadFile("VERSION")
		if err != nil {
			log.Printf("error: Could not read VERSION file: %v", err)
			return subcommands.ExitFailure
		}
		b.version = strings.TrimSpace(string(v))
	}

	zipFile, err := packager.Package(".", b.cacheDir, b.version, b.cached)
	if err != nil {
		log.Printf("error while creating zipfile: %v", err)
		return subcommands.ExitFailure
	}

	buildpackType := "uncached"
	if b.cached {
		buildpackType = "cached"
	}

	stat, err := os.Stat(zipFile)
	if err != nil {
		log.Printf("error while stating zipfile: %v", err)
		return subcommands.ExitFailure
	}

	fmt.Printf("%s buildpack created and saved as %s with a size of %dMB\n", buildpackType, zipFile, stat.Size()/1024/1024)
	return subcommands.ExitSuccess
}
func main() {
	subcommands.Register(subcommands.HelpCommand(), "")
	subcommands.Register(subcommands.FlagsCommand(), "")
	subcommands.Register(subcommands.CommandsCommand(), "")
	subcommands.Register(&summaryCmd{}, "Custom")
	subcommands.Register(&buildCmd{}, "Custom")

	flag.Parse()
	ctx := context.Background() // TODO what is this even??
	os.Exit(int(subcommands.Execute(ctx)))
}
