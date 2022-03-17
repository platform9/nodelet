# How to add new phases

## Using bash scripts
1. Add the bash script under `pf9-kube/root/opt/pf9/pf9-kube/phases` directory. The shell script must be executable.
2. Add a go file under `pf9-kube/nodelet/pkg/utils/phases` under the appropriate sub-directory. A sample of go file -
```
package <sub-directory name>

import (
	"github.com/platform9/nodelet/nodelet/pkg/utils/phases/base"
	sunpikev1alpha1 "github.com/platform9/pf9-qbert/sunpike/apiserver/pkg/apis/sunpike/v1alpha1"
)

func NewPhaseBlah() base.PhaseInterface {
	blahPhase := &base.BashScriptPhase{
		Filename: "<name of shell script>",
		Location: "/opt/pf9/pf9-kube/phases/<name of shell script>",
		HostPhase: sunpikev1alpha1.HostPhase{
			Name:  "<phase name # this will be displayed on the UI>",
			Order: <phase order>,
		},
		HasStatusCheck: <true/false>,
	}
	return blahPhase
}
```
3. Add the phase to either `GetMasterPhases` or `GetWorkerPhases` or both in `pf9-kube/nodelet/pkg/utils/phases/register.go`

## In golang
1. Add a go file under `pf9-kube/nodelet/pkg/utils/phases` under the appropriate sub-directory. The new phase struct must implement PhaseInterface defined in `pf9-kube/nodelet/pkg/utils/phases/base/base.go`.
2. Add the phase to either `GetMasterPhases` or `GetWorkerPhases` or both in `pf9-kube/nodelet/pkg/utils/phases/register.go`

# How are the phases executed

1. Phases are executed in ascending order as specified in GetMasterPhases and GetWorkerPhases during cluster creation.

2. Phase status are executed in ascending order as specified in GetMasterPhases and GetWorkerPhases.

3. Phase stops are executed in descending (reverse) order as specified in GetMasterPhases and GetWorkerPhases during cluster tear-down.