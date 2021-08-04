# broker-core

[![Made by Textile](https://img.shields.io/badge/made%20by-Textile-informational.svg)](https://textile.io)
[![Chat on Slack](https://img.shields.io/badge/slack-slack.textile.io-informational.svg)](https://slack.textile.io)
[![standard-readme compliant](https://img.shields.io/badge/readme%20style-standard-brightgreen.svg)](https://github.com/RichardLitt/standard-readme)

> Broker for the Filecoin network

Join us on our [public Slack channel](https://slack.textile.io/) for news, discussions, and status updates. [Check out our blog](https://blog.textile.io/) for the latest posts and announcements.

## Table of Contents

- [Background](#background)
- [Install](#install)
- [Getting Started](#getting-started)
  - [Storage providers: Run a `bidbot`](#storage providers-run-a-bidbot)
  - [Running locally with some test data](#running-locally-with-some-test-data)
  - [Step to deploy a daemon](#steps-to-deploy-a-daemon)
- [Contributing](#contributing)
- [Changelog](#changelog)
- [License](#license)

## Background

Broker packs and auctions uploaded data to storage providers on the Filecoin network.

## Install

```
go get github.com/textileio/broker-core
```

## Getting Started

### Storage providers: Run a `bidbot`

Storage providers on the Filecoin Network can bid in storage deal auctions.

Refer to the [Bidbot](https://github.com/textileio/bidbot) repo.

### Running locally with some test data

The `bench.sh` script depends on the `$SEED_PHRASE` environment variable for locking NEAR funds (the seed phrase itself is available in 1Password in "NEAR Developers").

```bash
$ REPO_PATH=. make up
$ cmd/storaged/bench.sh 127.0.0.1:8888 100 200 10 0.1
```

### CI/CD

The `k8` folder contains manifests to deploy the Broker system to a Kubernetes cluster.

The system is automatically deployed with the following rules:
- There are three environments: `edge`, `staging`, and `production` (these correspond to k8 namespaces).
- Any push on any branch will trigger a deploy to `edge` if the commit message contains the substring `[shipit]` :sunglasses:.
- Any push to `main` (including pull request merges) will trigger a deploy to `staging`.
- Releases trigger a deploy to `production`.

A bot named `uploadbot` also is available to run in any environment. To run it, go to the `k8` folder and run `DEPLOYMENT=<production|staging|edge> make run-uploadbot`.

### Dashboard

A [Grafana dashboard](https://gke.grafana.textile.dev/) are available to have observability in the system.
To sign in, use your GH account that should be part of the Textile organization.

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
