include ../.bingo/Variables.mk


.DEFAULT_GOAL=deploy

HEAD_SHORT ?= 5d7f606

deploy: $(KUSTOMIZE)
	cd ${DEPLOYMENT} && $(KUSTOMIZE) edit set image \
		textile/auctioneer:sha-$(HEAD_SHORT) \
		textile/auth:sha-$(HEAD_SHORT) \
		textile/broker:sha-$(HEAD_SHORT) \
		textile/dealer:sha-$(HEAD_SHORT) \
		textile/near:sha-$(HEAD_SHORT) \
		textile/packer:jsign \
		textile/storage:sha-$(HEAD_SHORT)
	cd ${DEPLOYMENT} && $(KUSTOMIZE) build . | kubectl apply -f -
.PHONY: deploy

run-uploadbot:
	- kubectl -n broker-${DEPLOYMENT} delete job/uploadbot
	cat base/uploadbot.yaml | sed 's/<ENV>/${DEPLOYMENT}/g' | kubectl apply -f -