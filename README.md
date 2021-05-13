# broker-core

[![Made by Textile](https://img.shields.io/badge/made%20by-Textile-informational.svg?style=popout-square)](https://textile.io)
[![Chat on Slack](https://img.shields.io/badge/slack-slack.textile.io-informational.svg?style=popout-square)](https://slack.textile.io)
[![GitHub license](https://img.shields.io/github/license/textileio/broker-core.svg?style=popout-square)](./LICENSE)
[![GitHub action](https://github.com/textileio/broker-core/workflows/Test/badge.svg?style=popout-square)](https://github.com/textileio/broker-core/actions)
[![standard-readme compliant](https://img.shields.io/badge/readme%20style-standard-brightgreen.svg?style=popout-square)](https://github.com/RichardLitt/standard-readme)

> Broker for the Filecoin network

Join us on our [public Slack channel](https://slack.textile.io/) for news, discussions, and status updates. [Check out our blog](https://blog.textile.io/) for the latest posts and announcements.

## Table of Contents

- [Background](#background)
- [Install](#install)
- [Getting Started](#getting-started)
- [Contributing](#contributing)
- [Changelog](#changelog)
- [License](#license)

## Background

Broker packs and auctions uploaded data to miners on the Filecoin network.

## Install

```
go get github.com/textileio/broker-core
```

## Getting Started

### Running locally with some test data

The `bench.sh` script depends on the `$SEED_PHRASE` environment variable for locking NEAR funds (the seed phrase itself is available in 1Password in "NEAR Developers").

```bash
$ REPO_PATH=. make up
$ cmd/storaged/bench.sh 127.0.0.1:8888 100 200 10 0.1
```

### Steps to deploy a daemon

Here's an example of deploying `authd`:

1. `git pull main`
2. `make docker-push-head` (this simply rebuilds all containers and pushes them to Docker Hub)
3. Changed `authd` yaml https://github.com/textileio/ttcloud/pull/282/commits/01eee8b949f3194c844b0f43ada0f89fa459eaaf
4. Run `kubectl -n broker-staging apply -f authd.yaml`  (`k8/broker/broker-staging` in ttcloud repo)

## Contributing

Pull requests and bug reports are very welcome ❤️

This repository falls under the Textile [Code of Conduct](./CODE_OF_CONDUCT.md).

Feel free to get in touch by:
-   [Opening an issue](https://github.com/textileio/broker-core/issues/new)
-   Joining the [public Slack channel](https://slack.textile.io/)
-   Sending an email to contact@textile.io

## Changelog

A changelog is published along with each [release](https://github.com/textileio/broker-core/releases).

## License

[MIT](LICENSE)
