package kubeutils_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"

	"github.com/platform9/nodelet/nodelet/pkg/utils/kubeutils"
)

func TestCommand(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("junit.xml")
	//RunSpecs(t, "Fileio Suite")
	RunSpecsWithDefaultAndCustomReporters(t, "kubeutils Suite", []Reporter{junitReporter})
}

var _ = Describe("kubeutils", func() {
	Context("Checking IP type", func() {
		var (
			ipv4    string
			ipv6    string
			outipv6 string
			ipnull  string
		)

		BeforeEach(func() {
			ipv4 = "10.126.2.34"
			ipv6 = "2001:db8::2:1"
			outipv6 = "[2001:db8::2:1]"
			ipnull = ""

		})
		It("it should correctly identify ipv4", func() {
			ip, err := kubeutils.Ip_for_http(ipv4)
			Expect(err).To(BeNil())
			Expect(ip).To(Equal(ipv4))
		})

		It("it should correctly identify ipv6", func() {
			ip, err := kubeutils.Ip_for_http(ipv6)
			Expect(err).To(BeNil())
			Expect(ip).To(Equal(outipv6))
		})

		It("it should give error", func() {
			ip, err := kubeutils.Ip_for_http(ipnull)
			Expect(err).NotTo(BeNil())
			Expect(ip).To(Equal(ipnull))
		})

	})

})
