package collector

import (
	"sync"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
)

const namespace = "github_billing"

var (
	up = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "up"),
		"Can be test_server reached",
		[]string{"collector"}, nil,
	)
)

var (
	factories = make(map[string]func(githubOrgs string, githubToken string) Collector)
)

type Collector interface {
	Update(ch chan<- prometheus.Metric) error
}

type BillingCollector struct {
	Collectors  map[string]Collector
	githubOrgs  string
	githubToken string
	logger      log.Logger
}

func registerCollector(collector string, factory func(githubOrgs string, githubToken string) Collector) {
	factories[collector] = factory
}

func NewBillingCollector(githubOrgs string, githubToken string, disabledCollectors string, logger log.Logger) *BillingCollector {
	collectors := make(map[string]Collector)
	disabledCollectorsList := parseArg(disabledCollectors)
	for _, disabledCollector := range disabledCollectorsList {
		delete(factories, disabledCollector)
	}
	for key, collector := range factories {
		collectors[key] = collector(githubOrgs, githubToken)
	}
	return &BillingCollector{
		Collectors:  collectors,
		githubOrgs:  githubOrgs,
		githubToken: githubToken,
		logger:      logger,
	}
}

func (n BillingCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- up
}

func (n BillingCollector) Collect(ch chan<- prometheus.Metric) {
	wg := sync.WaitGroup{}
	wg.Add(len(n.Collectors))
	for name, c := range n.Collectors {
		go func(name string, c Collector) {
			execute(name, c, ch, n.logger)
			wg.Done()
		}(name, c)
	}
	wg.Wait()
}

func execute(name string, c Collector, ch chan<- prometheus.Metric, logger log.Logger) {
	var success float64

	err := c.Update(ch)
	if err != nil {
		level.Error(logger).Log("msg", "Cannot collect metrics", "err", err)
		success = 0
	} else {
		success = 1
	}

	ch <- prometheus.MustNewConstMetric(up, prometheus.GaugeValue, success, name)
}
