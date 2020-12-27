# GitHub billing exporter

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

    ./mysqld_exporter \
    --github-orgs="ORG1,ORG2,..." \
    --github-token="example_token"

## Collectors

There are three collectors (`actions`, `packages` and `storage`) all enabled by default. Disabling collector(s) can be done using environment variable `DISABLED_COLLECTORS` or arg `--disabled-collectors`.

### List of collectors

Name	 | Description									 | Enabled
---------|-------------------------------------------------------------------------------|--------
actions  | Exposes billing statistics from `/orgs/{org}/settings/billing/actions`	 | `true`
packages | Exposes billing statistics from `/orgs/{org}/settings/billing/packages`	 | `true`
storage  | Exposes billing statistics from `/orgs/{org}/settings/billing/shared-storage` | `true`

## Environment variables / args reference

Env		      | Arg			| Description				     | Default  | Required
----------------------|-------------------------|--------------------------------------------|------------|---------
`DISABLED_COLLECTORS` | `--disabled-collectors` | Collectors to disable			     | `""`	  | `no`
`GITHUB_ORGS`	      | `--github-orgs`		| GitHub organizations to scrape metrics for | `""`	  | `yes`
`GITHUB_TOKEN`        | `--github-token`	| GitHub token with billind read privileges  | `""`	  | `yes`
`LISTEN_ADDRESS`      | `--web.listen-address`  | Address on which to expose metrics.        | `:9999`    | `no`
`METRICS_PATH`	      | `--web.telemetry-path`  | Path under which to expose metrics.        | `/metrics` | `no`
