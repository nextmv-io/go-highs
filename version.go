// © 2019-present nextmv.io inc

package highs

import (
	_ "embed"
	"strings"
)

//go:embed VERSION
var version string

// Version returns the version of the highs module.
func Version() string {
	return strings.TrimSpace(version)
}
