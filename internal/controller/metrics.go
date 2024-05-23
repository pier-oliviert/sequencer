package controller

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	buildSuccessCounter = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: "sequencer",
			Subsystem: "build",
			Name:      "success",
			Help:      "Number of builds that completed succesfully",
		},
	)

	buildErrorCounter = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: "sequencer",
			Subsystem: "build",
			Name:      "error",
			Help:      "Number of builds that failed",
		},
	)
)

func init() {
	metrics.Registry.MustRegister(buildSuccessCounter, buildErrorCounter)
}
