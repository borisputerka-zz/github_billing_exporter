package collector

import (
	"context"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/raynigon/github_billing_exporter/v2/pkg/gh_org"
)

var (
	orgActionsSubsystem = "actions_org"
)

type OrgActionsCollector struct {
	config            CollectorConfig
	metrics           map[string]*gh_org.GitHubOrgMetrics
	usedMinutesReal   *prometheus.Desc
	usedMinutesBilled *prometheus.Desc
	inclusiveMinutes  *prometheus.Desc
	usedMinutesTotal  *prometheus.Desc
	usedMinutesPaid   *prometheus.Desc
}

func init() {
	registerCollector(orgActionsSubsystem, NewOrgActionsCollector)
}

// NewOrgActionsCollector returns a new Collector exposing actions billing stats.
func NewOrgActionsCollector(config CollectorConfig, ctx context.Context) (Collector, error) {
	collector := &OrgActionsCollector{
		config:  config,
		metrics: make(map[string]*gh_org.GitHubOrgMetrics),
		usedMinutesReal: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, orgActionsSubsystem, "minutes_real_count"),
			"GitHub actions used minutes without platform multiplier",
			[]string{"org", "platform"}, nil,
		),
		usedMinutesBilled: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, orgActionsSubsystem, "minutes_billed_count"),
			"GitHub actions used minutes with platform multipliers",
			[]string{"org", "platform"}, nil,
		),
		inclusiveMinutes: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, orgActionsSubsystem, "minutes_inclusive"),
			"GitHub actions inclusive budget minutes",
			[]string{"org"}, nil,
		),
		usedMinutesTotal: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, orgActionsSubsystem, "minutes_total_count"),
			"Total GitHub actions minutes used",
			[]string{"org"}, nil,
		),
		usedMinutesPaid: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, orgActionsSubsystem, "minutes_paid_count"),
			"Total GitHub actions minutes paid for",
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
func (oac *OrgActionsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- oac.usedMinutesReal
	ch <- oac.usedMinutesBilled
	ch <- oac.inclusiveMinutes
	ch <- oac.usedMinutesTotal
	ch <- oac.usedMinutesPaid
}

func (oac *OrgActionsCollector) Reload(ctx context.Context) error {
	metrics := make(map[string]*gh_org.GitHubOrgMetrics)
	for _, org := range oac.config.Orgs {
		metrics[org] = gh_org.NewGitHubOrgMetrics(oac.config.Github, org)
	}
	oac.metrics = metrics
	return nil
}

func (oac *OrgActionsCollector) Update(ctx context.Context, ch chan<- prometheus.Metric) error {
	wg := sync.WaitGroup{}
	wg.Add(len(oac.metrics))
	errors := make(chan error, len(oac.metrics))
	for org, githubOrg := range oac.metrics {
		go func(org string, githubOrg *gh_org.GitHubOrgMetrics) {
			metrics, err := githubOrg.CollectActions(ctx)
			if err != nil {
				errors <- err
				wg.Done()
				return
			}
			// Use the original billed values
			NewPlatformMetric(ch, oac.usedMinutesBilled, prometheus.CounterValue, PlatformMetric{
				Linux:   float64(metrics.MinutesUsedBreakdown.Ubuntu),
				MacOS:   float64(metrics.MinutesUsedBreakdown.MacOS),
				Windows: float64(metrics.MinutesUsedBreakdown.Windows),
			}, org)
			// See Minute Multipliers https://docs.github.com/en/billing/managing-billing-for-github-actions/about-billing-for-github-actions
			NewPlatformMetric(ch, oac.usedMinutesReal, prometheus.CounterValue, PlatformMetric{
				Linux:   float64(metrics.MinutesUsedBreakdown.Ubuntu) / PlatformMultiplierLinux,
				MacOS:   float64(metrics.MinutesUsedBreakdown.MacOS) / PlatformMultiplierMacOS,
				Windows: float64(metrics.MinutesUsedBreakdown.Windows) / PlatformMultiplierWindows,
			}, org)
			// Platform independent metrics
			ch <- prometheus.MustNewConstMetric(oac.inclusiveMinutes, prometheus.GaugeValue, float64(metrics.IncludedMinutes), org)
			ch <- prometheus.MustNewConstMetric(oac.usedMinutesTotal, prometheus.CounterValue, float64(metrics.TotalMinutesUsed), org)
			ch <- prometheus.MustNewConstMetric(oac.usedMinutesPaid, prometheus.CounterValue, float64(metrics.TotalPaidMinutesUsed), org)
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
