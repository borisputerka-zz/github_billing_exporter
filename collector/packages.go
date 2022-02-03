package collector

import (
	"context"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/raynigon/github_billing_exporter/v2/pkg/gh_org"
)

var (
	orgPackagesSubsystem = "packages_org"
)

type OrgPackagesCollector struct {
	config            CollectorConfig
	metrics           map[string]*gh_org.GitHubOrgMetrics
	usedBandwithTotal *prometheus.Desc
	usedBandwithPaid  *prometheus.Desc
	inclusiveBandwith *prometheus.Desc
}

func init() {
	registerCollector(orgPackagesSubsystem, NewOrgPackagesCollectorCollector)
}

// NewOrgActionsCollector returns a new Collector exposing actions billing stats.
func NewOrgPackagesCollectorCollector(config CollectorConfig, ctx context.Context) (Collector, error) {
	collector := &OrgPackagesCollector{
		config:  config,
		metrics: make(map[string]*gh_org.GitHubOrgMetrics),
		usedBandwithTotal: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, orgPackagesSubsystem, "bandwith_total_count"),
			"GitHub packages total used bandwith in gigabytes",
			[]string{"org"}, nil,
		),
		usedBandwithPaid: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, orgPackagesSubsystem, "bandwith_paid_count"),
			"GitHub packages paid used bandwith in gigabytes",
			[]string{"org"}, nil,
		),
		inclusiveBandwith: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, orgPackagesSubsystem, "bandwith_inclusive"),
			"GitHub packages inclusive budget bandwith in gigabytes",
			[]string{"org"}, nil,
		),
	}
	err := collector.Reload(ctx)
	if err != nil {
		return nil, err
	}
	return collector, nil
}

// Describe implements Collector.
func (oac *OrgPackagesCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- oac.usedBandwithTotal
	ch <- oac.usedBandwithPaid
	ch <- oac.inclusiveBandwith
}

func (oac *OrgPackagesCollector) Reload(ctx context.Context) error {
	metrics := make(map[string]*gh_org.GitHubOrgMetrics)
	for _, org := range oac.config.Orgs {
		metrics[org] = gh_org.NewGitHubOrgMetrics(oac.config.Github, org)
	}
	oac.metrics = metrics
	return nil
}

func (oac *OrgPackagesCollector) Update(ctx context.Context, ch chan<- prometheus.Metric) error {
	wg := sync.WaitGroup{}
	wg.Add(len(oac.metrics))
	errors := make(chan error, len(oac.metrics))
	for org, githubOrg := range oac.metrics {
		go func(org string, githubOrg *gh_org.GitHubOrgMetrics) {
			metrics, err := githubOrg.CollectPackages(ctx)
			if err != nil {
				errors <- err
				wg.Done()
				return
			}

			ch <- prometheus.MustNewConstMetric(oac.usedBandwithTotal, prometheus.CounterValue, float64(metrics.TotalGigabytesBandwidthUsed), org)
			ch <- prometheus.MustNewConstMetric(oac.usedBandwithPaid, prometheus.CounterValue, float64(metrics.TotalPaidGigabytesBandwidthUsed), org)
			ch <- prometheus.MustNewConstMetric(oac.inclusiveBandwith, prometheus.GaugeValue, float64(metrics.IncludedGigabytesBandwidth), org)
			wg.Done()
		}(org, githubOrg)
	}
	wg.Wait()
	close(errors)
	for error := range errors {
		return error
	}
	return nil
}
