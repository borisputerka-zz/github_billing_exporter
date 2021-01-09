# GitHub billing exporter

[![GitHub Release](https://img.shields.io/github/release/borisputerka/github_billing_exporter.svg?style=flat)](https://github.com/borisputerka/github_billing_exporter/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/borisputerka/github_billing_exporter)](https://goreportcard.com/report/github.com/borisputerka/github_billing_exporter)

This expoter exposes [Prometheus](https://prometheus.io/) metrics from GitHub billing API [endpoint](https://docs.github.com/en/free-pro-team@latest/rest/reference/billing).

## Building and running

## Token privileges

Token needs to have access to read billing data.

### Build

    make

### Running

Running using an environment variable:

    export GITHUB_ORGS="ORG1,ORG2,..."
    export GITHUB_TOKEN="example_token"
    ./github_billing_exporter

Running using args:

    ./github_billing_exporter \
    --github-orgs="ORG1,ORG2,..." \
    --github-token="example_token"

## Collectors

There are three collectors (`actions`, `packages` and `storage`) all enabled by default. Disabling collector(s) can be done using arg `--no-collector.<name>`.

### List of collectors

Name	 | Description									 | Enabled
---------|-------------------------------------------------------------------------------|--------
actions  | Exposes billing statistics from `/orgs/{org}/settings/billing/actions`	 | `true`
packages | Exposes billing statistics from `/orgs/{org}/settings/billing/packages`	 | `true`
storage  | Exposes billing statistics from `/orgs/{org}/settings/billing/shared-storage` | `true`

## Environment variables / args reference

Version    | Env		   | Arg		     | Description				  | Default
-----------|-----------------------|-------------------------|--------------------------------------------|---------
=`0.1.0`   | `DISABLED_COLLECTORS` | `--disabled-collectors` | Collectors to disable			  | `""`
\>=`0.1.0` | `GITHUB_ORGS`	   | `--github-orgs`	     | GitHub organizations to scrape metrics for | `""`
\>=`0.1.0` | `GITHUB_TOKEN`        | `--github-token`	     | GitHub token with billind read privileges  | `""`
\>=`0.1.0` | `LISTEN_ADDRESS`      | `--web.listen-address`  | Address on which to expose metrics.        | `:9776`
\>=`0.1.0` | `METRICS_PATH`	   | `--web.telemetry-path`  | Path under which to expose metrics.        | `/metrics`
