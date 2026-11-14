// Package metrics holds the multi-tenancy domain counters.
//
// They live here rather than in cmd/ so the handlers package can increment them
// without importing main. Registration happens in this package's init, so the
// /metrics endpoint in cmd/ exposes them through the default registry.
package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	// TenantsCreated counts successfully provisioned tenants.
	TenantsCreated = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "multi_tenancy_tenants_created_total",
		Help: "Tenants provisioned.",
	})

	// QuotaChecks counts quota evaluations by outcome. A non-zero "exceeded"
	// rate is the signal that a tenant is hitting its plan limit.
	QuotaChecks = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "multi_tenancy_quota_checks_total",
			Help: "Quota checks by outcome.",
		},
		[]string{"outcome"},
	)
)

func init() {
	prometheus.MustRegister(TenantsCreated, QuotaChecks)
}
