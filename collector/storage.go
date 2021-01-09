package collector

import (
	"encoding/json"
	"fmt"
	"github.com/go-kit/kit/log"
	"github.com/prometheus/client_golang/prometheus"
	"net/http"
	"sync"
)

var (
	storageSubsystem = "storage"
)

type storage struct {
	DaysLeftInBillingCycle       int `json:"days_left_in_billing_cycle"`
	EstimatedPaidStorageForMonth int `json:"estimated_paid_storage_for_month"`
	EstimatedStorageForMonth     int `json:"estimated_storage_for_month"`
}

type storageCollector struct {
	daysLeftInBillingCycle       *prometheus.Desc
	estimatedPaidStorageForMonth *prometheus.Desc
	estimatedStorageForMonth     *prometheus.Desc

	mutex  sync.Mutex
	client *http.Client
	logger log.Logger
}

func init() {
	registerCollector("storage", defaultEnabled, NewStorageCollector)
}

// NewStorageCollector returns a new Collector exposing storage billing stats.
func NewStorageCollector(logger log.Logger) (Collector, error) {
	return &storageCollector{
		daysLeftInBillingCycle: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "cycle_remaining_days"),
			"GitHub packages paid used in gigabytes",
			[]string{"org"}, nil,
		),
		estimatedPaidStorageForMonth: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, storageSubsystem, "estimated_month_pay"),
			"GitHub packages month estimated pay",
			[]string{"org"}, nil,
		),
		estimatedStorageForMonth: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, storageSubsystem, "estimated_month_use"),
			"GitHub packages month estimated storage",
			[]string{"org"}, nil,
		),
		client: &http.Client{},
		logger: logger,
	}, nil
}

// Describe implements Collector.
func (sc *storageCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- sc.daysLeftInBillingCycle
	ch <- sc.estimatedPaidStorageForMonth
	ch <- sc.estimatedStorageForMonth
}

// Update implements Collector and exposes storage billing stats
// from api.github.com/orgs/<org>/settings/billing/shared-storage.
func (sc *storageCollector) Update(ch chan<- prometheus.Metric) error {
	orgs := parseArg(*githubOrgs)
	for _, org := range orgs {
		var s storage
		req, _ := http.NewRequest("GET", "https://api.github.com/orgs/"+org+"/settings/billing/shared-storage", nil)
		req.Header.Set("Authorization", "token "+*githubToken)
		resp, err := sc.client.Do(req)
		if err != nil {
			return err
		}

		if resp.StatusCode != 200 {
			return fmt.Errorf("status %s, organization: %s collector: %s", resp.Status, org, storageSubsystem)
		}

		err = json.NewDecoder(resp.Body).Decode(&s)
		if err != nil {
			return err
		}

		resp.Body.Close()

		ch <- prometheus.MustNewConstMetric(sc.daysLeftInBillingCycle, prometheus.GaugeValue, float64(s.DaysLeftInBillingCycle), org)
		ch <- prometheus.MustNewConstMetric(sc.estimatedPaidStorageForMonth, prometheus.GaugeValue, float64(s.EstimatedPaidStorageForMonth), org)
		ch <- prometheus.MustNewConstMetric(sc.estimatedStorageForMonth, prometheus.GaugeValue, float64(s.EstimatedStorageForMonth), org)
	}

	return nil
}
