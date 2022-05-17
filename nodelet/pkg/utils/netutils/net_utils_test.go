package netutils

import (
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestCommand(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("junit.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Phases Suite", []Reporter{junitReporter})
}

var _ = Describe("Net Utils", func() {
	var (
		validIPv4 = "10.12.13.14"
		validIPv6 = "2001:db8::1234:5678"
		netImpl   = &NetImpl{}
		cidr      = "10.20.0.0/22"
	)

	Context("Validates IP ", func() {
		It("If ipv4 it returns as it is", func() {
			ip, err := netImpl.IpForHttp(validIPv4)
			assert.Nil(GinkgoT(), err)
			assert.Equal(GinkgoT(), ip, validIPv4)
		})
		It("If ipv6 it adds bracket", func() {
			ip, err := netImpl.IpForHttp(validIPv6)
			assert.Nil(GinkgoT(), err)
			assert.Equal(GinkgoT(), ip, "[2001:db8::1234:5678]")
		})
		It("Fails if invalid ip ", func() {
			err := errors.New("invalid IP")
			_, reterr := netImpl.IpForHttp("10.12.1314")
			assert.NotNil(GinkgoT(), err)
			assert.Equal(GinkgoT(), reterr.Error(), err.Error())
		})
	})
	Context("IP generation from CIDR ", func() {
		It("Generates ip successfully from CIDR", func() {
			ip, err := netImpl.AddrConv(cidr, 10)
			fmt.Printf("ip is:%s", ip)
			assert.Nil(GinkgoT(), err)
			assert.Equal(GinkgoT(), ip, "10.20.0.10")
		})
		It("fails to generate ip if pos is out of range of accomodation of prefix of CIDR", func() {
			ip, err := netImpl.AddrConv(cidr, 10000)
			fmt.Printf("ip is:%s", ip)
			assert.NotNil(GinkgoT(), err)
			assert.Equal(GinkgoT(), ip, "")
		})
		It("fails if inavlid CIDR", func() {
			ip, err := netImpl.AddrConv("10.20.30.40", 20)
			assert.NotNil(GinkgoT(), err)
			assert.Equal(GinkgoT(), ip, "")
		})
	})
})
