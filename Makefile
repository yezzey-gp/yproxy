
GIT_REVISION=`git rev-parse --short HEAD`
SPQR_VERSION=`git describe --tags --abbrev=0`
LDFLAGS=-ldflags "-X github.com/yezzey-gp/yproxy/pkg.GitRevision=${GIT_REVISION} -X github.com/yezzey-gp/yproxy/pkg.SpqrVersion=${SPQR_VERSION}"

####################### BUILD #######################

build:
	mkdir -p devbin
	go build -pgo=auto -o devbin/yproxy $(LDFLAGS) ./cmd/yproxy
	go build -o devbin/client ./cmd/client

####################### TESTS #######################

unittest:
	go test -race ./pkg/message/...

version = $(shell git describe --tags --abbrev=0)
package:
	sed -i 's/YPROXY_VERSION/${version}/g' debian/changelog
	dpkg-buildpackage -us -uc
