
GIT_REVISION=`git rev-parse --short HEAD`
YPROXY_VERSION=`git describe --tags --abbrev=0`
LDFLAGS=-ldflags "-X github.com/yezzey-gp/yproxy/pkg.GitRevision=${GIT_REVISION} -X github.com/yezzey-gp/yproxy/pkg.YproxyVersion=${YPROXY_VERSION}"

####################### BUILD #######################

build:
	mkdir -p devbin
	go build -pgo=auto -o devbin/yproxy $(LDFLAGS) ./cmd/yproxy
	go build -o devbin/client ./cmd/client

####################### TESTS #######################

unittest:
	go test -race ./pkg/message/... ./pkg/proc/...

mockgen:
	mockgen -source=pkg/proc/yrreader.go -destination=pkg/mock/proc/yrreader.go -package=mock

version = $(shell git describe --tags --abbrev=0)
package:
	sed -i 's/YPROXY_VERSION/${version}/g' debian/changelog
	dpkg-buildpackage -us -uc
