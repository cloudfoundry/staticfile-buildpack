package libbuildpack

import "fmt"

const defaultVersionsError = "The buildpack manifest is misconfigured for 'default_versions'. " +
	"Contact your Cloud Foundry operator/admin. For more information, see " +
	"https://docs.cloudfoundry.org/buildpacks/custom.html#specifying-default-versions"

func dependencyMissingError(m *manifest, dep Dependency) string {
	var msg string
	otherVersions := m.allDependencyVersions(dep.Name)

	msg += fmt.Sprintf("DEPENDENCY MISSING IN MANIFEST:\n\n")

	if otherVersions == nil {
		msg += fmt.Sprintf("Dependency %s is not provided by this buildpack\n", dep.Name)
	} else {
		msg += fmt.Sprintf("Version %s of dependency %s is not supported by this buildpack.\n", dep.Version, dep.Name)
		msg += fmt.Sprintf("The versions of %s supported in this buildpack are:\n", dep.Name)

		for _, ver := range otherVersions {
			msg += fmt.Sprintf("\t- %s\n", ver)
		}
	}

	return msg
}
