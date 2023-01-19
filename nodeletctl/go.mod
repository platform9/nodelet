module github.com/platform9/nodelet/nodeletctl

replace github.com/platform9/pf9-qbert/sunpike/apiserver v0.0.0 => github.com/platform9/pf9-qbert/sunpike/apiserver v0.0.0-20210928133414-c4e8ce211671

go 1.16

require (
	github.com/ghodss/yaml v1.0.0
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51
	github.com/mitchellh/go-homedir v1.1.0
	github.com/platform9/nodelet/nodelet v0.0.0-20220420170655-9ece5c5b1f61
	github.com/platform9/pf9ctl v0.0.0-20230116114556-40afd6d532d3
	github.com/spf13/cobra v1.4.0
	github.com/spf13/viper v1.11.0
	go.etcd.io/etcd/client/v3 v3.5.4
	go.uber.org/zap v1.21.0
	gopkg.in/natefinch/lumberjack.v2 v2.0.0
	k8s.io/apimachinery v0.23.6
	k8s.io/client-go v0.23.6
)
