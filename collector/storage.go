package collector

import (
	"context"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/raynigon/github_billing_exporter/v2/pkg/gh_org"
)

var (
	orgStorageSubsystem = "storage_org"
)

type OrgStorageCollector struct {
	config           CollectorConfig
	metrics          map[string]*gh_org.GitHubOrgMetrics
	usedStorageTotal *prometheus.Desc
	usedStoragePaid  *prometheus.Desc
	billingCycleDays *prometheus.Desc
}

func init() {
	registerCollector(orgStorageSubsystem, NewOrgStorageCollector)
}

// NewOrgActionsCollector returns a new Collector exposing actions billing stats.
func NewOrgStorageCollector(config CollectorConfig, ctx context.Context) (Collector, error) {
	collector := &OrgStorageCollector{
		config:  config,
		metrics: make(map[string]*gh_org.GitHubOrgMetrics),
		usedStorageTotal: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, orgStorageSubsystem, "total_count"),
			"GitHub storage used total in gigabytes",
			[]string{"org"}, nil,
		),
		usedStoragePaid: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, orgStorageSubsystem, "paid_count"),
			"GitHub storage used paid in gigabytes",
			[]string{"org"}, nil,
		),
		billingCycleDays: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, orgStorageSubsystem, "billing_cycle_days"),
			"Days left in the current billing cycle",
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
func (oac *OrgStorageCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- oac.usedStorageTotal
	ch <- oac.usedStoragePaid
}

func (oac *OrgStorageCollector) Reload(ctx context.Context) error {
	metrics := make(map[string]*gh_org.GitHubOrgMetrics)
	for _, org := range oac.config.Orgs {
		metrics[org] = gh_org.NewGitHubOrgMetrics(oac.config.Github, org)
	}
	oac.metrics = metrics
	return nil
}

func (oac *OrgStorageCollector) Update(ctx context.Context, ch chan<- prometheus.Metric) error {
	wg := sync.WaitGroup{}
	wg.Add(len(oac.metrics))
	errors := make(chan error, len(oac.metrics))
	for org, githubOrg := range oac.metrics {
		go func(org string, githubOrg *gh_org.GitHubOrgMetrics) {
			metrics, err := githubOrg.CollectStorage(ctx)
			if err != nil {
				errors <- err
				wg.Done()
				return
			}

			ch <- prometheus.MustNewConstMetric(oac.usedStorageTotal, prometheus.CounterValue, float64(metrics.EstimatedStorageForMonth), org)
			ch <- prometheus.MustNewConstMetric(oac.usedStoragePaid, prometheus.CounterValue, float64(metrics.EstimatedPaidStorageForMonth), org)
			ch <- prometheus.MustNewConstMetric(oac.billingCycleDays, prometheus.GaugeValue, float64(metrics.DaysLeftInBillingCycle), org)
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
