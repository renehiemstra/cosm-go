package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const Version = "0.1.0" // Move the version constant here

// getGlobalCosmDir returns the global .cosm directory in the user's home directory
func getGlobalCosmDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %v", err)
	}
	return filepath.Join(homeDir, ".cosm"), nil
}

var ValidRegistries = []string{"cosmic-hub", "local"}

// PrintVersion prints the version of the cosm tool and exits
func PrintVersion() {
	fmt.Printf("cosm version %s\n", Version)
	os.Exit(0)
}

// ParseSemVer parses a semantic version string into its components
func ParseSemVer(version string) (semVer, error) {
	parts := strings.Split(strings.TrimPrefix(version, "v"), ".")
	if len(parts) < 2 {
		return semVer{}, fmt.Errorf("invalid version format '%s': must be vX.Y.Z or vX.Y", version)
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return semVer{}, fmt.Errorf("invalid major version in '%s': %v", version, err)
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return semVer{}, fmt.Errorf("invalid minor version in '%s': %v", version, err)
	}
	patch := 0
	if len(parts) > 2 {
		patch, err = strconv.Atoi(parts[2])
		if err != nil {
			return semVer{}, fmt.Errorf("invalid patch version in '%s': %v", version, err)
		}
	}
	return semVer{Major: major, Minor: minor, Patch: patch}, nil
}

// semVer represents a semantic version (vX.Y.Z)
type semVer struct {
	Major, Minor, Patch int
}

// MaxSemVer returns the higher of two semantic versions
func MaxSemVer(v1, v2 string) (string, error) {
	s1, err := ParseSemVer(v1)
	if err != nil {
		return "", err
	}
	s2, err := ParseSemVer(v2)
	if err != nil {
		return "", err
	}
	if s1.Major > s2.Major {
		return v1, nil
	}
	if s1.Major < s2.Major {
		return v2, nil
	}
	if s1.Minor > s2.Minor {
		return v1, nil
	}
	if s1.Minor < s2.Minor {
		return v2, nil
	}
	if s1.Patch >= s2.Patch {
		return v1, nil
	}
	return v2, nil
}

// GetMajorVersion extracts the major version number as a string (e.g., "v1" from "v1.2.0")
func GetMajorVersion(version string) (string, error) {
	s, err := ParseSemVer(version)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("v%d", s.Major), nil
}
