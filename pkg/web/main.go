package web

import (
	"net/http"

	"github.com/go-kit/log/level"
	"github.com/raynigon/github_billing_exporter/v2/pkg/config"
)

func RunWebserver(config config.GitHubBillingExporterConfig) {
	logger := config.GetLogger()
	listeningAddress := config.GetListeningAccess()
	level.Info(logger).Log("msg", "Starting Server", "listening address", listeningAddress)
	registerController(config)
	err := http.ListenAndServe(listeningAddress, nil)
	level.Error(logger).Log("msg", err)
}
