GIT_SHA    = $(shell git rev-parse --short HEAD)
GIT_DIRTY  = $(shell test -n "`git status --porcelain`" && echo "-dirty")
GIT_BRANCH = $(shell git rev-parse --abbrev-ref HEAD)

.PHONY: all
all: test build

.PHONY: build
build:
	go build -ldflags "-w -X github.com/ryane/konvert/cmd.Version=${GIT_BRANCH} -X github.com/ryane/konvert/cmd.GitCommit=${GIT_SHA}${GIT_DIRTY}" .

.PHONY: test
test:
	go test ./...

.PHONY: testv
testv:
	go test -v ./...

.PHONY: install
install:
	go install -ldflags "-w -X github.com/ryane/konvert/cmd.Version=${GIT_BRANCH} -X github.com/ryane/konvert/cmd.GitCommit=${GIT_SHA}${GIT_DIRTY}" .

docker: build
	# TODO: fix Version
	CGO_ENABLED=0 GOOS=linux go build -a -tags netgo -ldflags "-w -X github.com/ryane/konvert/cmd.Version=${GIT_BRANCH} -X github.com/ryane/konvert/cmd.GitCommit=${GIT_SHA}${GIT_DIRTY}" .
	docker build -t ryane/konvert:${GIT_SHA}${GIT_DIRTY} .

push: docker
	docker push ryane/konvert:${GIT_SHA}${GIT_DIRTY}

example: build
	cd example; ../konvert

deploy-example: example
	kustomize build example | kubectl apply -f -
