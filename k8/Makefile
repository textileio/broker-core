include ../.bingo/Variables.mk

.DEFAULT_GOAL=deploy

HEAD_SHORT ?= $(shell git rev-parse --short HEAD)

deploy: $(KUSTOMIZE)
	cd ${DEPLOYMENT} && $(KUSTOMIZE) edit set image \
		textile/api:sha-$(HEAD_SHORT) \
		textile/gql:sha-$(HEAD_SHORT) \
		textile/auctioneer:sha-$(HEAD_SHORT) \
		textile/auth:sha-$(HEAD_SHORT) \
		textile/broker:sha-$(HEAD_SHORT) \
		textile/dealer:sha-$(HEAD_SHORT) \
		textile/near:sha-$(HEAD_SHORT) \
		textile/eth:sha-$(HEAD_SHORT) \
		textile/poly:sha-$(HEAD_SHORT) \
		textile/packer:sha-$(HEAD_SHORT) \
		textile/piecer:sha-$(HEAD_SHORT) \
		textile/relay:sha-$(HEAD_SHORT) \
		textile/storage:sha-$(HEAD_SHORT)
	cd ${DEPLOYMENT} && $(KUSTOMIZE) build . | kubectl apply -f -
.PHONY: deploy

run-uploadbot:
	kubectl create configmap uploadbot --from-file=../cmd/storaged/bench.sh --dry-run=client -o yaml | kubectl apply -f -
	- kubectl -n broker-${DEPLOYMENT} delete job/uploadbot
	cat base/uploadbot.yaml | sed 's/<ENV>/${DEPLOYMENT}/g' | kubectl apply -f -
