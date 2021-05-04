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

build-authd: $(GOVVV)
	$(BIN_BUILD_FLAGS) go build -ldflags="${GOVVV_FLAGS}" ./cmd/authd
.PHONY: build-authd

build-neard: $(GOVVV)
	$(BIN_BUILD_FLAGS) go build -ldflags="${GOVVV_FLAGS}" ./cmd/neard
.PHONY: build-neard

build-auctioneerd: $(GOVVV)
	$(BIN_BUILD_FLAGS) go build -ldflags="${GOVVV_FLAGS}" ./cmd/auctioneerd
.PHONY: build-auctioneerd

build-minerd: $(GOVVV)
	$(BIN_BUILD_FLAGS) go build -ldflags="${GOVVV_FLAGS}" ./cmd/minerd
.PHONY: build-minerd

build-packerd: $(GOVVV)
	$(BIN_BUILD_FLAGS) go build -ldflags="${GOVVV_FLAGS}" ./cmd/packerd
.PHONY: build-packerd

build-dealerd: $(GOVVV)
	go build -ldflags="${GOVVV_FLAGS}" ./cmd/dealerd
.PHONY: build-dealerd


install: $(GOVVV)
	$(BIN_BUILD_FLAGS) go install -ldflags="${GOVVV_FLAGS}" ./...
.PHONY: install

install-minerd: $(GOVVV)
	$(BIN_BUILD_FLAGS) go install -ldflags="${GOVVV_FLAGS}" ./cmd/minerd
.PHONY: install-minerd

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

down:
	docker-compose -f docker-compose-dev.yml down
.PHONY: down

mocks: $(MOCKERY) clean-mocks
	$(MOCKERY) --all --dir gen --keeptree
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
	$(BUF) check lint
	# $(BUF) check breaking --against-input '.git#branch=main'

# https is what we run when testing in most CI providers.
# This does breaking change detection against our remote HTTPS git repository.
.PHONY: buf-https
buf-https: $(BUF)
	$(BUF) check lint
	# $(BUF) check breaking --against-input "$(HTTPS_GIT)#branch=main"

# ssh is what we run when testing in CI providers that provide ssh public key authentication.
# This does breaking change detection against our remote HTTPS ssh repository.
# This is especially useful for private repositories.
.PHONY: buf-ssh
buf-ssh: $(BUF)
	$(BUF) check lint
	# $(BUF) check breaking --against-input "$(SSH_GIT)#branch=main"

define docker_push_daemon_head
	for daemon in $(1); do \
		echo docker buildx build --platform linux/amd64 --push -t textile/$${daemon}:sha-$(HEAD_SHORT) -f cmd/$${daemon}/Dockerfile .; \
		docker buildx build --platform linux/amd64 --push -t textile/$${daemon}:sha-$(HEAD_SHORT) -f cmd/$${daemon}/Dockerfile .; \
	done
endef

docker-push-head:
	$(call docker_push_daemon_head,auctioneerd authd brokerd minerd neard packerd storaged dealerd)
.PHONY: docker-push-head

################################################################
# Hairy stuff from filecoin-ffi dep, soon getting rid of  this.
################################################################
FFI_PATH:=extern/filecoin-ffi/
FFI_DEPS:=.install-filcrypto
FFI_DEPS:=$(addprefix $(FFI_PATH),$(FFI_DEPS))

$(FFI_DEPS): .filecoin-install ;

.filecoin-install: $(FFI_PATH)
	$(MAKE) -C $(FFI_PATH) $(FFI_DEPS:$(FFI_PATH)%=%)
	@touch $@

MODULES+=$(FFI_PATH)
BUILD_DEPS+=.filecoin-install

$(MODULES): .update-modules ;

# dummy file that marks the last time modules were updated
.update-modules:
	git submodule update --init --recursive
	touch $@

build-ffi: clean-ffi $(BUILD_DEPS)
.PHONY: build-ffi

clean-ffi:
	rm -f .filecoin-install
	rm -f .update-modules
	rm -f extern/filecoin-ffi/.install-filcrypto
	git submodule deinit --all -f

