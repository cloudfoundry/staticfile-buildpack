## Installing the packager

```
go get github.com/cloudfoundry/libbuildpack
cd ~/go/src/github.com/cloudfoundry/libbuildpack/packager/buildpack-packager && go install
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
