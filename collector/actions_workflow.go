package collector

import (
	"context"
	"strconv"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/raynigon/github_billing_exporter/v2/pkg/gh_workflow"
)

var (
	repoActionsSubsystem = "actions_workflow"
)

type WorkflowActionsCollector struct {
	config            CollectorConfig
	metrics           map[string]*gh_workflow.GitHubWorkflowMetrics
	usedMinutesReal   *prometheus.Desc
	usedMinutesBilled *prometheus.Desc
}

func init() {
	registerCollector(repoActionsSubsystem, NewWorkflowActionsCollector)
}

// NewRepoActionsCollector returns a new Collector exposing actions billing stats.
func NewWorkflowActionsCollector(config CollectorConfig, ctx context.Context) (Collector, error) {
	collector := &WorkflowActionsCollector{
		config:  config,
		metrics: make(map[string]*gh_workflow.GitHubWorkflowMetrics),
		usedMinutesReal: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, repoActionsSubsystem, "minutes_real_count"),
			"GitHub actions used minutes without platform multiplier",
			[]string{"org", "repository", "workflow_name", "workflow_id", "platform"}, nil,
		),
		usedMinutesBilled: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, repoActionsSubsystem, "minutes_billed_count"),
			"GitHub actions used minutes with platform multipliers",
			[]string{"org", "repository", "workflow_name", "workflow_id", "platform"}, nil,
		),
	}
	err := collector.Reload(ctx)
	if err != nil {
		return nil, err
	}
	return collector, nil
}

// Describe implements Collector.
func (wac *WorkflowActionsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- wac.usedMinutesReal
	ch <- wac.usedMinutesBilled
}

func (wac *WorkflowActionsCollector) Reload(ctx context.Context) error {
	metrics := make(map[string]*gh_workflow.GitHubWorkflowMetrics)
	for _, org := range wac.config.Orgs {
		metrics[org] = gh_workflow.NewGitHubWorkflowMetrics(wac.config.Github, org, ctx)
	}
	wac.metrics = metrics
	return nil
}

func (wac *WorkflowActionsCollector) Update(ctx context.Context, ch chan<- prometheus.Metric) error {
	wg := sync.WaitGroup{}
	wg.Add(len(wac.metrics))
	for org, repoMetrics := range wac.metrics {
		go func(org string, repoMetrics *gh_workflow.GitHubWorkflowMetrics) {
			for _, workflow := range repoMetrics.CollectActions(ctx) {
				// Convert milliseconds to minutes
				realUsageLinux := 0.0
				realUsageMacOS := 0.0
				realUsageWindows := 0.0

				if workflow.Usage.Billable.Ubuntu != nil {
					realUsageLinux = float64(*workflow.Usage.Billable.Ubuntu.TotalMS) / 60_000.0
				}
				if workflow.Usage.Billable.MacOS != nil {
					realUsageMacOS = float64(*workflow.Usage.Billable.MacOS.TotalMS) / 60_000.0
				}
				if workflow.Usage.Billable.Windows != nil {
					realUsageWindows = float64(*workflow.Usage.Billable.Windows.TotalMS) / 60_000.0
				}

				// Use the real value in minutes
				NewPlatformMetric(ch, wac.usedMinutesReal, prometheus.CounterValue, PlatformMetric{
					Linux:   realUsageLinux,
					MacOS:   realUsageMacOS,
					Windows: realUsageWindows,
				}, org, *workflow.Repository.Name, *workflow.Workflow.Name, strconv.FormatInt(*workflow.Workflow.ID, 10))
				// Use the original billed values
				// See Minute Multipliers https://docs.github.com/en/billing/managing-billing-for-github-actions/about-billing-for-github-actions
				NewPlatformMetric(ch, wac.usedMinutesBilled, prometheus.CounterValue, PlatformMetric{
					Linux:   realUsageLinux * PlatformMultiplierLinux,
					MacOS:   realUsageMacOS * PlatformMultiplierMacOS,
					Windows: realUsageWindows * PlatformMultiplierWindows,
				}, org, *workflow.Repository.Name, *workflow.Workflow.Name, strconv.FormatInt(*workflow.Workflow.ID, 10))
			}
			wg.Done()
		}(org, repoMetrics)
	}
	wg.Wait()
	return nil
}
