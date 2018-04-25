package finalize_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestFinalize(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Finalize Suite")
}
