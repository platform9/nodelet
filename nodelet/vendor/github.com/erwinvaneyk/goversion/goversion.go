package goversion

import (
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"reflect"
	"strings"
	"time"

	"github.com/ghodss/yaml"
)

const (
	GitTreeStateDirty = "dirty"
	GitTreeStateClean = "clean"
)

// Info contains all the version-related information.
//
// TODO add version parsing and comparing.
type Info struct {
	// Version is the semantic version of the application.
	Version string `json:"version"`

	// BuildDate contains the RFC3339 timestamp normalized to UTC of when the binary was built.
	BuildDate string `json:"buildDate"`

	// BuildArch is the system architecture that was used to build the binary.
	BuildArch string `json:"buildArch"`

	// BuildOS is the operating system that was used to build the binary.
	BuildOS string `json:"buildOS"`

	// BuildBy is a free-form field that contains info about who or what was responsible for the build.
	BuildBy string `json:"buildBy"`

	// GoVersion the go version that was used to build the binary.
	GoVersion string `json:"goVersion"`

	// GitCommit is the HEAD commit at the moment of building.
	GitCommit string `json:"gitCommit"`

	// GitCommitDate contains the RFC3339 timestamp normalized to UTC of the GitCommit.
	GitCommitDate string `json:"gitCommitDate"`

	// GitBranch is the git branch that was checked out at time of building.
	GitBranch string `json:"gitBranch"`

	// GitTreeState indicates whether there where uncommitted changes when the binary was built.
	//
	// If there uncommitted changes, this field will be "dirty". Otherwise, if
	// there are no uncommitted changes, this field will be "clean".
	GitTreeState string `json:"gitTreeState"`
}

func (i Info) IsEmpty() bool {
	infoType := reflect.ValueOf(i)
	for i := 0; i < infoType.NumField(); i++ {
		if !infoType.Field(i).IsZero() {
			return false
		}
	}
	return true
}

func (i Info) String() string {
	return i.ToJSON()
}

func (i Info) ToJSON() string {
	bs, err := json.Marshal(i)
	if err != nil {
		panic(err)
	}

	return string(bs)
}

func (i Info) ToYAML() string {
	bs, err := yaml.Marshal(i)
	if err != nil {
		panic(err)
	}

	return string(bs)
}

func (i Info) ToPrettyJSON() string {
	bs, err := json.MarshalIndent(i, "", "  ")
	if err != nil {
		panic(err)
	}

	return string(bs)
}

func (i Info) ToLDFlags(pkg string) string {
	var flags []string
	structVal := reflect.ValueOf(i)
	for i := 0; i < structVal.NumField(); i++ {
		field := structVal.Field(i)
		fieldName := structVal.Type().Field(i).Name
		privateFieldName := strings.ToLower(fieldName[0:1]) + fieldName[1:]
		flags = append(flags, generateLDFlag(pkg, privateFieldName, field.String()))
	}
	return strings.Join(flags, " ")
}

// AugmentFromEnv will try to infer versioning information from the local environment and augment the Info struct with it.
func AugmentFromEnv(info Info) Info {
	// Infer build by
	if info.BuildBy == "" {
		out, err := exec.Command("git", "config", "user.name").CombinedOutput()
		if err == nil {
			info.BuildBy = strings.TrimSpace(string(out))
		}

		// Note: disabled for now, because it might be too easy to unintentionally distribute the email.
		// out, err = exec.Command("git", "config", "user.email").CombinedOutput()
		// if err == nil {
		// 	info.BuildBy += fmt.Sprintf(" (%s)", strings.TrimSpace(string(out)))
		// }

		info.BuildBy = strings.TrimSpace(info.BuildBy)
	}

	// Infer the build date
	if info.BuildDate == "" {
		info.BuildDate = time.Now().UTC().Format(time.RFC3339)
	}

	// Infer build OS
	if info.BuildOS == "" {
		out, err := exec.Command("uname").CombinedOutput()
		if err == nil {
			info.BuildOS = strings.TrimSpace(string(out))
		}
	}

	// Infer build architecture
	if info.BuildArch == "" {
		out, err := exec.Command("uname", "-m").CombinedOutput()
		if err == nil {
			info.BuildArch = strings.TrimSpace(string(out))
		}
	}

	// Infer the git commit
	if info.GitCommit == "" {
		out, err := exec.Command("git", "rev-parse", "HEAD").CombinedOutput()
		if err == nil {
			info.GitCommit = strings.TrimSpace(string(out))
		}
	}

	// Infer the git branch
	if info.GitBranch == "" {
		out, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").CombinedOutput()
		if err == nil {
			info.GitBranch = strings.TrimSpace(string(out))
		}
	}

	// Infer the git commit date
	if info.GitCommitDate == "" && info.GitCommit != "" {
		out, err := exec.Command("git", "show", "-s", "--format=%ci", info.GitCommit).CombinedOutput()
		if err == nil {
			// Convert git format (2020-09-28 16:30:29 +0200) to RFC3339 (2006-01-02T15:04:05Z07:00)
			gitCommitDate, err := time.Parse("2006-01-02 15:04:05 -0700", strings.TrimSpace(string(out)))
			if err == nil {
				info.GitCommitDate = gitCommitDate.UTC().Format(time.RFC3339)
			}
		}
	}

	// Infer git status
	if info.GitTreeState == "" {
		out, err := exec.Command("git", "diff", "--quiet").CombinedOutput()
		if len(out) == 0 {
			if err == nil {
				info.GitTreeState = GitTreeStateClean
			} else {
				info.GitTreeState = GitTreeStateDirty
			}
		}
	}

	// Infer go version
	if info.GoVersion == "" {
		out, err := exec.Command("go", "version").CombinedOutput()
		if err == nil {
			info.GoVersion = strings.Split(strings.TrimSpace(string(out)), " ")[2]
		}
	}

	return info
}

func ValidateStrict(versionInfo Info) error {
	// Check if all fields are set
	infoType := reflect.ValueOf(versionInfo)
	for i := 0; i < infoType.NumField(); i++ {
		fieldType := infoType.Type().Field(i)
		if infoType.Field(i).IsZero() {
			return errors.New("field is required in strict mode: " + fieldType.Name)
		}
	}

	// Ensure that the git state is clean.
	if versionInfo.GitTreeState == GitTreeStateDirty {
		return errors.New("goversion requires a clean git tree state in strict mode")
	}

	// Validate the format of GitCommitDate timestamp.
	if ts, err := time.Parse(time.RFC3339, versionInfo.GitCommitDate); err != nil {
		return fmt.Errorf("date format in gitCommitDate is not a valid RFC3339 time: %v", err)
	} else if !ts.UTC().Equal(ts) {
		return errors.New("buildDate is not a UTC time")
	}

	// Validate the format of BuildDate timestamp.
	if ts, err := time.Parse(time.RFC3339, versionInfo.BuildDate); err != nil {
		return fmt.Errorf("date format in buildDate is not a a valid RFC3339 time: %v", err)
	} else if !ts.UTC().Equal(ts) {
		return errors.New("buildDate is not a UTC time")
	}

	// TODO validate that Version is a semantic version
	return nil
}

func generateLDFlag(pkg string, field string, val string) string {
	return fmt.Sprintf("-X \"%s.%s=%s\"", pkg, field, val)
}
