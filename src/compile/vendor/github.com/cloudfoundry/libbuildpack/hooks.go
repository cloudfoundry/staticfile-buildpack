package libbuildpack

import (
	"sync"
)

type Hook interface {
	BeforeCompile(*Compiler) error
	AfterCompile(*Compiler) error
}

var hookArray []Hook
var hookArrayLock sync.Mutex

func AddHook(hook Hook) {
	hookArrayLock.Lock()
	hookArray = append(hookArray, hook)
	hookArrayLock.Unlock()
}
func ClearHooks() {
	hookArrayLock.Lock()
	hookArray = make([]Hook, 0)
	hookArrayLock.Unlock()
}

func RunBeforeCompile(compiler *Compiler) error {
	for _, hook := range hookArray {
		if err := hook.BeforeCompile(compiler); err != nil {
			return err
		}
	}
	return nil
}

func RunAfterCompile(compiler *Compiler) error {
	for _, hook := range hookArray {
		if err := hook.AfterCompile(compiler); err != nil {
			return err
		}
	}
	return nil
}

type DefaultHook struct {}
func (d DefaultHook) BeforeCompile(compiler *Compiler) error { return nil }
func (d DefaultHook) AfterCompile(compiler *Compiler) error { return nil }
