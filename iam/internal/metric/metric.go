package metric

import (
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

var requestCounter = prometheus.NewCounter(
	prometheus.CounterOpts{
		Name: "request_count",
		Help: "Request handled by handler",
		ConstLabels: prometheus.Labels{
			"name": os.Getenv("NAME"),
		},
	},
)

func CreateRegistry() *prometheus.Registry {
	register := prometheus.NewRegistry()
	register.MustRegister(requestCounter)
	register.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	register.MustRegister(collectors.NewGoCollector())
	return register
}
