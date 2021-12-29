GIT_SHA    = $(shell git rev-parse --short HEAD)
GIT_DIRTY  = $(shell test -n "`git status --porcelain`" && echo "-dirty")
GIT_BRANCH = $(shell git rev-parse --abbrev-ref HEAD)

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

docker: build
	# TODO: fix Version
	CGO_ENABLED=0 GOOS=linux go build -a -tags netgo -ldflags "-w -X github.com/kumorilabs/konvert/cmd.Version=${GIT_BRANCH} -X github.com/kumorilabs/konvert/cmd.GitCommit=${GIT_SHA}${GIT_DIRTY}" .
	docker build -t kumorilabs/konvert:${GIT_SHA}${GIT_DIRTY} .

push: docker
	docker push kumorilabs/konvert:${GIT_SHA}${GIT_DIRTY}

example: build
	kpt fn eval example/mysql --exec ./fn.sh --results-dir results --fn-config example/mysql/konvert.yaml
	cat results/results.yaml

deploy-example: example
	kustomize build example/mysql | kubectl apply -f -

build-example: example
	kustomize build example/mysql

clean-example:
	find example/mysql -type f -not -name konvert.yaml -delete
