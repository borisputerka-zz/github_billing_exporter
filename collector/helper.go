package collector

import "github.com/prometheus/client_golang/prometheus"

var (
	// See Minute Multipliers https://docs.github.com/en/billing/managing-billing-for-github-actions/about-billing-for-github-actions
	PlatformMultiplierLinux   = 1.0
	PlatformMultiplierMacOS   = 10.0
	PlatformMultiplierWindows = 2.0
)

type PlatformMetric struct {
	Linux   float64
	MacOS   float64
	Windows float64
}

func NewPlatformMetric(result chan<- prometheus.Metric, desc *prometheus.Desc, valueType prometheus.ValueType, values PlatformMetric, labelValues ...string) {
	result <- prometheus.MustNewConstMetric(desc, valueType, values.Linux, append(labelValues, "linux")...)
	result <- prometheus.MustNewConstMetric(desc, valueType, values.MacOS, append(labelValues, "macos")...)
	result <- prometheus.MustNewConstMetric(desc, valueType, values.Windows, append(labelValues, "windows")...)
}
