package metrics

import (
	"math"

	"github.com/prometheus/client_golang/promauto"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// <extension>RequestDuration provides a histogram of processing times. The
	// buckets should use periods that are intuitive for people.
	//
	// For example, it provides metrics:
	//   allocate_k8s_token_request_duration_seconds{code="...", le="..."}
	//   ...
	//   allocate_k8s_token_request_duration_seconds{code="..."}
	//   allocate_k8s_token_request_duration_seconds{code="..."}

	TokenRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "allocate_k8s_token_request_duration_seconds",
			Help: "Request status codes and execution times.",
			Buckets: []float64{
				0.001, 0.01, 0.1, 1.0, 5.0, 10.0, 30.0, 60.0, 120.0, 300.0, math.Inf(+1),
			},
		},
		[]string{"method", "code"},
	)

	BMCRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "bmc_store_password_request_duration_seconds",
			Help: "Request status codes and execution times.",
			Buckets: []float64{
				0.001, 0.01, 0.1, 1.0, 5.0, 10.0, 30.0, 60.0, 120.0, 300.0, math.Inf(+1),
			},
		},
		[]string{"method", "code"},
	)
)
