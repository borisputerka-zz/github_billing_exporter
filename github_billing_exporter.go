package main

import (
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/borisputerka/github_billing_exporter/collector"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/promlog"
	"github.com/prometheus/common/promlog/flag"
	"github.com/prometheus/common/version"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	listenAddress = kingpin.Flag(
		"web.listen-address",
		"Address on which to expose metrics.",
	).Envar("LISTEN_ADDRESS").Default(":9776").String()
	metricsPath = kingpin.Flag(
		"web.telemetry-path",
		"Path under which to expose metrics.",
	).Envar("METRICS_PATH").Default("/metrics").String()
	disabledCollectors = kingpin.Flag(
		"disabled-collectors",
		"Collectors to disable",
	).Envar("COLLECTORS_DISABLED").String()
	githubToken = kingpin.Flag(
		"github-token",
		"GitHub token to access api",
	).Envar("GITHUB_TOKEN").String()
	githubOrgs = kingpin.Flag("github-orgs",
		"Organizations to get metrics from",
	).Envar("GITHUB_ORGS").String()
	gracefulStop = make(chan os.Signal)
)

func main() {
	promlogConfig := &promlog.Config{}
	flag.AddFlags(kingpin.CommandLine, promlogConfig)
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()
	logger := promlog.New(promlogConfig)

	signal.Notify(gracefulStop, syscall.SIGTERM)
	signal.Notify(gracefulStop, syscall.SIGINT)
	signal.Notify(gracefulStop, syscall.SIGHUP)
	signal.Notify(gracefulStop, syscall.SIGQUIT)

	exporter := collector.NewBillingCollector(*githubOrgs, *githubToken, *disabledCollectors, logger)
	prometheus.MustRegister(exporter)
	prometheus.MustRegister(version.NewCollector("github_billing_exporter"))

	level.Info(logger).Log("msg", "Starting exporter", "version", version.Info())
	level.Info(logger).Log("Build context", version.BuildContext())
	level.Info(logger).Log("msg", "Starting Server", "listening address", *listenAddress)

	// listener for the termination signals from the OS
	go func() {
		level.Info(logger).Log("msg", "listening and wait for graceful stop")
		sig := <-gracefulStop
		level.Info(logger).Log("msg", "Shutting exporter", "caught sig", sig)
		time.Sleep(2 * time.Second)
		os.Exit(0)
	}()

	http.Handle(*metricsPath, promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html>
			<head><title>GitHub Billing Expoorter</title></head>
			<body>
			<h1>GitHub Billing Exporter</h1>
			<p><a href="` + *metricsPath + `">Metrics</a></p>
			</body>
			</html>`))
	})
	level.Error(logger).Log("msg", http.ListenAndServe(*listenAddress, nil))
}
