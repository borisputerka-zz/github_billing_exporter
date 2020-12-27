package collector

import (
	"encoding/json"
	"github.com/prometheus/client_golang/prometheus"
	"net/http"
	"sync"
)

var ()

type actions struct {
	TotalMinutesUsed     int `json:"total_minutes_used"`
	TotalPaidMinutedUsed int `json:"total_paid_minutes_used"`
	IncludedMinutes      int `json:"included_minutes"`
}

type ActionsCollector struct {
	GithubOrgs  string
	GithubToken string

	usedMinutesTotal     *prometheus.Desc
	paidMinutedUsedTotal *prometheus.Desc
	includedMinutes      *prometheus.Desc

	mutex  sync.Mutex
	client *http.Client
}

func init() {
	registerCollector("actions", NewActionsCollector)
}

func NewActionsCollector(githubOrgs string, githubToken string) Collector {
	return &ActionsCollector{
		GithubOrgs:  githubOrgs,
		GithubToken: githubToken,
		usedMinutesTotal: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "actions", "used_minutes"),
			"Total GitHub actions used minutes",
			[]string{"org"}, nil,
		),
		paidMinutedUsedTotal: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "actions", "paid_minutes"),
			"Total GitHub actions paid minutes",
			[]string{"org"}, nil,
		),
		includedMinutes: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "actions", "included_minutes"),
			"GitHub actions included minutes",
			[]string{"org"}, nil,
		),
		client: &http.Client{},
	}
}

func (ac *ActionsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- ac.usedMinutesTotal
	ch <- ac.paidMinutedUsedTotal
	ch <- ac.includedMinutes
}

func (ac *ActionsCollector) Update(ch chan<- prometheus.Metric) error {
	orgs := parseArg(ac.GithubOrgs)
	for _, org := range orgs {
		var a actions
		req, _ := http.NewRequest("GET", "/orgs/"+org+"/settings/billing/actions", nil)
		req.Header.Set("Authorization", "token "+ac.GithubToken)
		resp, err := ac.client.Do(req)
		if err != nil {
			return err
		}
		if resp.StatusCode != 200 {
			return err
		}

		err = json.NewDecoder(resp.Body).Decode(&a)
		if err != nil {
			return err
		}

		ch <- prometheus.MustNewConstMetric(ac.usedMinutesTotal, prometheus.GaugeValue, float64(a.TotalMinutesUsed), org)
		ch <- prometheus.MustNewConstMetric(ac.paidMinutedUsedTotal, prometheus.GaugeValue, float64(a.TotalPaidMinutedUsed), org)
		ch <- prometheus.MustNewConstMetric(ac.includedMinutes, prometheus.GaugeValue, float64(a.IncludedMinutes), org)
	}

	return nil
}
