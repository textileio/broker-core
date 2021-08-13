include .bingo/Variables.mk

.DEFAULT_GOAL=build

HEAD_SHORT ?= $(shell git rev-parse --short HEAD)

BIN_BUILD_FLAGS?=CGO_ENABLED=0
BIN_VERSION?="git"
GOVVV_FLAGS=$(shell $(GOVVV) -flags -version $(BIN_VERSION) -pkg $(shell go list ./buildinfo))

lint: $(GOLANGCI_LINT)
	$(GOLANGCI_LINT) run
.PHONYY: lint

build: $(GOVVV)
	$(BIN_BUILD_FLAGS) go build -ldflags="${GOVVV_FLAGS}" ./...
.PHONY: build

build-storaged: $(GOVVV)
	$(BIN_BUILD_FLAGS) go build -ldflags="${GOVVV_FLAGS}" ./cmd/storaged
.PHONY: build-storaged

build-brokerd: $(GOVVV)
	$(BIN_BUILD_FLAGS) go build -ldflags="${GOVVV_FLAGS}" ./cmd/brokerd
.PHONY: build-brokerd

build-apid: $(GOVVV)
	$(BIN_BUILD_FLAGS) go build -ldflags="${GOVVV_FLAGS}" ./cmd/apid
.PHONY: build-apid

build-authd: $(GOVVV)
	$(BIN_BUILD_FLAGS) go build -ldflags="${GOVVV_FLAGS}" ./cmd/authd
.PHONY: build-authd

build-neard: $(GOVVV)
	$(BIN_BUILD_FLAGS) go build -ldflags="${GOVVV_FLAGS}" ./cmd/chainapis/neard
.PHONY: build-neard

build-ethd: $(GOVVV)
	$(BIN_BUILD_FLAGS) go build -ldflags="${GOVVV_FLAGS}" ./cmd/chainapis/ethd
.PHONY: build-ethd

build-polyd: $(GOVVV)
	$(BIN_BUILD_FLAGS) go build -ldflags="${GOVVV_FLAGS}" ./cmd/chainapis/polyd
.PHONY: build-polyd

build-auctioneerd: $(GOVVV)
	$(BIN_BUILD_FLAGS) go build -ldflags="${GOVVV_FLAGS}" ./cmd/auctioneerd
.PHONY: build-auctioneerd

build-packerd: $(GOVVV)
	$(BIN_BUILD_FLAGS) go build -ldflags="${GOVVV_FLAGS}" ./cmd/packerd
.PHONY: build-packerd

build-piecerd: $(GOVVV)
	$(BIN_BUILD_FLAGS) go build -ldflags="${GOVVV_FLAGS}" ./cmd/piecerd
.PHONY: build-piecerd

build-dealerd: $(GOVVV)
	$(BIN_BUILD_FLAGS) go build -ldflags="${GOVVV_FLAGS}" ./cmd/dealerd
.PHONY: build-dealerd

install: $(GOVVV)
	$(BIN_BUILD_FLAGS) go install -ldflags="${GOVVV_FLAGS}" ./...
.PHONY: install

define gen_release_files
	$(GOX) -osarch=$(3) -output="build/$(2)/$(2)_${BIN_VERSION}_{{.OS}}-{{.Arch}}/$(2)" -ldflags="${GOVVV_FLAGS}" $(1)
	mkdir -p build/dist; \
	cd build/$(2); \
	for release in *; do \
		cp ../../LICENSE ../../README.md $${release}/; \
		if [ $${release} != *"windows"* ]; then \
  		BIN_FILE=$(2) $(GOMPLATE) -f ../../dist/install.tmpl -o "$${release}/install"; \
			tar -czvf ../dist/$${release}.tar.gz $${release}; \
		else \
			zip -r ../dist/$${release}.zip $${release}; \
		fi; \
	done
endef

up:
	COMPOSE_DOCKER_CLI_BUILD=1 docker-compose -f docker-compose-dev.yml up --build
.PHONY: up

up-patched:
	./tests/buildx_patch.sh
	COMPOSE_DOCKER_CLI_BUILD=1 docker-compose -f docker-compose-dev.yml up --build -V
.PHONY: up-patched

down:
	docker-compose -f docker-compose-dev.yml down
.PHONY: down

mocks: $(MOCKERY) clean-mocks
	$(MOCKERY) --name="(ChainAPI|Broker|FilClient|LotusClient|Auctioneer|Dealer|Piecer|Packer)" --keeptree --recursive
.PHONY: mocks

clean-mocks:
	rm -rf mocks
.PHONY: clean-mocks

protos: $(BUF) $(PROTOC_GEN_GO) $(PROTOC_GEN_GO_GRPC) clean-protos
	$(BUF) generate --template '{"version":"v1beta1","plugins":[{"name":"go","out":"gen","opt":"paths=source_relative","path":$(PROTOC_GEN_GO)},{"name":"go-grpc","out":"gen","opt":"paths=source_relative","path":$(PROTOC_GEN_GO_GRPC)}]}'
.PHONY: protos

clean-protos:
	find . -type f -name '*.pb.go' -delete
	find . -type f -name '*pb_test.go' -delete
.PHONY: clean-protos

# local is what we run when testing locally.
# This does breaking change detection against our local git repository.
.PHONY: buf-local
buf-local: $(BUF)
	$(BUF) lint
	# $(BUF) breaking --against-input '.git#branch=main'

# https is what we run when testing in most CI providers.
# This does breaking change detection against our remote HTTPS git repository.
.PHONY: buf-https
buf-https: $(BUF)
	$(BUF) lint
	# $(BUF) breaking --against-input "$(HTTPS_GIT)#branch=main"

# ssh is what we run when testing in CI providers that provide ssh public key authentication.
# This does breaking change detection against our remote HTTPS ssh repository.
# This is especially useful for private repositories.
.PHONY: buf-ssh
buf-ssh: $(BUF)
	$(BUF) lint
	# $(BUF) breaking --against-input "$(SSH_GIT)#branch=main"

define gen_sql_assets
	for daemon in $(1); do \
		cd cmd/$${daemon}d/store && $(GO_BINDATA) -pkg migrations -prefix migrations/ -o migrations/migrations.go -ignore=migrations.go migrations && $(SQLC) generate; cd -; \
	done
endef

sql-assets: $(GO_BINDATA) $(SQLC)
	$(call gen_sql_assets,broker piecer dealer packer auth api);
.PHONY: sql-assets

generate: protos sql-assets
.PHONY: generate

define docker_push_daemon_head
	for daemon in $(1); do \
		echo docker buildx build --platform linux/amd64 --push -t textile/$${daemon}:sha-$(HEAD_SHORT) -f cmd/$${daemon}d/Dockerfile .; \
		docker buildx build --platform linux/amd64 --push -t textile/$${daemon}:sha-$(HEAD_SHORT) -f cmd/$${daemon}d/Dockerfile .; \
	done
endef

docker-push-head:
	$(call docker_push_daemon_head,auctioneer auth broker dealer chainapis/near chainapis/eth chainapis/poly packer piecer storage);
.PHONY: docker-push-head
