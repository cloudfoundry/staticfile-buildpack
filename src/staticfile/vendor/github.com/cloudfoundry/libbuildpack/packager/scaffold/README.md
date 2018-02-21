### Buildpack User Documentation

### Building the Buildpack
To build this buildpack, run the following command from the buildpack's directory:

1. Source the .envrc file in the buildpack directory.
```bash
source .envrc
```
To simplify the process in the future, install [direnv](https://direnv.net/) which will automatically source .envrc when you change directories.

1. Install buildpack-packager
```bash
./scripts/install_tools.sh
```

1. Build the buildpack
```bash
buildpack-packager build
```

1. Use in Cloud Foundry
Upload the buildpack to your Cloud Foundry and optionally specify it by name

```bash
cf create-buildpack [BUILDPACK_NAME] [BUILDPACK_ZIP_FILE_PATH] 1
cf push my_app [-b BUILDPACK_NAME]
```

### Testing
Buildpacks use the [Cutlass](https://github.com/cloudfoundry/libbuildpack/cutlass) framework for running integration tests.

To test this buildpack, run the following command from the buildpack's directory:

1. Source the .envrc file in the buildpack directory.

```bash
source .envrc
```
To simplify the process in the future, install [direnv](https://direnv.net/) which will automatically source .envrc when you change directories.

1. Run unit tests

```bash
./scripts/unit.sh
```

1. Run integration tests

```bash
./scripts/integration.sh
```

More information can be found on Github [cutlass](https://github.com/cloudfoundry/libbuildpack/cutlass).

### Reporting Issues
Open an issue on this project

## Disclaimer
This buildpack is experimental and not yet intended for production use.
