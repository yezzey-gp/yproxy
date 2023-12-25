
build:
	mkdir -p devbin
	go build -o devbin/yproxy ./cmd/yproxy
	go build -o devbin/client ./cmd/client

####################### TESTS #######################

unittest:
	go test -race ./pkg/message/...

