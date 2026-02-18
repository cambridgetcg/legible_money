# SESSION-B7-2: BINARY-VERSION

## Fix: `legbled version` returns empty name/version/server_name

### Problem

`legbled version` and `legbled version --long` returned empty strings for `name`, `server_name`, and `version`. Operators on testnet had no way to identify which build they were running.

### Root Cause

The Makefile `build` target ran `go build` without `-ldflags`, so the `github.com/cosmos/cosmos-sdk/version` package variables were never populated.

### Changes

#### 1. `Makefile` — Add version ldflags

Added version metadata variables and LDFLAGS:

```makefile
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "v0.1.0-dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TAGS := netgo,ledger

LDFLAGS := -X github.com/cosmos/cosmos-sdk/version.Name=legible \
           -X github.com/cosmos/cosmos-sdk/version.AppName=legbled \
           -X github.com/cosmos/cosmos-sdk/version.Version=$(VERSION) \
           -X github.com/cosmos/cosmos-sdk/version.Commit=$(COMMIT) \
           -X "github.com/cosmos/cosmos-sdk/version.BuildTags=$(BUILD_TAGS)"
```

Updated `build` target:

```makefile
build:
	go build -ldflags "$(LDFLAGS) -s -w" -o build/legbled ./cmd/legbled
```

#### 2. `deployment/docker/Dockerfile.cosmos` — Pass version via build args

```dockerfile
ARG VERSION="v0.1.0-dev"
ARG COMMIT="unknown"
ARG BUILD_TAGS="netgo,ledger"

RUN CGO_ENABLED=1 go build -o /usr/local/bin/legbled \
    -ldflags="-s -w \
      -X github.com/cosmos/cosmos-sdk/version.Name=legible \
      -X github.com/cosmos/cosmos-sdk/version.AppName=legbled \
      -X github.com/cosmos/cosmos-sdk/version.Version=${VERSION} \
      -X github.com/cosmos/cosmos-sdk/version.Commit=${COMMIT} \
      -X 'github.com/cosmos/cosmos-sdk/version.BuildTags=${BUILD_TAGS}'" \
    ./cmd/legbled
```

Docker builds can override with:
```bash
docker build --build-arg VERSION=v0.2.0 --build-arg COMMIT=abc1234 ...
```

### Verification

```
$ make build
Building legbled...

$ build/legbled version
v0.1.0-338-g613e569-dirty

$ build/legbled version --long
build_tags: netgo,ledger
commit: 613e569
cosmos_sdk_version: v0.50.15
go: go version go1.26.0 darwin/arm64
name: legible
server_name: legbled
version: v0.1.0-338-g613e569-dirty
```

All fields populated correctly. The `-dirty` suffix and commit count will resolve once changes are committed and tagged.
