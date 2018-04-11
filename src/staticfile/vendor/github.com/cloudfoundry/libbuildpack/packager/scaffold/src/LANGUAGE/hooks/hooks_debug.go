package hooks

import (
	"fmt"
	"os"

	"github.com/cloudfoundry/libbuildpack"
)

type hooks1 struct {
	libbuildpack.DefaultHook
}

type hooks2 struct {
	libbuildpack.DefaultHook
}

func init() {
	if os.Getenv("BP_DEBUG") != "" {
		libbuildpack.AddHook(hooks1{})
		libbuildpack.AddHook(hooks2{})
	}
}

func (h hooks1) BeforeCompile(compiler *libbuildpack.Stager) error {
	fmt.Println("HOOKS 1: BeforeCompile")
	return nil
}

func (h hooks2) AfterCompile(compiler *libbuildpack.Stager) error {
	fmt.Println("HOOKS 2: AfterCompile")
	return nil
}
