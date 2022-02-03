ARG ARCH="amd64"
ARG OS="linux"
FROM quay.io/prometheus/busybox-${OS}-${ARCH}:glibc
LABEL maintainer="Simon Schneider <dev@raynigon.com>"

COPY github_billing_exporter /bin/github_billing_exporter

EXPOSE      9776
USER        nobody
ENTRYPOINT  [ "/bin/github_billing_exporter" ]
