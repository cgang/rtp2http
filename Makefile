LDFLAGS := "-s -w"

amd64:
	CGO_ENABLED="0" GOARCH="amd64" go build -ldflags=${LDFLAGS} -o bin/rtp2http-amd64

arm64:
	CGO_ENABLED="0" GOARCH="arm64" go build -ldflags=${LDFLAGS} -o bin/rtp2http-arm64

all: amd64 arm64
