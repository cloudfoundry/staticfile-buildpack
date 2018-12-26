package hooks_test

import (
	"testing"

	"github.com/kardolus/httpmock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = BeforeSuite(func() {
	httpmock.Activate()
})

var _ = AfterSuite(func() {
	httpmock.DeactivateAndReset()
})

func TestHooks(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Hooks Suite")
}
