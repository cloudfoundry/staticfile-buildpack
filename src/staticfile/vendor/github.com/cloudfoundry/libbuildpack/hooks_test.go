package libbuildpack_test

import (
	"errors"

	bp "github.com/cloudfoundry/libbuildpack"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

//go:generate mockgen -source=hooks.go --destination=mocks_hooks_test.go --package=libbuildpack_test --imports=.=github.com/cloudfoundry/libbuildpack

var _ = Describe("Hooks", func() {
	var (
		mockCtrl   *gomock.Controller
		mockHook   *MockHook
		mockStager *bp.Stager
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockHook = NewMockHook(mockCtrl)
		mockStager = &bp.Stager{}
	})

	AfterEach(func() {
		bp.ClearHooks()
	})

	Describe("RunBeforeCompile", func() {
		It("Runs BeforeCompile on an added hook", func() {
			mockHook.EXPECT().BeforeCompile(mockStager)
			bp.AddHook(mockHook)
			err := bp.RunBeforeCompile(mockStager)
			Expect(err).To(Succeed())
		})

		It("Returns errors", func() {
			mockHook.EXPECT().BeforeCompile(mockStager).Return(errors.New("err"))
			bp.AddHook(mockHook)
			err := bp.RunBeforeCompile(mockStager)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("RunAfterCompile", func() {
		It("Runs AfterCompile on an added hook", func() {
			mockHook.EXPECT().AfterCompile(mockStager)
			bp.AddHook(mockHook)
			err := bp.RunAfterCompile(mockStager)
			Expect(err).To(Succeed())
		})

		It("Returns errors", func() {
			mockHook.EXPECT().AfterCompile(mockStager).Return(errors.New("err"))
			bp.AddHook(mockHook)
			err := bp.RunAfterCompile(mockStager)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("DefaultHook", func() {
		It("fulfils Hook interface", func() {
			var hook bp.Hook
			hook = bp.DefaultHook{}
			Expect(hook).ToNot(BeNil())
		})
	})
})
