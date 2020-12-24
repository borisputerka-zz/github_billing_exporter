package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/promlog"
	"github.com/prometheus/common/promlog/flag"
	"github.com/prometheus/common/version"
	"gopkg.in/alecthomas/kingpin.v2"
)

const (
	namespace = "github"
)

var (
	metricsAddress = kingpin.Flag("metrics_address", "Address on which to expose metrics.").Envar("METRICS_ADDRESS").Default(":9999").String()
	metricsPath    = kingpin.Flag("metrics_path", "Path under which to expose metrics.").Envar("METRICS_PATH").Default("/metrics").String()
	githubToken    = kingpin.Flag("github_token", "GitHub token to access api").Envar("GITHUB_TOKEN").String()
	githubOrgs     = kingpin.Flag("github_orgs", "Organizations to get metrics from").Envar("GITHUB_ORGS").String()
	gracefulStop   = make(chan os.Signal)
)

var (
	up = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "up"),
		"Can be test_server reached",
		nil, nil,
	)
	minutesUsedTotal = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "minutes_used_total"),
		"GitHub run status",
		[]string{"org"}, nil,
	)
	paidMinutedUsedTotal = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "paid_minutes_total"),
		"GitHub jobs in run",
		[]string{"org"}, nil,
	)
	includedMinutes = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "includedMinutes"),
		"GitHub runs total duration",
		[]string{"org"}, nil,
	)
)

type billing struct {
	TotalMinutesUsed     int `json:"total_minutes_used"`
	TotalPaidMinutedUsed int `json:"total_paid_minutes_used"`
	IncludedMinutes      int `json:"included_minutes"`
}

type Exporter struct {
	GithubOrgs  string
	GithubToken string
	mutex       sync.Mutex
	client      *http.Client
	logger      log.Logger
}

func NewExporter(githubOrgs string, githubToken string) *Exporter {
	return &Exporter{
		GithubOrgs:  githubOrgs,
		GithubToken: githubToken,
		client:      &http.Client{},
	}
}

func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- up
	ch <- minutesUsedTotal
	ch <- paidMinutedUsedTotal
	ch <- includedMinutes
}

func (e *Exporter) collectBillingMetrics(ch chan<- prometheus.Metric) bool {
	orgs := e.parseOrgs()

	for _, org := range orgs {
		var b billing
		req, _ := http.NewRequest("GET", "/orgs/"+org+"/settings/billing/actions", nil)
		req.Header.Set("Authorization", "token "+e.GithubToken)
		resp, err := e.client.Do(req)
		if err != nil {
			return false
		}
		if resp.StatusCode != 200 {
			return false
		}

		err = json.NewDecoder(resp.Body).Decode(&b)
		if err != nil {
			return false
		}

		ch <- prometheus.MustNewConstMetric(minutesUsedTotal, prometheus.GaugeValue, float64(b.TotalMinutesUsed), org)
		ch <- prometheus.MustNewConstMetric(paidMinutedUsedTotal, prometheus.GaugeValue, float64(b.TotalPaidMinutedUsed), org)
		ch <- prometheus.MustNewConstMetric(includedMinutes, prometheus.GaugeValue, float64(b.IncludedMinutes), org)
	}

	return true
}

func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	ok := e.collectBillingMetrics(ch)
	if ok {
		ch <- prometheus.MustNewConstMetric(up, prometheus.GaugeValue, 1.0)
	} else {
		ch <- prometheus.MustNewConstMetric(up, prometheus.GaugeValue, 0.0)
	}
}

func (e *Exporter) parseOrgs() []string {
	orgs := strings.ReplaceAll(e.GithubOrgs, " ", "")
	orgsList := strings.Split(orgs, ",")

	return orgsList
}

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

	exporter := NewExporter(*githubOrgs, *githubToken)
	prometheus.MustRegister(exporter)
	prometheus.MustRegister(version.NewCollector("github_billing_exporter"))

	level.Info(logger).Log("msg", "Starting exporter", "version", version.Info())
	level.Info(logger).Log("Build context", version.BuildContext())
	level.Info(logger).Log("msg", "Starting Server", "listening address", *metricsAddress)

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
			<h1>Node Exporter</h1>
			<p><a href="` + *metricsPath + `">Metrics</a></p>
			</body>
			</html>`))
	})
	level.Error(logger).Log("msg", http.ListenAndServe(*metricsAddress, nil))

}
