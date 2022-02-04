# GitHub billing exporter

[![GitHub Release](https://img.shields.io/github/release/raynigon/github_billing_exporter.svg?style=flat)](https://github.com/raynigon/github_billing_exporter/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/raynigon/github_billing_exporter/v2)](https://goreportcard.com/report/github.com/raynigon/github_billing_exporter/v2)
[![Docker Image Version (latest semver)](https://img.shields.io/docker/v/raynigon/github_billing_exporter?sort=semver)](https://hub.docker.com/r/raynigon/github-billing-exporter)

Forked From: https://github.com/borisputerka/github_billing_exporter because its not maintained there anymore.

This exporter exposes [Prometheus](https://prometheus.io/) metrics from GitHub billing API [endpoint](https://docs.github.com/en/free-pro-team@latest/rest/reference/billing) and the GitHub timing API [endpoint](https://docs.github.com/en/rest/reference/actions#get-workflow-usage).


## Installation

For pre-built binaries please take a look at the releases.
https://github.com/raynigon/github_billing_exporter/releases

### Docker

```bash
docker pull raynigon/github-billing-exporter:latest
docker run --rm -p 9776:9776 raynigon/github-billing-exporter:latest
```

Example `docker-compose.yml`:

```yaml
github_billing_exporter:
    image: raynigon/github-billing-exporter:latest
    command:
     - '--githun.token=<SECRET>'
    restart: always
    ports:
    - "127.0.0.1:9776:9776"
```

### Kubernetes

You can find an deployment definition at: https://github.com/raynigon/github_billing_exporter/tree/main/examples/kubernetes/deployment.yaml .

## Building and running

## Token privileges

Token needs to have access to read billing data, repositories and workflows.

### Build

    make

### Running

Running using an environment variable:

    export GBE_GITHUB_ORGS="ORG1 ORG2 ..."
    export GBE_GITHUB_TOKEN="example_token"
    ./github_billing_exporter

Running using args:

    ./github_billing_exporter \
    --github-orgs="ORG1 ORG2 ..." \
    --github-token="example_token"

## Collectors

There are three collectors (`actions`, `packages` and `storage`) all enabled by default. Disabling collector(s) can be done using arg `--no-collector.<name>`.

### List of collectors

Name	          | Description									                                        | Enabled
------------------|-------------------------------------------------------------------------------------|--------
actions_org       | Exposes billing statistics from `/orgs/{org}/settings/billing/actions`	            | `true`
packages_org      | Exposes billing statistics from `/orgs/{org}/settings/billing/packages`	            | `true`
storage_org       | Exposes billing statistics from `/orgs/{org}/settings/billing/shared-storage`       | `true`
actions_workflow  | Exposes used time from `/repos/{org}/{repo}/actions/workflows/{workflow_id}/timing` | `true`

## Environment variables / args reference

Version    | Env		               | Arg		             | Description			                	       | Default
-----------|---------------------------|-------------------------|-------------------------------------------------|---------
\>=`0.3.0` | `GBE_LISTEN_ADDRESS`      | `--web.listen-address`  | Address on which to expose metrics.             | `:9776`
\>=`0.3.0` | `GBE_METRICS_PATH`	       | `--web.telemetry-path`  | Path under which to expose metrics.             | `/metrics`
\>=`0.3.0` | `GBE_GITHUB_TOKEN`        | `--github.token`	     | GitHub token with billing/repo read privileges  | `""`
\>=`0.3.0` | `GBE_GITHUB_ORGS`	       | `--github.orgs`	     | GitHub organizations to scrape metrics for      | `""`
\>=`0.3.0` | `GBE_LOG_LEVEL`           | `--log.level`	         | -                                               | `"info"`
\>=`0.3.0` | `GBE_LOG_FORMAT`          | `--log.format`	         | -                                               | `"logfmt"`
\>=`0.3.0` | `GBE_LOG_OUTPUT`          | `--log.output`	         | -                                               | `"stdout"`
\>=`0.4.0` | `GBE_DISABLED_COLLECTORS` | `--disabled-collectors` | Collectors to disable			               | `""`
