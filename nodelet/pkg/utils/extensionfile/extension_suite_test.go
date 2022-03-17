package extensionfile_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"

	"github.com/platform9/nodelet/mocks"
	"github.com/platform9/nodelet/nodelet/pkg/utils/config"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
	"github.com/platform9/nodelet/nodelet/pkg/utils/extensionfile"
	"github.com/platform9/nodelet/nodelet/pkg/utils/fileio"
)

func TestCommand(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("junit.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Tasks Suite", []Reporter{junitReporter})
}

func interfaceToExtensionObj(input interface{}, output *extensionfile.ExtensionData) {
	dataByteArr, _ := input.([]byte)
	err := json.Unmarshal(dataByteArr, &output)
	assert.Nil(GinkgoT(), err)
}

var _ = Describe("Test extensionfile.go", func() {
	var (
		mockCtrl                       *gomock.Controller
		mockFile                       *mocks.MockFileInterface
		fakeClusterID                  string               = "fake-cluster-id"
		fakeClusterRole                string               = constants.RoleMaster
		pristineJSON                   string               = "testdata/extension.json.pristine"
		pristineOld                    string               = "testdata/extension.old.pristine"
		misconfiguredOld               string               = "testdata/extension.old.misconfigured.pristine"
		pristineConverted              string               = "testdata/converted.json.pristine"
		pristineConvertedMisconfigured string               = "testdata/converted.json.misconfigured.pristine"
		fileHandler                    fileio.FileInterface = fileio.New()
		failedCheck                    string               = "fake check"
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockFile = mocks.NewMockFileInterface(mockCtrl)
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	It("Is loading JSON extension file", func() {
		extFile := "testdata/extension.json"
		fileHandler.CopyFile(pristineJSON, extFile)
		extn := extensionfile.New(fileio.New(), extFile, &config.DefaultConfig)
		err := extn.Load()
		assert.Nil(GinkgoT(), err)
		expected, _ := fileHandler.ReadFile(pristineJSON)
		actual, _ := fileHandler.ReadFile(extFile)
		assert.Equal(GinkgoT(), string(expected), string(actual))
		_ = fileHandler.DeleteFile(extFile)
	})

	It("Is loading extension file of old format", func() {
		extFile := "testdata/extension.old"
		fileHandler.CopyFile(pristineOld, extFile)
		extn := extensionfile.New(fileio.New(), extFile, &config.DefaultConfig)
		err := extn.Load()
		assert.Nil(GinkgoT(), err)
		expected, err := fileHandler.ReadFile(pristineConverted)
		Expect(err).To(BeNil())
		actual, err := fileHandler.ReadFile(extFile)
		Expect(err).To(BeNil())
		assert.Equal(GinkgoT(), string(expected), string(actual))
		_ = fileHandler.DeleteFile(extFile)
	})

	It("Is loading extension file of old format with incorrect datatype for some values", func() {
		// Following list contains the issues and their expected new values
		// ClusterID = \"\" ==> ClusterID = ""
		// StartAttempts = "a" ==> StartAttempts = 0
		extFile := "testdata/misconfigured.old"
		fileHandler.CopyFile(misconfiguredOld, extFile)
		extn := extensionfile.New(fileio.New(), extFile, &config.DefaultConfig)
		err := extn.Load()
		assert.Nil(GinkgoT(), err)
		expected, err := fileHandler.ReadFile(pristineConvertedMisconfigured)
		Expect(err).To(BeNil())
		actual, err := fileHandler.ReadFile(extFile)
		Expect(err).To(BeNil())
		assert.Equal(GinkgoT(), string(expected), string(actual))
		_ = fileHandler.DeleteFile(extFile)
	})

	It("Is loading non-existent extension data file", func() {
		// This should get us a near-blank extension data struct instance
		// and write a new file. For that reason check for error while deleting the test file.
		extFile := "testdata/non-existent.json"
		extn := extensionfile.New(fileHandler, extFile, &config.DefaultConfig)
		err := extn.Load()
		assert.Nil(GinkgoT(), err)
		err = fileHandler.DeleteFile(extFile)
		assert.Nil(GinkgoT(), err)
	})

	It("Is writing extension file with service state ok", func() {
		extn := extensionfile.New(mockFile, constants.ExtensionOutputFile, &config.DefaultConfig)
		extn.StartAttempts = 0
		extn.ServiceState = constants.ServiceStateTrue
		extn.ClusterID = fakeClusterID
		extn.ClusterRole = fakeClusterRole
		mockFile.EXPECT().WriteToFile(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Do(
			func(filename string, data interface{}, append bool) error {
				assert.Equal(GinkgoT(), constants.ExtensionOutputFile, filename)
				assert.Equal(GinkgoT(), false, append)
				actual := extensionfile.New(nil, constants.ExtensionOutputFile, &config.DefaultConfig)
				interfaceToExtensionObj(data, &actual)
				assert.Equal(GinkgoT(), actual.ClusterID, extn.ClusterID)
				assert.Equal(GinkgoT(), actual.ClusterRole, extn.ClusterRole)
				assert.Equal(GinkgoT(), actual.NodeState, constants.OkState)
				return nil
			})
		extn.Write()
	})

	It("Is writing extension file with service state ok and stopped", func() {
		extn := extensionfile.New(mockFile, constants.ExtensionOutputFile, &config.DefaultConfig)
		extn.StartAttempts = 0
		extn.ServiceState = constants.ServiceStateFalse
		extn.ClusterID = fakeClusterID
		extn.ClusterRole = fakeClusterRole
		extn.Cfg.KubeServiceState = constants.ServiceStateFalse
		extn.KubeRunning = false
		mockFile.EXPECT().WriteToFile(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Do(
			func(filename string, data interface{}, append bool) error {
				assert.Equal(GinkgoT(), constants.ExtensionOutputFile, filename)
				assert.Equal(GinkgoT(), false, append)
				actual := extensionfile.New(nil, constants.ExtensionOutputFile, &config.DefaultConfig)
				interfaceToExtensionObj(data, &actual)
				assert.Equal(GinkgoT(), actual.ClusterID, extn.ClusterID)
				assert.Equal(GinkgoT(), actual.ClusterRole, extn.ClusterRole)
				assert.Equal(GinkgoT(), actual.NodeState, constants.OkState)
				return nil
			})
		extn.Write()
	})

	It("Is writing extension file with service state converging", func() {
		extn := extensionfile.New(mockFile, constants.ExtensionOutputFile, &config.DefaultConfig)
		extn.StartAttempts = 1
		extn.ServiceState = constants.ServiceStateFalse
		extn.ClusterID = fakeClusterID
		extn.ClusterRole = fakeClusterRole
		extn.Cfg.KubeServiceState = constants.ServiceStateTrue
		extn.KubeRunning = false
		mockFile.EXPECT().WriteToFile(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Do(
			func(filename string, data interface{}, append bool) error {
				assert.Equal(GinkgoT(), constants.ExtensionOutputFile, filename)
				assert.Equal(GinkgoT(), false, append)
				actual := extensionfile.New(nil, constants.ExtensionOutputFile, &config.DefaultConfig)
				interfaceToExtensionObj(data, &actual)
				assert.Equal(GinkgoT(), actual.ClusterID, extn.ClusterID)
				assert.Equal(GinkgoT(), actual.ClusterRole, extn.ClusterRole)
				assert.Equal(GinkgoT(), actual.NodeState, constants.ConvergingState)
				return nil
			})
		extn.Write()
	})

	It("Is writing extension file with service state retrying", func() {
		extn := extensionfile.New(mockFile, constants.ExtensionOutputFile, &config.DefaultConfig)
		extn.StartAttempts = 3
		extn.ServiceState = constants.ServiceStateFalse
		extn.ClusterID = fakeClusterID
		extn.ClusterRole = fakeClusterRole
		extn.Cfg.KubeServiceState = constants.ServiceStateTrue
		extn.KubeRunning = false
		mockFile.EXPECT().WriteToFile(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Do(
			func(filename string, data interface{}, append bool) error {
				assert.Equal(GinkgoT(), constants.ExtensionOutputFile, filename)
				assert.Equal(GinkgoT(), false, append)
				actual := extensionfile.New(nil, constants.ExtensionOutputFile, &config.DefaultConfig)
				interfaceToExtensionObj(data, &actual)
				assert.Equal(GinkgoT(), actual.ClusterID, extn.ClusterID)
				assert.Equal(GinkgoT(), actual.ClusterRole, extn.ClusterRole)
				assert.Equal(GinkgoT(), actual.NodeState, constants.RetryingState)
				return nil
			})
		extn.Write()
	})

	It("Is writing extension file with service state error", func() {
		extn := extensionfile.New(mockFile, constants.ExtensionOutputFile, &config.DefaultConfig)
		extn.StartAttempts = 11
		extn.ServiceState = constants.ServiceStateFalse
		extn.ClusterID = fakeClusterID
		extn.ClusterRole = fakeClusterRole
		extn.Cfg.KubeServiceState = constants.ServiceStateTrue
		extn.KubeRunning = false
		mockFile.EXPECT().WriteToFile(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Do(
			func(filename string, data interface{}, append bool) error {
				assert.Equal(GinkgoT(), constants.ExtensionOutputFile, filename)
				assert.Equal(GinkgoT(), false, append)
				actual := extensionfile.New(nil, constants.ExtensionOutputFile, &config.DefaultConfig)
				interfaceToExtensionObj(data, &actual)
				assert.Equal(GinkgoT(), actual.ClusterID, extn.ClusterID)
				assert.Equal(GinkgoT(), actual.ClusterRole, extn.ClusterRole)
				assert.Equal(GinkgoT(), actual.NodeState, constants.ErrorState)
				return nil
			})
		extn.Write()
	})

	It("Is writing extension file with failed check that should NOT be cleared", func() {
		fakeFailedCheckTime := time.Now().Unix()
		// fake the current time to be within the status reap interval
		fakeCurrTime := fakeFailedCheckTime + constants.FailedStatusCheckReapInterval - 1
		extn := extensionfile.New(mockFile, constants.ExtensionOutputFile, &config.DefaultConfig)
		extn.StartAttempts = 1
		extn.ServiceState = constants.ServiceStateFalse
		extn.Cfg.KubeServiceState = constants.ServiceStateFalse
		extn.KubeRunning = true
		extn.ClusterID = fakeClusterID
		extn.ClusterRole = fakeClusterRole
		extn.LastFailedCheck = failedCheck
		extn.CurrentStatusCheckTime = fakeCurrTime
		extn.LastFailedCheckTime = fakeFailedCheckTime
		mockFile.EXPECT().WriteToFile(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Do(
			func(filename string, data interface{}, append bool) error {
				assert.Equal(GinkgoT(), constants.ExtensionOutputFile, filename)
				assert.Equal(GinkgoT(), false, append)
				actual := extensionfile.New(nil, constants.ExtensionOutputFile, &config.DefaultConfig)
				interfaceToExtensionObj(data, &actual)
				assert.Equal(GinkgoT(), actual.ClusterID, extn.ClusterID)
				assert.Equal(GinkgoT(), actual.ClusterRole, extn.ClusterRole)
				assert.Equal(GinkgoT(), actual.NodeState, constants.ConvergingState)
				assert.Equal(GinkgoT(), actual.LastFailedCheck, failedCheck)
				assert.Equal(GinkgoT(), actual.LastFailedCheckTime, fakeFailedCheckTime)
				assert.Equal(GinkgoT(), actual.CurrentStatusCheckTime, fakeCurrTime)
				return nil
			})
		extn.Write()
	})

	It("Is writing extension file with failed check that should be cleared", func() {
		fakeFailedCheckTime := time.Now().Unix()
		// fake the current time to be after the status reap interval
		fakeCurrTime := fakeFailedCheckTime + constants.FailedStatusCheckReapInterval + 2
		extn := extensionfile.New(mockFile, constants.ExtensionOutputFile, &config.DefaultConfig)
		extn.StartAttempts = 1
		extn.ServiceState = constants.ServiceStateFalse
		extn.Cfg.KubeServiceState = constants.ServiceStateFalse
		extn.KubeRunning = true
		extn.ClusterID = fakeClusterID
		extn.ClusterRole = fakeClusterRole
		extn.LastFailedCheck = failedCheck
		extn.CurrentStatusCheckTime = fakeCurrTime
		extn.LastFailedCheckTime = fakeFailedCheckTime
		mockFile.EXPECT().WriteToFile(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Do(
			func(filename string, data interface{}, append bool) error {
				assert.Equal(GinkgoT(), constants.ExtensionOutputFile, filename)
				assert.Equal(GinkgoT(), false, append)
				actual := extensionfile.New(nil, constants.ExtensionOutputFile, &config.DefaultConfig)
				interfaceToExtensionObj(data, &actual)
				assert.Equal(GinkgoT(), actual.ClusterID, extn.ClusterID)
				assert.Equal(GinkgoT(), actual.ClusterRole, extn.ClusterRole)
				assert.Equal(GinkgoT(), actual.NodeState, constants.ConvergingState)
				assert.Equal(GinkgoT(), actual.LastFailedCheck, "")
				assert.Equal(GinkgoT(), actual.LastFailedCheckTime, int64(0))
				assert.Equal(GinkgoT(), actual.CurrentStatusCheckTime, fakeCurrTime)
				return nil
			})
		extn.Write()
	})
})
