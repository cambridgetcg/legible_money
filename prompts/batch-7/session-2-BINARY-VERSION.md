# Session: BINARY-VERSION — Set version metadata + ldflags in Makefile

You are working in the legible_money Cosmos SDK chain codebase at ~/Desktop/legible_money.

## Task: Set binary version, name, and server_name via ldflags

### Context
B6-7 found that `legbled version` returns empty name/version/server_name. For testnet, operators need to identify what version they're running.

### Step 1: Read existing build setup
```bash
cat Makefile | head -60
# or
cat scripts/build.sh | head -40
```
Find how the binary is currently built. Check if there's already ldflags infrastructure.

### Step 2: Add version variables
The Cosmos SDK reads version info from `github.com/cosmos/cosmos-sdk/version` package. Set these via ldflags:

```makefile
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "v0.1.0-dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS := -X github.com/cosmos/cosmos-sdk/version.Name=legible \
           -X github.com/cosmos/cosmos-sdk/version.AppName=legbled \
           -X github.com/cosmos/cosmos-sdk/version.Version=$(VERSION) \
           -X github.com/cosmos/cosmos-sdk/version.Commit=$(COMMIT) \
           -X github.com/cosmos/cosmos-sdk/version.BuildTags="netgo,ledger"

build:
	go build -ldflags "$(LDFLAGS) -s -w" -o build/legbled ./cmd/legbled
```

### Step 3: Also update Dockerfile
In `deployment/docker/Dockerfile.cosmos`, the build command should use the same ldflags:
```dockerfile
RUN go build -ldflags "${LDFLAGS} -s -w" -o /usr/local/bin/legbled ./cmd/legbled
```

### Step 4: Verify
```bash
make build  # or however it's invoked
./build/legbled version --long
```

Should show:
```
name: legible
server_name: legbled
version: v0.1.0-dev (or git tag)
commit: abc1234
```

## Report format
Write to reports/batch-7/SESSION-B7-2-BINARY-VERSION.md:
- Makefile/build changes
- Dockerfile changes
- `legbled version --long` output
- Build verification
