GIT_SHA    = $(shell git rev-parse --short HEAD)
GIT_DIRTY  = $(shell test -n "`git status --porcelain`" && echo "-dirty")
GIT_BRANCH = $(shell git rev-parse --abbrev-ref HEAD)

EXAMPLE ?= mysql

.PHONY: all
all: test build

.PHONY: build
build:
	go build -ldflags "-w -X github.com/kumorilabs/konvert/cmd.Version=${GIT_BRANCH} -X github.com/kumorilabs/konvert/cmd.GitCommit=${GIT_SHA}${GIT_DIRTY}" .

.PHONY: test
test:
	go test ./...

.PHONY: testv
testv:
	go test -v ./...

.PHONY: install
install:
	go install -ldflags "-w -X github.com/kumorilabs/konvert/cmd.Version=${GIT_BRANCH} -X github.com/kumorilabs/konvert/cmd.GitCommit=${GIT_SHA}${GIT_DIRTY}" .

example: build
	kpt fn eval example/${EXAMPLE} --exec "./konvert fn" --results-dir results --fn-config example/${EXAMPLE}/konvert.yaml
	cat results/results.yaml

deploy-example: example
	kustomize build example/${EXAMPLE} | kubectl apply -f -

build-example: example
	kustomize build example/${EXAMPLE}

clean-example:
	find example/${EXAMPLE} -type f -not -name konvert.yaml -delete
