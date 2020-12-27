package collector

import (
	"encoding/json"
	"github.com/prometheus/client_golang/prometheus"
	"net/http"
	"sync"
)

type storage struct {
	DaysLeftInBillingCycle       int `json:"days_left_in_billing_cycle"`
	EstimatedPaidStorageForMonth int `json:"estimated_paid_storage_for_month"`
	EstimatedStorageForMonth     int `json:"estimated_storage_for_month"`
}

type StorageCollector struct {
	GithubOrgs  string
	GithubToken string

	daysLeftInBillingCycle       *prometheus.Desc
	estimatedPaidStorageForMonth *prometheus.Desc
	estimatedStorageForMonth     *prometheus.Desc

	mutex  sync.Mutex
	client *http.Client
}

func init() {
	registerCollector("storage", NewStorageCollector)
}

func NewStorageCollector(githubOrgs string, githubToken string) Collector {
	return &StorageCollector{
		GithubOrgs:  githubOrgs,
		GithubToken: githubToken,
		daysLeftInBillingCycle: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "cycle_remaining_days"),
			"GitHub packages paid used in gigabytes",
			[]string{"org"}, nil,
		),
		estimatedPaidStorageForMonth: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "storage", "estimated_month_pay"),
			"GitHub packages month estimated pay",
			[]string{"org"}, nil,
		),
		estimatedStorageForMonth: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "storage", "estimated_month_use"),
			"GitHub packages month estimated storage",
			[]string{"org"}, nil,
		),
		client: &http.Client{},
	}
}

func (sc *StorageCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- sc.daysLeftInBillingCycle
	ch <- sc.estimatedPaidStorageForMonth
	ch <- sc.estimatedStorageForMonth
}

func (sc *StorageCollector) Update(ch chan<- prometheus.Metric) error {
	orgs := parseArg(sc.GithubOrgs)
	for _, org := range orgs {
		var s storage
		req, _ := http.NewRequest("GET", "/orgs/"+org+"/settings/billing/shared-storage", nil)
		req.Header.Set("Authorization", "token "+sc.GithubToken)
		resp, err := sc.client.Do(req)
		if err != nil {
			return err
		}
		if resp.StatusCode != 200 {
			return err
		}

		err = json.NewDecoder(resp.Body).Decode(&s)
		if err != nil {
			return err
		}

		ch <- prometheus.MustNewConstMetric(sc.daysLeftInBillingCycle, prometheus.GaugeValue, float64(s.DaysLeftInBillingCycle), org)
		ch <- prometheus.MustNewConstMetric(sc.estimatedPaidStorageForMonth, prometheus.GaugeValue, float64(s.EstimatedPaidStorageForMonth), org)
		ch <- prometheus.MustNewConstMetric(sc.estimatedStorageForMonth, prometheus.GaugeValue, float64(s.EstimatedStorageForMonth), org)
	}

	return nil
}
