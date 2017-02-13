package main_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestCompile(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Compile Suite")
}
