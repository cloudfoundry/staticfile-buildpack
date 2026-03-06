package finalize_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"testing"
)

func TestFinalize(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Finalize Suite")
}
