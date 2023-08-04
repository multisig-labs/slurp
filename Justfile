# Build vars for versioning the binary
VERSION := `grep "const Version " pkg/version/version.go | sed -E 's/.*"(.+)"$$/\1/'`
GIT_COMMIT := `git rev-parse HEAD`
BUILD_DATE := `date '+%Y-%m-%d'`
VERSION_PATH := "github.com/multisig-labs/slurp/pkg/version"
LDFLAGS := "-X " + VERSION_PATH + ".BuildDate=" + BUILD_DATE + " -X " + VERSION_PATH + ".Version=" + VERSION + " -X " + VERSION_PATH + ".GitCommit=" + GIT_COMMIT

default:
  @just --list --unsorted

build:
	CGO_ENABLED=1 go build -ldflags "{{LDFLAGS}}" -o bin/slurp main.go

install: build
  mv bin/slurp $GOPATH/bin/slurp

# Delete and recreate a sqlite db
create-db:
	-rm slurp.db
	cat schema.sql | sqlite3 slurp.db
	sqlc generate