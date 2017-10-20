test:
	go test -v ./...

test-docker:
	docker run -i -t --rm --entrypoint /bin/sh \
		-v $(shell pwd):/go/src/github.com/jcoene/reactor \
		golang:1.9.1 \
		-c "cd /go/src/github.com/jcoene/reactor && go test -v ./..."
