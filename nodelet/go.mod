module github.com/platform9/nodelet/nodelet

go 1.17

require (
	github.com/cenkalti/backoff v2.2.1+incompatible
	github.com/choria-io/go-validator v1.1.1
	github.com/erwinvaneyk/goversion v0.1.3
	github.com/ghodss/yaml v1.0.0
	github.com/golang/mock v1.4.3
	github.com/google/gofuzz v1.1.0
	github.com/grpc-ecosystem/go-grpc-middleware v1.2.2
	github.com/imdario/mergo v0.3.11
	github.com/mitchellh/mapstructure v1.1.2
	github.com/olekukonko/tablewriter v0.0.4
	github.com/onsi/ginkgo v1.14.2
	github.com/onsi/gomega v1.10.3
	github.com/platform9/pf9-qbert/sunpike/apiserver v0.0.0
	github.com/platform9/pf9-qbert/sunpike/conductor v0.0.0-20210928133414-c4e8ce211671
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cobra v1.1.1
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.7.0
	github.com/stretchr/testify v1.6.1
	go.uber.org/zap v1.16.0
	google.golang.org/grpc v1.33.2
	k8s.io/apimachinery v0.20.6
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/erwinvaneyk/cobras v0.0.0-20200914200705-1d2dfabe2493 // indirect
	github.com/fsnotify/fsnotify v1.4.9 // indirect
	github.com/go-logr/logr v0.2.1 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.0 // indirect
	github.com/google/uuid v1.2.0 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/json-iterator/go v1.1.10 // indirect
	github.com/magiconair/properties v1.8.1 // indirect
	github.com/mattn/go-runewidth v0.0.9 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.1 // indirect
	github.com/nxadm/tail v1.4.5 // indirect
	github.com/pelletier/go-toml v1.8.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/spf13/afero v1.2.2 // indirect
	github.com/spf13/cast v1.3.0 // indirect
	github.com/spf13/jwalterweatherman v1.0.0 // indirect
	github.com/subosito/gotenv v1.2.0 // indirect
	go.uber.org/atomic v1.7.0 // indirect
	go.uber.org/multierr v1.6.0 // indirect
	golang.org/x/net v0.0.0-20210226172049-e18ecbb05110 // indirect
	golang.org/x/sys v0.0.0-20210426230700-d19ff857e887 // indirect
	golang.org/x/text v0.3.4 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	google.golang.org/genproto v0.0.0-20201110150050-8816d57aaa9a // indirect
	google.golang.org/protobuf v1.26.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/ini.v1 v1.51.0 // indirect
	gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20200605160147-a5ece683394c // indirect
	k8s.io/api v0.20.6 // indirect
	k8s.io/klog/v2 v2.4.0 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.0.3 // indirect
)

// The replaced version for v0.0.0 should equal the version of github.com/platform9/pf9-qbert/sunpike/conductor
replace github.com/platform9/pf9-qbert/sunpike/apiserver v0.0.0 => github.com/platform9/pf9-qbert/sunpike/apiserver v0.0.0-20210928133414-c4e8ce211671

// To build/test nodelet with local changes to the sunpike components.
// Uncomment the lines below, and comment out the replace above.
// Do not commit these changes!
// replace (
// 	github.com/platform9/pf9-qbert/sunpike/apiserver => ../../pf9-qbert/sunpike/apiserver
// 	github.com/platform9/pf9-qbert/sunpike/conductor => ../../pf9-qbert/sunpike/conductor
// )
