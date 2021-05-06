# broker-core

# Running locally with some test data

The `bench.sh` script depends on the `$SEED_PHRASE` environment variable for locking NEAR funds (the seed phrase itself is available in 1Password in "NEAR Developers").

```bash
$ REPO_PATH=. make up
$ cmd/storaged/bench.sh 127.0.0.1:8888 100
```

# Steps to deploy a daemon
Here's an example of deploying `authd`:

1. pull main
1. `make docker-push-head` (this simply rebuilds all containers and push them to Dockerhub)
1. Changed `authd` yaml https://github.com/textileio/ttcloud/pull/282/commits/01eee8b949f3194c844b0f43ada0f89fa459eaaf
1. run `kubectl -n broker-staging apply -f authd.yaml`  (`k8/broker/broker-staging` in ttcloud repo)
