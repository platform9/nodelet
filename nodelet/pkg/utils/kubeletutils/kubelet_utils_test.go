package kubeletutils

import (
	"context"
	"fmt"
	"github.com/golang/mock/gomock"
	"github.com/platform9/nodelet/nodelet/mocks"
	"github.com/platform9/nodelet/nodelet/pkg/utils/config"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
	"github.com/platform9/nodelet/nodelet/pkg/utils/netutils"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
)

func TestCommand(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("junit.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Phases Suite", []Reporter{junitReporter})
}

var _ = Describe("Test Kubelet Utils", func() {
	var (
		ctx              context.Context
		mockCtrl         *gomock.Controller
		fakeKubeletUtils KubeletImpl
		fakeCfg          *config.Config
		fakeCmd          *mocks.MockCLI
		fakeNetUtils     *mocks.MockNetInterface
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		fakeCmd = mocks.NewMockCLI(mockCtrl)
		fakeKubeletUtils.Cmd = fakeCmd
		fakeNetUtils = mocks.NewMockNetInterface(mockCtrl)
		fakeKubeletUtils.NetUtils = fakeNetUtils
		ctx = context.TODO()
		// Setup Config
		var err error
		fakeCfg, err = config.GetDefaultConfig()
		assert.Nil(GinkgoT(), err)
	})

	AfterEach(func() {
		mockCtrl.Finish()
		ctx.Done()
	})

	Context("validate ensure kubelet stopped method", func() {
		It("should fail with error if pf9-kubelet is running and kubelet stop returns error", func() {
			fakeErr := fmt.Errorf("fake error")
			fakeStdErr := []string{"fakeStderr"}
			fakeStdOut := []string{"fakeStdout"}
			fakeCmd.EXPECT().RunCommandWithStdOutStdErr(context.Background(), nil, 0, "", "sudo", "systemctl", "is-active", "pf9-kubelet").Return(0, fakeStdOut, fakeStdErr, fakeErr).Times(1)
			fakeCmd.EXPECT().RunCommandWithStdErr(context.Background(), nil, 0, "", "sudo", "systemctl", "stop", "pf9-kubelet").Return(0, fakeStdErr, fakeErr).Times(1)

			retErr := fakeKubeletUtils.EnsureKubeletStopped()
			Expect(retErr).ToNot(BeNil())
			Expect(retErr).To(BeIdenticalTo(fakeErr))
		})

		It("should succeed without error if pf9-kubelet is running and kubelet stop does not return error", func() {
			fakeErr := fmt.Errorf("fake error")
			fakeStdErr := []string{""}
			fakeStdOut := []string{"fakeStdout"} // anything but inactive which indicates that kubelet is inactive
			fakeCmd.EXPECT().RunCommandWithStdOutStdErr(context.Background(), nil, 0, "", "sudo", "systemctl", "is-active", "pf9-kubelet").Return(0, fakeStdOut, fakeStdErr, fakeErr).Times(1)
			fakeCmd.EXPECT().RunCommandWithStdErr(context.Background(), nil, 0, "", "sudo", "systemctl", "stop", "pf9-kubelet").Return(0, fakeStdErr, nil).Times(1)

			retErr := fakeKubeletUtils.EnsureKubeletStopped()
			Expect(retErr).To(BeNil())
		})

		It("should succeed without error if pf9-kubelet is not running", func() {
			fakeErrStr := "fake error"
			fakeErr := fmt.Errorf(fakeErrStr)
			fakeStdOut := []string{"inactive"}
			fakeStdErr := []string{"fake stderr"}
			fakeCmd.EXPECT().RunCommandWithStdOutStdErr(context.Background(), nil, 0, "", "sudo", "systemctl", "is-active", "pf9-kubelet").Return(0, fakeStdOut, fakeStdErr, fakeErr).Times(1)
			ret := fakeKubeletUtils.EnsureKubeletStopped()
			Expect(ret).To(BeNil())
		})

	})

	Context("validate fetch aws instance id", func() {
		It("should fail with error if unable to read aws instance id file", func() {
			constants.AWSInstanceIdLoc = "testdata/dir-does-not-exist/aws-instance-id"
			instanceId, err := fakeKubeletUtils.FetchAwsInstanceId()
			Expect(err).ToNot(BeNil())
			Expect(instanceId).To(BeEmpty())
		})

		It("should succeed without error if able to read aws instance id file", func() {
			constants.AWSInstanceIdLoc = "testdata/aws-instance-id"
			instanceId, err := fakeKubeletUtils.FetchAwsInstanceId()
			Expect(err).To(BeNil())
			Expect(instanceId).To(BeIdenticalTo("test-data"))
		})
	})

	Context("validate TrimSans method", func() {
		It("should remove new lines from string containing only newlines and no spaces", func() {
			strWithNewLines := "this\nis\na\nstring\ncontaining\nonly\nnew\nlines\n"
			strWithOutNewLines := "thisisastringcontainingonlynewlines"
			strAfterRemovingNewlines := fakeKubeletUtils.TrimSans(strWithNewLines)
			Expect(strAfterRemovingNewlines).NotTo(BeEmpty())
			Expect(strAfterRemovingNewlines).To(BeIdenticalTo(strWithOutNewLines))
		})

		It("should remove spaces from string containing only spaces and no new lines", func() {
			strWithSpaces := " this is a string containing only new lines"
			strWithOutSpaces := "thisisastringcontainingonlynewlines"
			strAfterRemovingSpaces := fakeKubeletUtils.TrimSans(strWithSpaces)
			Expect(strAfterRemovingSpaces).NotTo(BeEmpty())
			Expect(strAfterRemovingSpaces).To(BeIdenticalTo(strWithOutSpaces))
		})

		It("should remove spaces and newlines from string containing both spaces and newlines", func() {
			strWithNewLinesAndSpaces := "this is\na string\ncontaining\nonly\nnew lines "
			strWithOutNewLinesAndSpaces := "thisisastringcontainingonlynewlines"
			strAfterRemovingNewlinesAndSpaces := fakeKubeletUtils.TrimSans(strWithNewLinesAndSpaces)
			Expect(strAfterRemovingNewlinesAndSpaces).NotTo(BeEmpty())
			Expect(strAfterRemovingNewlinesAndSpaces).To(BeIdenticalTo(strWithOutNewLinesAndSpaces))
		})

		It("should not remove anything from strings not containing newlines and spaces", func() {
			strWithOutNewLines := "thisisastringcontainingonlynewlines"
			strWithOutNewLinesAndSpaces := "thisisastringcontainingonlynewlines"
			strAfterRemovingNewlinesAndSpaces := fakeKubeletUtils.TrimSans(strWithOutNewLinesAndSpaces)
			Expect(strAfterRemovingNewlinesAndSpaces).NotTo(BeEmpty())
			Expect(strAfterRemovingNewlinesAndSpaces).To(BeIdenticalTo(strWithOutNewLines))
		})
	})

	Context("validate prepare kubelet bootstrap config", func() {
		BeforeEach(func() {
			constants.KubeletConfigDir = "testdata" + constants.KubeletConfigDir
			constants.KubeletBootstrapConfig = "testdata" + constants.KubeletBootstrapConfig
		})
		AfterEach(func() {
			err := os.RemoveAll("testdata/var")
			Expect(err).To(BeNil())
		})

		It("should succeed without error and prepare default kubelet bootstrap config", func() {
			expectedDnsIp, err := netutils.New().AddrConv(fakeCfg.ServicesCIDR, 10)
			Expect(err).To(BeNil())
			expectedKubeletBootstrapConfig := "apiVersion: kubelet.config.k8s.io/v1beta1\n" +
				"kind: KubeletConfiguration\n" +
				"address: 0.0.0.0\n" +
				"authentication:\n" +
				"  anonymous:\n" +
				"    enabled: false\n" +
				"  webhook:\n" +
				"    enabled: true\n" +
				"  x509:\n" +
				"    clientCAFile: " + constants.KubeletClientCaFile + "\n" +
				"authorization:\n" +
				"  mode: AlwaysAllow\n" +
				"clusterDNS:\n" +
				"- " + expectedDnsIp + "\n" +
				"clusterDomain: " + constants.DnsDomain + "\n" +
				"cpuManagerPolicy: " + fakeCfg.CPUManagerPolicy + "\n" +
				"topologyManagerPolicy: " + fakeCfg.TopologyManagerPolicy + "\n" +
				"reservedSystemCPUs: " + fakeCfg.ReservedCPUs + "\n" +
				"featureGates:\n" +
				"  DynamicKubeletConfig: true\n" +
				"maxPods: 200\n" +
				"readOnlyPort: 0\n" +
				"tlsCertFile: " + constants.KubeletTlsCertFile + "\n" +
				"tlsPrivateKeyFile: " + constants.KubeletTlsPrivateKeyFile + "\n" +
				"tlsCipherSuites: " + constants.KubeletTlsCipherSuites + "\n"

			if fakeCfg.ContainerdCgroup == "systemd" {
				expectedKubeletBootstrapConfig += "cgroupDriver: systemd\n"
			} else {
				expectedKubeletBootstrapConfig += "cgroupDriver: cgroupfs\n"
			}
			if fakeCfg.ClusterRole == "master" {
				expectedKubeletBootstrapConfig += "staticPodPath: " + constants.KubeletStaticPodPath + "\n"
			}
			if fakeCfg.AllowSwap {
				expectedKubeletBootstrapConfig += "failSwapOn: false\n"
			}

			fakeStdOut := []string{"fake stdout"}
			fakeStdErr := []string{"fake stderr"}
			usrgrp := constants.Pf9User + ":" + constants.Pf9Group
			dir := constants.KubeletConfigDir
			fakeCmd.EXPECT().RunCommandWithStdOutStdErr(context.Background(), nil, 0, "", "sudo", "chown", "-R", usrgrp, dir).Return(0, fakeStdOut, fakeStdErr, nil).Times(1)
			fakeNetUtils.EXPECT().AddrConv(fakeCfg.ServicesCIDR, 10).Return(expectedDnsIp, nil)

			// fakeCfg loads the default config
			err = fakeKubeletUtils.PrepareKubeletBootstrapConfig(*fakeCfg)
			Expect(err).To(BeNil())

			Expect(constants.KubeletBootstrapConfig).To(BeAnExistingFile())
			actualKubeletBootstrapConfig, err := os.ReadFile(constants.KubeletBootstrapConfig)
			Expect(err).To(BeNil())
			Expect(string(actualKubeletBootstrapConfig)).To(BeIdenticalTo(expectedKubeletBootstrapConfig))
		})

		It("should succeed without error and prepare kubelet bootstrap config with cgroupdriver as cgroupfs, clusterrole as master and allow swap", func() {
			// Config customized to change the following
			fakeCfg.ContainerdCgroup = "test"
			fakeCfg.ClusterRole = "master"
			fakeCfg.AllowSwap = true

			expectedDnsIp, err := netutils.New().AddrConv(fakeCfg.ServicesCIDR, 10)
			Expect(err).To(BeNil())
			expectedKubeletBootstrapConfig := "apiVersion: kubelet.config.k8s.io/v1beta1\n" +
				"kind: KubeletConfiguration\n" +
				"address: 0.0.0.0\n" +
				"authentication:\n" +
				"  anonymous:\n" +
				"    enabled: false\n" +
				"  webhook:\n" +
				"    enabled: true\n" +
				"  x509:\n" +
				"    clientCAFile: " + constants.KubeletClientCaFile + "\n" +
				"authorization:\n" +
				"  mode: AlwaysAllow\n" +
				"clusterDNS:\n" +
				"- " + expectedDnsIp + "\n" +
				"clusterDomain: " + constants.DnsDomain + "\n" +
				"cpuManagerPolicy: " + fakeCfg.CPUManagerPolicy + "\n" +
				"topologyManagerPolicy: " + fakeCfg.TopologyManagerPolicy + "\n" +
				"reservedSystemCPUs: " + fakeCfg.ReservedCPUs + "\n" +
				"featureGates:\n" +
				"  DynamicKubeletConfig: true\n" +
				"maxPods: 200\n" +
				"readOnlyPort: 0\n" +
				"tlsCertFile: " + constants.KubeletTlsCertFile + "\n" +
				"tlsPrivateKeyFile: " + constants.KubeletTlsPrivateKeyFile + "\n" +
				"tlsCipherSuites: " + constants.KubeletTlsCipherSuites + "\n"

			if fakeCfg.ContainerdCgroup == "systemd" {
				expectedKubeletBootstrapConfig += "cgroupDriver: systemd\n"
			} else {
				expectedKubeletBootstrapConfig += "cgroupDriver: cgroupfs\n"
			}
			if fakeCfg.ClusterRole == "master" {
				expectedKubeletBootstrapConfig += "staticPodPath: " + constants.KubeletStaticPodPath + "\n"
			}
			if fakeCfg.AllowSwap {
				expectedKubeletBootstrapConfig += "failSwapOn: false\n"
			}

			fakeStdOut := []string{"fake stdout"}
			fakeStdErr := []string{"fake stderr"}
			usrgrp := constants.Pf9User + ":" + constants.Pf9Group
			dir := constants.KubeletConfigDir
			fakeCmd.EXPECT().RunCommandWithStdOutStdErr(context.Background(), nil, 0, "", "sudo", "chown", "-R", usrgrp, dir).Return(0, fakeStdOut, fakeStdErr, nil).Times(1)
			fakeNetUtils.EXPECT().AddrConv(fakeCfg.ServicesCIDR, 10).Return(expectedDnsIp, nil)

			err = fakeKubeletUtils.PrepareKubeletBootstrapConfig(*fakeCfg)
			Expect(err).To(BeNil())

			Expect(constants.KubeletBootstrapConfig).To(BeAnExistingFile())
			actualKubeletBootstrapConfig, err := os.ReadFile(constants.KubeletBootstrapConfig)
			Expect(err).To(BeNil())
			Expect(string(actualKubeletBootstrapConfig)).To(BeIdenticalTo(expectedKubeletBootstrapConfig))
		})
	})

	Context("validate kubelet setup", func() {
		constants.KubeEnvPath = "testdata" + constants.KubeEnvPath
		constants.Pf9KubeletSystemdUnitTemplate = "testdata" + constants.Pf9KubeletSystemdUnitTemplate

		constants.KubeletEnvPath = "testdata" + constants.KubeletEnvPath
		constants.SystemdRuntimeUnitDir = "testdata" + constants.SystemdRuntimeUnitDir

		It("should succeed without error and do kubelet setup", func() {
			err := os.MkdirAll(constants.SystemdRuntimeUnitDir, 0777)
			Expect(err).To(BeNil())

			fakeKubeletArgs := "fakeKubeletArgs"
			fakeStdOut := []string{"fake stdout"}
			fakeStdErr := []string{"fake stderr"}
			fakeCmd.EXPECT().RunCommandWithStdErr(context.Background(), nil, 0, "", "sudo", "touch", constants.SystemdRuntimeUnitDir+"/pf9-kubelet.service").Return(0, fakeStdErr, nil).Times(1)
			usrgrp := constants.Pf9User + ":" + constants.Pf9Group
			fakeCmd.EXPECT().RunCommandWithStdOutStdErr(context.Background(), nil, 0, "", "sudo", "chown", usrgrp, constants.SystemdRuntimeUnitDir+"/pf9-kubelet.service").Return(0, fakeStdOut, fakeStdErr, nil).Times(1)
			fakeCmd.EXPECT().RunCommandWithStdOutStdErr(context.Background(), nil, 0, "", "sudo", "chmod", "770", constants.SystemdRuntimeUnitDir+"/pf9-kubelet.service").Return(0, fakeStdOut, fakeStdErr, nil).Times(1)

			fakeCmd.EXPECT().RunCommandWithStdErr(ctx, nil, 0, "", "sudo", "systemctl", "daemon-reload").Return(0, nil, nil).Times(1)

			err = fakeKubeletUtils.KubeletSetup(fakeKubeletArgs)
			Expect(err).To(BeNil())

			err = os.Remove(constants.KubeletEnvPath)
			Expect(err).To(BeNil())
			err = os.RemoveAll("testdata/run")
			Expect(err).To(BeNil())
		})

		It("should fail with error if generation of kubelet systemd unit fails and returns err", func() {
			fakeKubeletArgs := "fakeKubeletArgs"
			fakeErr := fmt.Errorf("fakeErr")
			fakeStdErr := []string{"fake stderr"}
			fakeCmd.EXPECT().RunCommandWithStdErr(context.Background(), nil, 0, "", "sudo", "touch", constants.SystemdRuntimeUnitDir+"/pf9-kubelet.service").Return(0, fakeStdErr, fakeErr).Times(1)

			retErr := fakeKubeletUtils.KubeletSetup(fakeKubeletArgs)
			Expect(retErr).ToNot(BeNil())
			Expect(retErr).To(BeIdenticalTo(fakeErr))

			err := os.Remove(constants.KubeletEnvPath)
			Expect(err).To(BeNil())
		})

		It("should fail with error if it fails to reload the daemon", func() {
			err := os.MkdirAll(constants.SystemdRuntimeUnitDir, 0777)
			Expect(err).To(BeNil())

			fakeKubeletArgs := "fakeKubeletArgs"
			fakeErr := fmt.Errorf("fake error")
			fakeStdOut := []string{"fake stdout"}
			fakeStdErr := []string{"fake stderr"}
			fakeCmd.EXPECT().RunCommandWithStdErr(context.Background(), nil, 0, "", "sudo", "touch", constants.SystemdRuntimeUnitDir+"/pf9-kubelet.service").Return(0, fakeStdErr, nil).Times(1)
			usrgrp := constants.Pf9User + ":" + constants.Pf9Group
			fakeCmd.EXPECT().RunCommandWithStdOutStdErr(context.Background(), nil, 0, "", "sudo", "chown", usrgrp, constants.SystemdRuntimeUnitDir+"/pf9-kubelet.service").Return(0, fakeStdOut, fakeStdErr, nil).Times(1)
			fakeCmd.EXPECT().RunCommandWithStdOutStdErr(context.Background(), nil, 0, "", "sudo", "chmod", "770", constants.SystemdRuntimeUnitDir+"/pf9-kubelet.service").Return(0, fakeStdOut, fakeStdErr, nil).Times(1)

			fakeCmd.EXPECT().RunCommandWithStdErr(ctx, nil, 0, "", "sudo", "systemctl", "daemon-reload").Return(0, fakeStdErr, fakeErr).Times(1)

			retErr := fakeKubeletUtils.KubeletSetup(fakeKubeletArgs)
			Expect(retErr).ToNot(BeNil())
			Expect(retErr).To(BeIdenticalTo(fakeErr))

			err = os.Remove(constants.KubeletEnvPath)
			Expect(err).To(BeNil())
			err = os.RemoveAll("testdata/run")
			Expect(err).To(BeNil())
		})
	})

})
