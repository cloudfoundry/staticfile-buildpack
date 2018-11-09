package shims_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestShims(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Shims Suite")
}
