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
	actionsSubsystem = "actions"
)

type actions struct {
	TotalMinutesUsed     int `json:"total_minutes_used"`
	TotalPaidMinutedUsed int `json:"total_paid_minutes_used"`
	IncludedMinutes      int `json:"included_minutes"`
}

type actionsCollector struct {
	usedMinutesTotal     *prometheus.Desc
	paidMinutedUsedTotal *prometheus.Desc
	includedMinutes      *prometheus.Desc

	mutex  sync.Mutex
	client *http.Client
	logger log.Logger
}

func init() {
	registerCollector("actions", defaultEnabled, NewActionsCollector)
}

// NewActionsCollector returns a new Collector exposing actions billing stats.
func NewActionsCollector(logger log.Logger) (Collector, error) {
	return &actionsCollector{
		usedMinutesTotal: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, actionsSubsystem, "used_minutes"),
			"Total GitHub actions used minutes",
			[]string{"org"}, nil,
		),
		paidMinutedUsedTotal: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, actionsSubsystem, "paid_minutes"),
			"Total GitHub actions paid minutes",
			[]string{"org"}, nil,
		),
		includedMinutes: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, actionsSubsystem, "included_minutes"),
			"GitHub actions included minutes",
			[]string{"org"}, nil,
		),
		client: &http.Client{},
		logger: logger,
	}, nil
}

// Describe implements Collector.
func (ac *actionsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- ac.usedMinutesTotal
	ch <- ac.paidMinutedUsedTotal
	ch <- ac.includedMinutes
}

// Update implements Collector and exposes actions billing stats
// from api.github.com/orgs/<org>/settings/billing/actions.
func (ac *actionsCollector) Update(ch chan<- prometheus.Metric) error {
	orgs := parseArg(*githubOrgs)
	for _, org := range orgs {
		var a actions
		req, _ := http.NewRequest("GET", "https://api.github.com/orgs/"+org+"/settings/billing/actions", nil)
		req.Header.Set("Authorization", "token "+*githubToken)
		resp, err := ac.client.Do(req)
		if err != nil {
			return err
		}

		if resp.StatusCode != 200 {
			return fmt.Errorf("status %s, organization: %s collector: %s", resp.Status, org, actionsSubsystem)
		}

		err = json.NewDecoder(resp.Body).Decode(&a)
		if err != nil {
			return err
		}

		resp.Body.Close()

		ch <- prometheus.MustNewConstMetric(ac.usedMinutesTotal, prometheus.GaugeValue, float64(a.TotalMinutesUsed), org)
		ch <- prometheus.MustNewConstMetric(ac.paidMinutedUsedTotal, prometheus.GaugeValue, float64(a.TotalPaidMinutedUsed), org)
		ch <- prometheus.MustNewConstMetric(ac.includedMinutes, prometheus.GaugeValue, float64(a.IncludedMinutes), org)
	}

	return nil
}
