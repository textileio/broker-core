include ../.bingo/Variables.mk

.DEFAULT_GOAL=deploy

HEAD_SHORT ?= 86c66dc

deploy: $(KUSTOMIZE)
	cd ${DEPLOYMENT} && $(KUSTOMIZE) edit set image \
		textile/auctioneer:sha-$(HEAD_SHORT) \
		textile/auth:sha-$(HEAD_SHORT) \
		textile/broker:jsign \
		textile/dealer:sha-$(HEAD_SHORT) \
		textile/near:sha-$(HEAD_SHORT) \
		textile/packer:jsign \
		textile/piecer:sha-$(HEAD_SHORT) \
		textile/storage:sha-$(HEAD_SHORT)
	cd ${DEPLOYMENT} && $(KUSTOMIZE) build . | kubectl apply -f -
.PHONY: deploy

run-uploadbot:
	kubectl create configmap uploadbot --from-file=../cmd/storaged/bench.sh --dry-run=client -o yaml | kubectl apply -f -
	- kubectl -n broker-${DEPLOYMENT} delete job/uploadbot
	cat base/uploadbot.yaml | sed 's/<ENV>/${DEPLOYMENT}/g' | kubectl apply -f -
