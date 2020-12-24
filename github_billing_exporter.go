package main

import (
	"encoding/json"
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
	namespace = "github_billing"
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

	// Actions billing metrics
	usedMinutesTotal = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "actions", "used_minutes"),
		"Total GitHub actions used minutes",
		[]string{"org"}, nil,
	)
	paidMinutedUsedTotal = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "actions", "paid_minutes"),
		"Total GitHub actions paid minutes",
		[]string{"org"}, nil,
	)
	includedMinutes = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "actions", "included_minutes"),
		"GitHub actions included minutes",
		[]string{"org"}, nil,
	)

	// Packages billing metrics
	totalGigabytesBandwidthUsed = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "packages", "bandwidth_used_gigabytes"),
		"GitHub packages used in gigabytes",
		[]string{"org"}, nil,
	)
	totalPaidGigabytesBandwidthUsed = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "packages", "bandwidth_used_paid_gigabytes"),
		"GitHub packages paid used in gigabytes",
		[]string{"org"}, nil,
	)
	includedGigabytesBandwidth = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "packages", "bandwidth_included_gigabytes"),
		"GitHub packages paid used in gigabytes",
		[]string{"org"}, nil,
	)

	// Shared storage metrics
	daysLeftInBillingCycle = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "cycle_remaining_days"),
		"GitHub packages paid used in gigabytes",
		[]string{"org"}, nil,
	)
	estimatedPaidStorageForMonth = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "storage", "estimated_month_pay"),
		"GitHub packages month estimated pay",
		[]string{"org"}, nil,
	)
	estimatedStorageForMonth = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "storage", "estimated_month_use"),
		"GitHub packages month estimated storage",
		[]string{"org"}, nil,
	)
)

type billing struct {
	// Actions billing
	TotalMinutesUsed     int `json:"total_minutes_used"`
	TotalPaidMinutedUsed int `json:"total_paid_minutes_used"`
	IncludedMinutes      int `json:"included_minutes"`

	// Packages billing
	TotalGigabytesBandwidthUsed     int `json:"total_gigabytes_bandwidth_used"`
	TotalPaidGigabytesBandwidthUsed int `json:"total_paid_gigabytes_bandwidth_used"`
	IncludedGigabytesBandwidth      int `json:"included_gigabytes_bandwidth"`

	// Storage billing
	DaysLeftInBillingCycle          int `json:"days_left_in_billing_cycle"`
	EstimatedPaidStorageForMonth    int `json:"estimated_paid_storage_for_month"`
	EstimatedStorageForMonth        int `json:"estimated_storage_for_month"`
}

type BillingCollector struct {
	GithubOrgs  string
	GithubToken string
	mutex       sync.Mutex
	client      *http.Client
}

func NewBillingCollector(githubOrgs string, githubToken string) *BillingCollector {
	return &BillingCollector{
		GithubOrgs:  githubOrgs,
		GithubToken: githubToken,
		client:      &http.Client{},
	}
}

func (e *BillingCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- up

	ch <- usedMinutesTotal
	ch <- paidMinutedUsedTotal
	ch <- includedMinutes

	ch <- totalGigabytesBandwidthUsed
	ch <- totalPaidGigabytesBandwidthUsed
	ch <- includedGigabytesBandwidth

	ch <- daysLeftInBillingCycle
	ch <- estimatedPaidStorageForMonth
	ch <- estimatedStorageForMonth
}

func (e *BillingCollector) Collect(ch chan<- prometheus.Metric) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	ok := e.collectActionsBillingMetrics(ch)
	ok = e.collectPackagesBillingMetrics(ch) && ok
	if ok {
		ch <- prometheus.MustNewConstMetric(up, prometheus.GaugeValue, 1.0)
	} else {
		ch <- prometheus.MustNewConstMetric(up, prometheus.GaugeValue, 0.0)
	}
}

func (e *BillingCollector) collectActionsBillingMetrics(ch chan<- prometheus.Metric) bool {
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

		ch <- prometheus.MustNewConstMetric(usedMinutesTotal, prometheus.GaugeValue, float64(b.TotalMinutesUsed), org)
		ch <- prometheus.MustNewConstMetric(paidMinutedUsedTotal, prometheus.GaugeValue, float64(b.TotalPaidMinutedUsed), org)
		ch <- prometheus.MustNewConstMetric(includedMinutes, prometheus.GaugeValue, float64(b.IncludedMinutes), org)
	}

	return true
}

func (e *BillingCollector) collectPackagesBillingMetrics(ch chan<- prometheus.Metric) bool {
	orgs := e.parseOrgs()
	for _, org := range orgs {
		var b billing
		req, _ := http.NewRequest("GET", "/orgs/"+org+"/settings/billing/packages", nil)
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

		ch <- prometheus.MustNewConstMetric(totalGigabytesBandwidthUsed, prometheus.GaugeValue, float64(b.TotalGigabytesBandwidthUsed), org)
		ch <- prometheus.MustNewConstMetric(totalPaidGigabytesBandwidthUsed, prometheus.GaugeValue, float64(b.TotalPaidGigabytesBandwidthUsed), org)
		ch <- prometheus.MustNewConstMetric(includedGigabytesBandwidth, prometheus.GaugeValue, float64(b.IncludedGigabytesBandwidth), org)
	}

	return true
}

func (e *BillingCollector) collectStorageBillingMetrics(ch chan<- prometheus.Metric) bool {
	orgs := e.parseOrgs()
	for _, org := range orgs {
		var b billing
		req, _ := http.NewRequest("GET", "/orgs/"+org+"/settings/billing/shared-storage", nil)
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

		ch <- prometheus.MustNewConstMetric(daysLeftInBillingCycle, prometheus.GaugeValue, float64(b.DaysLeftInBillingCycle), org)
		ch <- prometheus.MustNewConstMetric(estimatedPaidStorageForMonth, prometheus.GaugeValue, float64(b.EstimatedPaidStorageForMonth), org)
		ch <- prometheus.MustNewConstMetric(estimatedStorageForMonth, prometheus.GaugeValue, float64(b.EstimatedStorageForMonth), org)
	}

	return true
}

func (e *BillingCollector) parseOrgs() []string {
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

	exporter := NewBillingCollector(*githubOrgs, *githubToken)
	exporter2 := NewBillingCollector(*githubOrgs, *githubToken)
	prometheus.MustRegister(exporter, exporter2)
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