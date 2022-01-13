package config_test

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/ghodss/yaml"
	fuzz "github.com/google/gofuzz"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/platform9/nodelet/pkg/utils/config"
	"github.com/platform9/nodelet/pkg/utils/constants"
	sunpikev1alpha1 "github.com/platform9/pf9-qbert/sunpike/apiserver/pkg/apis/sunpike/v1alpha1"
	"github.com/platform9/pf9-qbert/sunpike/apiserver/pkg/hosts"
)

const (
	testDataDir       = "testdata/tmp"
	kubeEnvFieldCount = 76 // The number of fields in a kube.env file (that are generated from a HostSpec).
)

var _ = Describe("Test kubeenv.go", func() {
	fuzzer := fuzz.New()

	BeforeEach(func() {
		err := os.MkdirAll(testDataDir, 0755)
		Expect(err).To(BeNil())
	})

	AfterEach(func() {
		err := os.RemoveAll(testDataDir)
		Expect(err).To(BeNil())
	})

	Context("when formatting a KubeEnvMap to an kube.env file", func() {
		It("should generate a source-able file with a single field", func() {
			kubeEnvMap := config.KubeEnvMap{
				"FOO": "BAR",
			}
			tmpFile := path.Join(testDataDir, "single_kube.env")
			fd, err := os.Create(tmpFile)
			if err != nil {
				return
			}
			err = kubeEnvMap.ToKubeEnv(fd)
			Expect(err).To(BeNil())
			fd.Close()

			buf := bytes.NewBuffer(nil)
			cmd := exec.Command("sh", "-c", fmt.Sprintf("source %s && echo $FOO", tmpFile))
			cmd.Stdout = buf
			cmd.Stderr = os.Stderr
			err = cmd.Run()
			Expect(err).To(BeNil())
			Expect(strings.TrimSpace(buf.String())).To(Equal(kubeEnvMap["FOO"]))
		})

		It("should generate a source-able file regardless of the input", func() {
			host := &sunpikev1alpha1.Host{}
			fuzzer.Fuzz(host)
			host.Spec.ExtraCfg = map[string]string{
				// Avoid fuzzing of the key because that checking that is out-of-scope.
				"FOO": "bar",
			}

			kubeEnvMap := config.ConvertHostToKubeEnvMap(host)

			tmpFile := path.Join(testDataDir, "fuzzed_kube.env")
			fd, err := os.Create(tmpFile)
			if err != nil {
				return
			}
			err = kubeEnvMap.ToKubeEnv(fd)
			Expect(err).To(BeNil())
			fd.Close()

			cmd := exec.Command("sh", "-c", "source "+tmpFile)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err = cmd.Run()
			Expect(err).To(BeNil())
		})

		It("should generate a source-able file with multi-line variables", func() {
			kubeEnvMap := config.KubeEnvMap{
				"MULTILINE": `line 1
line 2
line 3`,
			}
			tmpFile := path.Join(testDataDir, "multiline_kube.env")
			fd, err := os.Create(tmpFile)
			if err != nil {
				return
			}
			err = kubeEnvMap.ToKubeEnv(fd)
			Expect(err).To(BeNil())
			fd.Close()

			cmd := exec.Command("sh", "-c", fmt.Sprintf("source %s", tmpFile))
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err = cmd.Run()
			Expect(err).To(BeNil())
		})
	})

	Context("when formatting a KubeEnvMap to a YAML file", func() {
		It("should generate a valid config.Config YAML regardless of the input", func() {
			host := &sunpikev1alpha1.Host{}
			fuzzer.Fuzz(host)
			host.Spec.ExtraCfg = map[string]string{
				// Avoid fuzzing of the key because that checking that is out-of-scope.
				"FOO": "bar",
			}

			kubeEnvMap := config.ConvertHostToKubeEnvMap(host)
			buf := bytes.NewBuffer(nil)
			err := kubeEnvMap.ToYAML(buf)
			Expect(err).To(BeNil())

			cfg := &config.Config{}
			err = yaml.Unmarshal(buf.Bytes(), cfg)
			Expect(err).To(BeNil())
			Expect(cfg).ToNot(BeZero())
		})
	})

	Context("when converting a Host to a KubeEnvMap", func() {

		It("should convert an empty Host", func() {
			kubeEnvMap := config.ConvertHostToKubeEnvMap(&sunpikev1alpha1.Host{})

			// The KubeEnvMap should have entries for each defined key, even if
			// they are all empty.
			Expect(kubeEnvMap).ToNot(BeEmpty())
			// Expect(len(kubeEnvMap)).To(Equal(kubeEnvFieldCount))
			for k, v := range kubeEnvMap {
				if v == "0" || v == "false" {
					// The zero values of other types are expected in string-form.
					continue
				}
				Expect(v).To(BeZero(), fmt.Sprintf("Field %s is not zero'ed and had the value %v", k, v))
			}
		})

		It("should convert a fuzzed Host", func() {
			host := &sunpikev1alpha1.Host{}
			fuzzer.Fuzz(host)
			kubeEnvMap := config.ConvertHostToKubeEnvMap(host)
			Expect(kubeEnvMap).ToNot(BeEmpty())
			//Expect(len(kubeEnvMap)).To(Equal(kubeEnvFieldCount + len(host.Spec.ExtraCfg)))
		})

		It("should correctly map the non-empty fields of a valid Host", func() {
			// Fill a couple of fields of different types and different locations
			// to see if the fields are converted as expected.
			host := &sunpikev1alpha1.Host{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-123",
				},
				Spec: sunpikev1alpha1.HostSpec{
					ExtraCfg: map[string]string{
						"FOO": "BAR",
					},
					PF9Cfg: sunpikev1alpha1.PF9Opts{
						ClusterID:   "CLUSTER-123",
						ClusterRole: constants.RoleMaster,
						Keystone: sunpikev1alpha1.KeystoneOpts{
							Enabled:  true,
							Password: "some-password",
						},
					},
					ClusterCfg: sunpikev1alpha1.KubeClusterOpts{
						Apiserver: sunpikev1alpha1.KubeApiserverOpts{
							Authz: false,
						},
					},
					ExtraOpts: "--arg1 val1 --arg2 val2",
				},
			}

			kubeEnvMap := config.ConvertHostToKubeEnvMap(host)
			Expect(kubeEnvMap[config.HostIDKey]).To(Equal(host.Name))
			Expect(kubeEnvMap["CLUSTER_ID"]).To(Equal(host.Spec.PF9Cfg.ClusterID))
			Expect(kubeEnvMap["ROLE"]).To(Equal(host.Spec.PF9Cfg.ClusterRole))
			Expect(kubeEnvMap["FOO"]).To(Equal(host.Spec.ExtraCfg["FOO"]))
			Expect(kubeEnvMap["AUTHZ_ENABLED"]).To(Equal("false"))
			Expect(kubeEnvMap["KEYSTONE_ENABLED"]).To(Equal("true"))
			Expect(kubeEnvMap["OS_PASSWORD"]).To(Equal(host.Spec.PF9Cfg.Keystone.Password))
			Expect(kubeEnvMap["EXTRA_OPTS"]).To(Equal(host.Spec.ExtraOpts))
		})

		It("should not override fields with fields from ExtraCfg", func() {
			host := &sunpikev1alpha1.Host{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-123",
				},
				Spec: sunpikev1alpha1.HostSpec{
					ExtraCfg: map[string]string{
						config.HostIDKey: "override-456",
					},
				},
			}

			kubeEnvMap := config.ConvertHostToKubeEnvMap(host)
			// Expect(len(kubeEnvMap)).To(Equal(kubeEnvFieldCount))
			Expect(kubeEnvMap[config.HostIDKey]).To(Equal(host.Name))
		})
	})

	Context("when converting a KubeEnvMap to a Nodelet Config", func() {
		It("should generate a valid config.Config regardless of the input", func() {
			host := &sunpikev1alpha1.Host{}
			fuzzer.Fuzz(host)

			cfg, err := config.ConvertHostToKubeEnvMap(host).ToConfig()
			Expect(err).To(BeNil())
			// Check if a couple of keys match
			Expect(cfg.HostID).To(Equal(host.Name))
			Expect(cfg.ClusterRole).To(Equal(host.Spec.PF9Cfg.ClusterRole))
		})

		It("should generate a valid config.Config for an empty HostSpec", func() {
			host := &sunpikev1alpha1.Host{}

			cfg, err := config.ConvertHostToKubeEnvMap(host).ToConfig()
			Expect(err).To(BeNil())
			cfg.Debug = "" // Exception, because the debug field is always filled because of the difference in type.
			Expect(*cfg).To(BeZero())
		})
	})

	Context("when converting a KubeEnvMap to a Host", func() {

		It("should generate a valid Host for an empty KubeEnvMap", func() {
			kubeEnv := config.KubeEnvMap{}

			host, err := kubeEnv.ToHost()
			Expect(err).To(BeNil())
			Expect(host).ToNot(BeNil())
			Expect(host.Spec.ExtraCfg).To(HaveLen(0))
			host.Spec.ExtraCfg = nil // ExtraCfg is always set, so set to nil to be able to do the check on the next line.
			Expect(host.Spec).To(BeZero())
		})

		It("should generate a valid Host for an empty KubeEnvMap", func() {
			kubeEnv := config.KubeEnvMap{
				config.HostIDKey:  hosts.GenerateID(),
				"AUTHZ_ENABLED":   "true",
				"MIN_NUM_WORKERS": "42",
				"OS_USERNAME":     "123abc",
				"ROLE":            constants.RoleWorker,
				"DEBUG":           "true",
				// ExtraCfg fields
				"FOO":      "BAR",
				"PLATFORM": "9",
			}

			host, err := kubeEnv.ToHost()
			Expect(err).To(BeNil())
			Expect(host.Name).To(Equal(kubeEnv[config.HostIDKey]))
			Expect(host.Spec.ExtraCfg).To(HaveLen(2))
			Expect(host.Spec.ExtraCfg["FOO"]).To(Equal(kubeEnv["FOO"]))
			Expect(host.Spec.ExtraCfg["PLATFORM"]).To(Equal("9"))
			Expect(host.Spec.ClusterCfg.Apiserver.Authz).To(BeTrue())
			Expect(host.Spec.PF9Cfg.ClusterRole).To(Equal(kubeEnv["ROLE"]))
			Expect(host.Spec.PF9Cfg.Debug).To(BeTrue())
			Expect(host.Spec.ClusterCfg.Addons.CAS.MinWorkers).To(Equal(int32(42)))
		})
	})
})
