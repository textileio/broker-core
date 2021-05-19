include ../.bingo/Variables.mk

include ${K8_DEPLOYMENT}/config.env
export

.DEFAULT_GOAL=build

HEAD_SHORT ?= $(shell git rev-parse --short HEAD)

build: $(ENVSUBST) $(KUSTOMIZE)
	cd ${K8_DEPLOYMENT} && $(KUSTOMIZE) edit set image \
		textile/auctioneer:sha-$(HEAD_SHORT) \
		textile/auth:sha-$(HEAD_SHORT) \
		textile/broker:sha-$(HEAD_SHORT) \
		textile/dealer:sha-$(HEAD_SHORT) \
		textile/near:sha-$(HEAD_SHORT) \
		textile/packer:sha-$(HEAD_SHORT) \
		textile/storage:sha-$(HEAD_SHORT)
	cd ${K8_DEPLOYMENT} && $(KUSTOMIZE) build . | $(ENVSUBST)
.PHONY: build
