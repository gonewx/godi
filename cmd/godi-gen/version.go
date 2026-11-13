package main

import (
	"runtime/debug"

	"github.com/gonewx/godi/internal/genlib"
)

// develVersion is what a generator built without a module version reports.
const develVersion = genlib.DevelVersion

// generatorVersion is the version of the running generator: the module version
// Go stamped into the binary when it was built as a dependency (`go tool
// godi-gen`, `go install ...@v0.2.0`), or develVersion otherwise.
func generatorVersion() string {
	return versionFromBuildInfo(debug.ReadBuildInfo())
}

// versionFromBuildInfo picks the generator version out of build info. A binary
// without build info, or with an empty or "(devel)" main-module version (a
// `go run` or `go test` inside this repository), reports develVersion.
func versionFromBuildInfo(info *debug.BuildInfo, ok bool) string {
	if !ok || info == nil || info.Main.Version == "" {
		return develVersion
	}
	return info.Main.Version
}
