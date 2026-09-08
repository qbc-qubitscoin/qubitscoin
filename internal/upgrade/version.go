// Package upgrade provides binary self-update and on-chain protocol upgrade
// scheduling for QubitsCoin nodes.
package upgrade

import (
	"fmt"
	"strconv"
	"strings"
)

// Current is the version embedded at compile time.
// Override with -ldflags "-X github.com/qbc-qubitscoin/qubitscoin/internal/upgrade.version=X.Y.Z"
var version = "0.5.0"

// Version represents a semantic version (Major.Minor.Patch).
type Version struct {
	Major, Minor, Patch int
}

// Current returns the running node's version.
func Current() Version {
	v, err := ParseVersion(version)
	if err != nil {
		return Version{0, 5, 0}
	}
	return v
}

// ParseVersion parses a "vX.Y.Z" or "X.Y.Z" string.
func ParseVersion(s string) (Version, error) {
	s = strings.TrimPrefix(s, "v")
	parts := strings.SplitN(s, ".", 3)
	if len(parts) != 3 {
		return Version{}, fmt.Errorf("invalid version %q", s)
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return Version{}, err
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return Version{}, err
	}
	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return Version{}, err
	}
	return Version{major, minor, patch}, nil
}

// String returns "vX.Y.Z".
func (v Version) String() string {
	return fmt.Sprintf("v%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// After returns true if v is strictly newer than others.
func (v Version) After(other Version) bool {
	if v.Major != other.Major {
		return v.Major > other.Major
	}
	if v.Minor != other.Minor {
		return v.Minor > other.Minor
	}
	return v.Patch > other.Patch
}

// Equal returns true if versions are identical.
func (v Version) Equal(other Version) bool {
	return v == other
}
