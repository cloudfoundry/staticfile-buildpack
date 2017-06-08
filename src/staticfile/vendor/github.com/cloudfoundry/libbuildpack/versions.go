package libbuildpack

import (
	"fmt"
	"sort"

	"github.com/Masterminds/semver"
)

func FindMatchingVersion(constraint string, versions []string) (string, error) {
	var err error
	var depVersions []*semver.Version
	var depVersion *semver.Version
	var versionConstraint *semver.Constraints
	for _, ver := range versions {
		depVersion, err = semver.NewVersion(ver)

		if err != nil {
			return "", err
		}

		versionConstraint, err = semver.NewConstraint(constraint)

		if err != nil {
			return "", err
		}

		if versionConstraint.Check(depVersion) {
			depVersions = append(depVersions, depVersion)
		}
	}

	if len(depVersions) != 0 {
		sort.Sort(semver.Collection(depVersions))

		return depVersions[len(depVersions)-1].String(), nil
	}

	return "", fmt.Errorf("no match found for %s in %v", constraint, versions)
}
