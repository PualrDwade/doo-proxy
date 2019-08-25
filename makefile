all: build

TAG?=dev
FLAGS=
ENVVAR=
GOOS?=linux
COMPONENT=doo-proxy

build: clean
	$(ENVVAR) GOOS=$(GOOS) go build -o ${COMPONENT}

test: clean build
	$(ENVVAR) go test --test.short -race ./... $(FLAGS)

run: build
	./${COMPONENT} --credential "MYTESTCREDENTIAL"

install: 
	go install

clean:
	rm -rf ${COMPONENT}
format:
	test -z "$$(find . -path ./vendor -prune -type f -o -name '*.go' -exec gofmt -s -d {} + | tee /dev/stderr)" || \
	test -z "$$(find . -path ./vendor -prune -type f -o -name '*.go' -exec gofmt -s -w {} + | tee /dev/stderr)"

.PHONY: all  build test clean format install
