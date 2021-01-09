package collector

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/go-kit/kit/log"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	packagesSubsystem = "packages"
)

type packages struct {
	TotalGigabytesBandwidthUsed     int `json:"total_gigabytes_bandwidth_used"`
	TotalPaidGigabytesBandwidthUsed int `json:"total_paid_gigabytes_bandwidth_used"`
	IncludedGigabytesBandwidth      int `json:"included_gigabytes_bandwidth"`
}

type PackagesCollector struct {
	totalGigabytesBandwidthUsed     *prometheus.Desc
	totalPaidGigabytesBandwidthUsed *prometheus.Desc
	includedGigabytesBandwidth      *prometheus.Desc

	mutex  sync.Mutex
	client *http.Client
	logger log.Logger
}

func init() {
	registerCollector("packages", defaultEnabled, NewPackagesCollector)
}

func NewPackagesCollector(logger log.Logger) (Collector, error) {
	return &PackagesCollector{
		totalGigabytesBandwidthUsed: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, packagesSubsystem, "bandwidth_used_gigabytes"),
			"GitHub packages used in gigabytes",
			[]string{"org"}, nil,
		),
		totalPaidGigabytesBandwidthUsed: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, packagesSubsystem, "bandwidth_used_paid_gigabytes"),
			"GitHub packages paid used in gigabytes",
			[]string{"org"}, nil,
		),
		includedGigabytesBandwidth: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, packagesSubsystem, "bandwidth_included_gigabytes"),
			"GitHub packages paid used in gigabytes",
			[]string{"org"}, nil,
		),
		client: &http.Client{},
		logger: logger,
	}, nil
}

func (pc *PackagesCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- pc.totalGigabytesBandwidthUsed
	ch <- pc.totalPaidGigabytesBandwidthUsed
	ch <- pc.includedGigabytesBandwidth
}

func (pc *PackagesCollector) Update(ch chan<- prometheus.Metric) error {
	orgs := parseArg(*githubOrgs)
	for _, org := range orgs {
		var p packages
		req, _ := http.NewRequest("GET", "https://api.github.com/orgs/"+org+"/settings/billing/packages", nil)
		req.Header.Set("Authorization", "token "+*githubToken)
		resp, err := pc.client.Do(req)
		if err != nil {
			return err
		}

		if resp.StatusCode != 200 {
			return fmt.Errorf("status %s, organization: %s collector: %s", resp.Status, org, packagesSubsystem)
		}

		err = json.NewDecoder(resp.Body).Decode(&p)
		if err != nil {
			return err
		}
		resp.Body.Close()

		ch <- prometheus.MustNewConstMetric(pc.totalGigabytesBandwidthUsed, prometheus.GaugeValue, float64(p.TotalGigabytesBandwidthUsed), org)
		ch <- prometheus.MustNewConstMetric(pc.totalPaidGigabytesBandwidthUsed, prometheus.GaugeValue, float64(p.TotalPaidGigabytesBandwidthUsed), org)
		ch <- prometheus.MustNewConstMetric(pc.includedGigabytesBandwidth, prometheus.GaugeValue, float64(p.IncludedGigabytesBandwidth), org)
	}

	return nil
}
