LDFLAGS := "-s -w"

amd64:
	CGO_ENABLED="0" GOARCH="amd64" go build -ldflags=${LDFLAGS} -o bin/amd64/rtp2http

arm64:
	CGO_ENABLED="0" GOARCH="arm64" go build -ldflags=${LDFLAGS} -o bin/arm64/rtp2http

install:
	cp bin/amd64/rtp2http /usr/local/bin

all: amd64 arm64
