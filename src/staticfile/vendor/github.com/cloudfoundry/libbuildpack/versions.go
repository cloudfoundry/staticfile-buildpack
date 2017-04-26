package libbuildpack

import (
	"fmt"

	"github.com/Masterminds/semver"
)

func FindMatchingVersion(versionSpec string, existingVersions []string) (string, error) {
	constraint, err := semver.NewConstraint(versionSpec)
	if err != nil {
		return "", err
	}

	maxVersion, _ := semver.NewVersion("0")

	for _, ver := range existingVersions {
		v, err := semver.NewVersion(ver)
		if err != nil {
			return "", err
		}
		if constraint.Check(v) && maxVersion.LessThan(v) {
			maxVersion = v
		}
	}

	if maxVersion.Original() != "0" {
		return maxVersion.Original(), nil
	}

	return "", fmt.Errorf("no match found for %s in %v", versionSpec, existingVersions)
}
