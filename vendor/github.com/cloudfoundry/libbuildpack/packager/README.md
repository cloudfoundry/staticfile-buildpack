## Installing the packager

```
go get github.com/cloudfoundry/libbuildpack
cd ~/go/src/github.com/cloudfoundry/libbuildpack && GO111MODULE=on go mod download
cd packager/buildpack-packager &&  GO111MODULE=on go install
```

## How to regenerate bindata.go
Run `go generate` when you add, remove, or change the files in the `scaffold` directory.

This will generate a new `bindata.go` file, which you **SHOULD** commit to the repo.
Both the scaffold directory that this file is created from and the file itself belong in the repo.
Make changes directly to the scaffold directory and its files, not bindata.go.

For more on go-bindata: https://github.com/jteeuwen/go-bindata

## Running tests

```
ginkgo -r
```

---

## Selective Dependency Packaging

`buildpack-packager` supports building **cached** buildpacks that contain only a named
subset of dependencies. This is useful for operators who need a smaller artifact or a
pre-defined variant (e.g. a "minimal" build without optional agent libraries).

> **Note:** `--profile`, `--exclude`, and `--include` are only valid for **cached**
> buildpacks (`--cached` flag). Using them on an uncached build is a hard error.

### Packaging profiles

A buildpack author defines named profiles in `manifest.yml` under the
`packaging_profiles` key:

```yaml
packaging_profiles:
  minimal:
    description: "Core runtime only — no agents or profilers"
    exclude:
      - agent-dep
      - profiler-dep

  no-profiler:
    description: "Full build minus the profiler library"
    exclude:
      - profiler-dep
```

Each profile has:

| Field         | Type           | Description |
|---------------|----------------|-------------|
| `description` | string         | Human-readable summary shown by `buildpack-packager summary` |
| `exclude`     | list of string | Dependency names to omit when this profile is active |

Profile names must match `^[a-z0-9_-]+$` (lowercase letters, digits, hyphens,
underscores). This keeps names safe for embedding in zip filenames.

All dependency names listed in a profile's `exclude` list must exist in the
manifest — a typo is a hard error at packaging time.

### CLI flags

| Flag | Argument | Description |
|------|----------|-------------|
| `--profile` | profile name | Activate a named profile from `manifest.yml` |
| `--exclude` | `dep1,dep2,...` | Additional dependencies to exclude (comma-separated) |
| `--include` | `dep1,dep2,...` | Restore dependencies that the active profile excluded |

#### Examples

```sh
# Build with the "minimal" profile (omits agent-dep and profiler-dep)
buildpack-packager --cached --stack cflinuxfs4 --profile minimal

# Build with the "minimal" profile but restore profiler-dep
buildpack-packager --cached --stack cflinuxfs4 --profile minimal --include profiler-dep

# No profile — just exclude a specific dependency
buildpack-packager --cached --stack cflinuxfs4 --exclude agent-dep

# Combine a profile with an extra exclusion
buildpack-packager --cached --stack cflinuxfs4 --profile no-profiler --exclude agent-dep
```

### Resolution order

1. Profile's `exclude` list is applied first.
2. `--exclude` names are **unioned** with the profile exclusions.
3. `--include` names are **removed** from the exclusion set (overrides the profile).

Excluded dependencies are neither downloaded nor written into the packaged
`manifest.yml`.

### Output filename

The zip filename encodes which variant was built:

| Options used | Filename pattern |
|---|---|
| No opts | `<lang>_buildpack-cached-<stack>-v<ver>.zip` |
| `--profile minimal` | `<lang>_buildpack-cached-minimal-<stack>-v<ver>.zip` |
| `--profile minimal --include profiler-dep` | `<lang>_buildpack-cached-minimal+custom-<stack>-v<ver>.zip` |
| `--profile minimal --exclude extra-dep` | `<lang>_buildpack-cached-minimal+custom-<stack>-v<ver>.zip` |
| `--exclude agent-dep` (no profile) | `<lang>_buildpack-cached-custom-<stack>-v<ver>.zip` |

The `+custom` suffix appears only when the result deviates from a pure profile:
either an extra `--exclude` was added, or `--include` actually overrode one of
the profile's exclusions.

### Error conditions

All validation errors are **hard errors** — the packager exits non-zero with a
descriptive message. There are no silent no-ops or warnings.

| Situation | Error message |
|---|---|
| `--profile` / `--exclude` / `--include` on uncached build | `--profile/--exclude/--include are only valid for cached buildpacks` |
| `--include` without `--profile` | `--include requires --profile` |
| Unknown profile name | `packaging profile "<name>" not found in manifest` |
| Invalid profile name characters | `profile name "<name>" is invalid: must match ^[a-z0-9_-]+$` |
| Unknown dep in `--exclude` | `dependency "<name>" not found in manifest` |
| Unknown dep in `--include` | `dependency "<name>" not found in manifest` |
| `--include` of dep not excluded by profile | `--include "<name>" has no effect: dependency is not excluded by the profile or --exclude` |
| Profile's `exclude` list references unknown dep | `profile "<name>" references unknown dependency "<dep>"` |

### Go API

`Package()` is unchanged and delegates to `PackageWithOptions` with zero options:

```go
// Legacy — unchanged behaviour
zipFile, err := packager.Package(bpDir, cacheDir, version, stack, cached)

// New — selective packaging
zipFile, err := packager.PackageWithOptions(bpDir, cacheDir, version, stack, true,
    packager.PackageOptions{
        Profile: "minimal",
        Include: []string{"profiler-dep"},
    })
```

`PackageOptions` fields:

```go
type PackageOptions struct {
    // Profile is a packaging_profiles key from manifest.yml.
    Profile string
    // Exclude lists additional dependency names to skip.
    Exclude []string
    // Include restores dependency names excluded by Profile.
    // Requires Profile to be set. Hard error if a name was not excluded.
    Include []string
}
```

