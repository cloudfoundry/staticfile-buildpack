package libbuildpack

import (
	"fmt"
	"sort"

	semver2 "github.com/Masterminds/semver"
	semver1 "github.com/blang/semver"
)

func FindMatchingVersion(constraint string, versions []string) (string, error) {
	version, err := matchBlang(constraint, versions)
	if err == nil {
		return version, nil
	}

	return matchMasterminds(constraint, versions)
}

func matchBlang(constraint string, versions []string) (string, error) {
	var depVersions semver1.Versions
	versionConstraint, err := semver1.ParseRange(constraint)
	if err != nil {
		return "", err
	}

	for _, ver := range versions {
		depVersion, err := semver1.Parse(ver)
		if err != nil {
			return "", err
		}

		if versionConstraint(depVersion) {
			depVersions = append(depVersions, depVersion)
		}
	}

	if len(depVersions) != 0 {
		sort.Sort(depVersions)
		return depVersions[len(depVersions)-1].String(), nil
	}

	return "", fmt.Errorf("no match found for %s in %v", constraint, versions)
}

func matchMasterminds(constraint string, versions []string) (string, error) {
	var depVersions []*semver2.Version
	versionConstraint, err := semver2.NewConstraint(constraint)
	if err != nil {
		return "", err
	}

	for _, ver := range versions {
		depVersion, err := semver2.NewVersion(ver)
		if err != nil {
			return "", err
		}

		if versionConstraint.Check(depVersion) {
			depVersions = append(depVersions, depVersion)
		}
	}

	if len(depVersions) != 0 {
		sort.Sort(semver2.Collection(depVersions))
		return depVersions[len(depVersions)-1].String(), nil
	}

	return "", fmt.Errorf("no match found for %s in %v", constraint, versions)
}
