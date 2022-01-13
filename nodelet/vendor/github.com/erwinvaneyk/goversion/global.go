//go:generate goversion generate --goversion "" --pkg goversion -o global.gen.go
package goversion

import (
	"reflect"
	"sync"
)

var (
	versionInfoMu = &sync.RWMutex{}
	PackageName   = reflect.TypeOf(versionInfo).PkgPath()
	versionInfo   Info
)

func Set(updatedVersion Info) {
	if updatedVersion.IsEmpty() {
		return
	}
	versionInfoMu.Lock()
	defer versionInfoMu.Unlock()
	versionInfo = updatedVersion
}

func Get() Info {
	versionInfoMu.RLock()
	defer versionInfoMu.RUnlock()
	return versionInfo
}
