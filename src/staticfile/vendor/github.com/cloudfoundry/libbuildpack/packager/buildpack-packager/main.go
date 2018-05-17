package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudfoundry/libbuildpack"
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
	anyStack bool
	version  string
	cacheDir string
	stack    string
}

func (*buildCmd) Name() string     { return "build" }
func (*buildCmd) Synopsis() string { return "Create a buildpack zipfile from the current directory" }
func (*buildCmd) Usage() string {
	return `build -stack <stack>|-any-stack [-cached] [-version <version>] [-cachedir <path to cachedir>]:
  When run in a directory that is structured as a buildpack, creates a zip file.

`
}
func (b *buildCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&b.version, "version", "", "version to build as")
	f.BoolVar(&b.cached, "cached", false, "include dependencies")
	f.StringVar(&b.cacheDir, "cachedir", packager.CacheDir, "cache dir")

	f.StringVar(&b.stack, "stack", "", "stack to package buildpack for")
	f.BoolVar(&b.anyStack, "any-stack", false, "package buildpack for any stack")
}
func (b *buildCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if b.stack == "" && !b.anyStack {
		log.Printf("error: must either specify a stack or pass -any-stack")
		return subcommands.ExitFailure
	}
	if b.stack != "" && b.anyStack {
		log.Printf("error: cannot specify a stack AND pass -any-stack")
		return subcommands.ExitFailure
	}
	if b.version == "" {
		v, err := ioutil.ReadFile("VERSION")
		if err != nil {
			log.Printf("error: Could not read VERSION file: %v", err)
			return subcommands.ExitFailure
		}
		b.version = strings.TrimSpace(string(v))
	}

	zipFile, err := packager.Package(".", b.cacheDir, b.version, b.stack, b.cached)
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

type initCmd struct {
	name string
	dir  string
}

func (*initCmd) Name() string { return "init" }
func (*initCmd) Synopsis() string {
	return "Creates a folder with the basic structure of a new buildpack"
}
func (i *initCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&i.name, "name", "", "Name of the buildpack. Required.")
	f.StringVar(&i.dir, "path", "", "Path to folder to create. Defaults to the name + '-buildpack' in the current directory.")
}
func (*initCmd) Usage() string {
	return `init:
	Create a new directory that is structured as a buildpack.
`
}
func (i *initCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	fmt.Println("Init", i.name, i.dir)

	if i.name == "" {
		log.Printf("error: no name entered for new buildpack")
		return subcommands.ExitUsageError
	}

	// assume user doesn't want -buildpack in the language name
	i.name = strings.TrimSuffix(i.name, "-buildpack")

	if i.dir == "" {
		i.dir = i.name + "-buildpack"
	}
	var err error
	i.dir, err = filepath.Abs(i.dir)
	if err != nil {
		log.Printf("error: couldn't get absolute path to default directory: %v", err)
		return subcommands.ExitFailure
	}

	if exists, err := libbuildpack.FileExists(i.dir); err != nil {
		return subcommands.ExitFailure
	} else if exists {
		log.Printf("error: directory %s already exists", i.dir)
		return subcommands.ExitUsageError
	}

	if err := packager.Scaffold(i.dir, i.name); err != nil {
		log.Printf("Error creating new buildpack scaffolding: %v", err)
		return subcommands.ExitFailure
	}

	return subcommands.ExitSuccess
}

type upgradeCmd struct {
	dir   string
	force bool
}

func (*upgradeCmd) Name() string { return "upgrade" }
func (*upgradeCmd) Synopsis() string {
	return "Upgrades a buildpack scaffolded by buildpack-packager init"
}
func (u *upgradeCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&u.dir, "path", ".", "Path to folder to create. Defaults to the current directory.")
	f.BoolVar(&u.force, "force", false, "Regenerate files even if they have been modified")
}
func (*upgradeCmd) Usage() string {
	return `upgrade:
	Update an existing buildpack with changes made to scaffolding code.
`
}
func (u *upgradeCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	var err error
	u.dir, err = filepath.Abs(u.dir)
	if err != nil {
		log.Printf("error: couldn't get absolute path to default directory: %v", err)
		return subcommands.ExitFailure
	}

	if exists, err := libbuildpack.FileExists(u.dir); err != nil {
		return subcommands.ExitFailure
	} else if !exists {
		log.Printf("error: directory %s does not exist", u.dir)
		return subcommands.ExitUsageError
	}

	if err := packager.Upgrade(u.dir, u.force); err != nil {
		log.Printf("Error upgrading buildpack: %v", err)
		return subcommands.ExitFailure
	}

	return subcommands.ExitSuccess
}
func main() {
	subcommands.Register(subcommands.HelpCommand(), "")
	subcommands.Register(subcommands.FlagsCommand(), "")
	subcommands.Register(subcommands.CommandsCommand(), "")
	subcommands.Register(&summaryCmd{}, "Custom")
	subcommands.Register(&buildCmd{}, "Custom")
	subcommands.Register(&initCmd{}, "Custom")
	subcommands.Register(&upgradeCmd{}, "Custom")

	flag.Parse()
	ctx := context.Background()
	os.Exit(int(subcommands.Execute(ctx)))
}
