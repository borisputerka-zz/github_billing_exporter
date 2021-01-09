package collector

import (
	"fmt"
	"gopkg.in/alecthomas/kingpin.v2"
	"sync"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	namespace       = "github_billing"
	defaultEnabled  = true
	defaultDisabled = false
)

var (
	up = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "up"),
		"Can be test_server reached",
		[]string{"collector"}, nil,
	)
)

var (
	factories        = make(map[string]func(logger log.Logger) (Collector, error))
	collectorState   = make(map[string]*bool)
	forcedCollectors = map[string]bool{} // collectors which have been explicitly enabled or disabled
)

var (
	githubToken = kingpin.Flag(
		"github-token",
		"GitHub token to access api",
	).Envar("GITHUB_TOKEN").String()
	githubOrgs = kingpin.Flag("github-orgs",
		"Organizations to get metrics from",
	).Envar("GITHUB_ORGS").String()
)

// Collector is the interface a collector has to implement.
type Collector interface {
	Update(ch chan<- prometheus.Metric) error
}

// BillingCollector implements the prometheus.Collector interface.
type BillingCollector struct {
	Collectors map[string]Collector
	logger     log.Logger
}

func registerCollector(collector string, isDefaultEnabled bool, factory func(logger log.Logger) (Collector, error)) {
	var helpDefaultState string
	if isDefaultEnabled {
		helpDefaultState = "enabled"
	} else {
		helpDefaultState = "disabled"
	}

	flagName := fmt.Sprintf("collector.%s", collector)
	flagHelp := fmt.Sprintf("Enable the %s collector (default: %s).", collector, helpDefaultState)
	defaultValue := fmt.Sprintf("%v", isDefaultEnabled)
	flag := kingpin.Flag(flagName, flagHelp).Default(defaultValue).Bool()
	collectorState[collector] = flag
	factories[collector] = factory
}

// NewBillingCollector creates a new BillingCollector.
func NewBillingCollector(logger log.Logger) (*BillingCollector, error) {
	collectors := make(map[string]Collector)
	for key, enabled := range collectorState {
		if *enabled {
			collector, err := factories[key](log.With(logger, "collector", key))
			if err != nil {
				return nil, err
			}
			collectors[key] = collector
		}
		if !*enabled {
			level.Info(logger).Log("msg", "Collector disabled", "name", key)
		}
	}

	return &BillingCollector{
		Collectors: collectors,
		logger:     logger,
	}, nil
}

// Describe implements the prometheus.Collector interface.
func (n BillingCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- up
}

// Collect implements the prometheus.Collector interface.
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
