package web

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/raynigon/github_billing_exporter/v2/pkg/config"
)

func registerController(config config.GitHubBillingExporterConfig) {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { indexController(w, r, config) })
	http.Handle(config.GetMetricsPath(), promhttp.Handler())
}

func indexController(w http.ResponseWriter, r *http.Request, config config.GitHubBillingExporterConfig) {
	w.Header().Add("Content-Type", "text/html")
	w.WriteHeader(200)
	_, _ = w.Write([]byte(`<html>
		<head><title>GitHub Exporter</title></head>
		<body>
		<h1>GitHub Exporter</h1>
		<p><a href="` + config.GetMetricsPath() + `">Metrics</a></p>
		</body>
		</html>`))
}
