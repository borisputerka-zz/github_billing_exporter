package collector

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

type packages struct {
	TotalGigabytesBandwidthUsed     int `json:"total_gigabytes_bandwidth_used"`
	TotalPaidGigabytesBandwidthUsed int `json:"total_paid_gigabytes_bandwidth_used"`
	IncludedGigabytesBandwidth      int `json:"included_gigabytes_bandwidth"`
}

type PackagesCollector struct {
	GithubOrgs  string
	GithubToken string

	totalGigabytesBandwidthUsed     *prometheus.Desc
	totalPaidGigabytesBandwidthUsed *prometheus.Desc
	includedGigabytesBandwidth      *prometheus.Desc

	mutex  sync.Mutex
	client *http.Client
}

func init() {
	registerCollector("packages", NewPackagesCollector)
}

func NewPackagesCollector(githubOrgs string, githubToken string) Collector {
	return &PackagesCollector{
		GithubOrgs:  githubOrgs,
		GithubToken: githubToken,
		totalGigabytesBandwidthUsed: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "packages", "bandwidth_used_gigabytes"),
			"GitHub packages used in gigabytes",
			[]string{"org"}, nil,
		),
		totalPaidGigabytesBandwidthUsed: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "packages", "bandwidth_used_paid_gigabytes"),
			"GitHub packages paid used in gigabytes",
			[]string{"org"}, nil,
		),
		includedGigabytesBandwidth: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "packages", "bandwidth_included_gigabytes"),
			"GitHub packages paid used in gigabytes",
			[]string{"org"}, nil,
		),
		client: &http.Client{},
	}
}

func (pc *PackagesCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- pc.totalGigabytesBandwidthUsed
	ch <- pc.totalPaidGigabytesBandwidthUsed
	ch <- pc.includedGigabytesBandwidth
}

func (pc *PackagesCollector) Update(ch chan<- prometheus.Metric) error {
	orgs := parseArg(pc.GithubOrgs)
	for _, org := range orgs {
		var p packages
		req, _ := http.NewRequest("GET", "/orgs/"+org+"/settings/billing/packages", nil)
		req.Header.Set("Authorization", "token "+pc.GithubToken)
		resp, err := pc.client.Do(req)
		if err != nil {
			return err
		}
		if resp.StatusCode != 200 {
			return err
		}

		err = json.NewDecoder(resp.Body).Decode(&p)
		if err != nil {
			return err
		}

		ch <- prometheus.MustNewConstMetric(pc.totalGigabytesBandwidthUsed, prometheus.GaugeValue, float64(p.TotalGigabytesBandwidthUsed), org)
		ch <- prometheus.MustNewConstMetric(pc.totalPaidGigabytesBandwidthUsed, prometheus.GaugeValue, float64(p.TotalPaidGigabytesBandwidthUsed), org)
		ch <- prometheus.MustNewConstMetric(pc.includedGigabytesBandwidth, prometheus.GaugeValue, float64(p.IncludedGigabytesBandwidth), org)
	}

	return nil
}
