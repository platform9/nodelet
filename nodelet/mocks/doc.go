//go:generate go run -mod=mod github.com/golang/mock/mockgen -package mocks -destination=./mock_phases.go -source=../pkg/phases/phase_interface.go -build_flags=-mod=mod
//go:generate go run -mod=mod github.com/golang/mock/mockgen -package mocks -destination=./mock_command.go -source=../pkg/utils/command/command.go -build_flags=-mod=mod
//go:generate go run -mod=mod github.com/golang/mock/mockgen -package mocks -destination=./mock_extension.go -source=../pkg/utils/extensionfile/extension.go -build_flags=-mod=mod
//go:generate go run -mod=mod github.com/golang/mock/mockgen -package mocks -destination=./mock_fileio.go -source=../pkg/utils/fileio/fileio.go -build_flags=-mod=mod

package mocks

import _ "github.com/golang/mock/mockgen/model"
