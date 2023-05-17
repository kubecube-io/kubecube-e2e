.PHONY: test build docker-build docker-build-multi-arch vet vendor build-clear clear

IMG ?= kubecube-e2e:latest
MULTI_ARCH ?= true

test:
	go test ./e2e -v

build: vet
ifeq ($(MULTI_ARCH),true)
	CGO_ENABLED=0 GOOS=linux GO111MODULE=on go test -mod=vendor -c -o cube.test ./e2e
else
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go test -mod=vendor -c -o cube.test ./e2e
endif

docker-build:
	docker build -f ./Dockerfile -t ${IMG} .
docker-push: ## Push docker image with the manager.
	docker push ${IMG}

vet:
	go vet ./...

vendor:
	go mod vendor

build-clear: vet
ifeq ($(MULTI_ARCH),true)
	CGO_ENABLED=0 GOOS=linux GO111MODULE=on go build -mod=vendor -o cube.clear cmd/clear/main.go
else
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -mod=vendor -o cube.clear cmd/clear/main.go
endif

clear: build-clear
	./cube.clear

