package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics records organization service observability signals.
type Metrics struct {
	organizationsCreated   prometheus.Counter
	organizationsUpdated   prometheus.Counter
	organizationsDeleted   prometheus.Counter
	membershipsCreated     prometheus.Counter
	membershipsRemoved     prometheus.Counter
	membershipRoleChanges  prometheus.Counter
	invitationsCreated     prometheus.Counter
	invitationsAccepted    prometheus.Counter
	invitationsRevoked     prometheus.Counter
	authorizationDenied    prometheus.Counter
	crossTenantDenied      prometheus.Counter
	lookupLatency          prometheus.Histogram
	permissionChecks       *prometheus.CounterVec
}

// New creates organization metrics.
func New(service string) *Metrics {
	return &Metrics{
		organizationsCreated: promauto.NewCounter(prometheus.CounterOpts{
			Name: "organizations_created_total",
			Help: "Total organizations created.",
		}),
		organizationsUpdated: promauto.NewCounter(prometheus.CounterOpts{
			Name: "organizations_updated_total",
			Help: "Total organizations updated.",
		}),
		organizationsDeleted: promauto.NewCounter(prometheus.CounterOpts{
			Name: "organizations_deleted_total",
			Help: "Total organizations deleted.",
		}),
		membershipsCreated: promauto.NewCounter(prometheus.CounterOpts{
			Name: "memberships_created_total",
			Help: "Total memberships created.",
		}),
		membershipsRemoved: promauto.NewCounter(prometheus.CounterOpts{
			Name: "memberships_removed_total",
			Help: "Total memberships removed.",
		}),
		membershipRoleChanges: promauto.NewCounter(prometheus.CounterOpts{
			Name: "role_changes_total",
			Help: "Total membership role changes.",
		}),
		invitationsCreated: promauto.NewCounter(prometheus.CounterOpts{
			Name: "invitations_created_total",
			Help: "Total invitations created.",
		}),
		invitationsAccepted: promauto.NewCounter(prometheus.CounterOpts{
			Name: "invitations_accepted_total",
			Help: "Total invitations accepted.",
		}),
		invitationsRevoked: promauto.NewCounter(prometheus.CounterOpts{
			Name: "invitations_revoked_total",
			Help: "Total invitations revoked.",
		}),
		authorizationDenied: promauto.NewCounter(prometheus.CounterOpts{
			Name: "authorization_denied_total",
			Help: "Total authorization denials.",
		}),
		crossTenantDenied: promauto.NewCounter(prometheus.CounterOpts{
			Name: "cross_tenant_access_denied_total",
			Help: "Total cross-tenant access denials.",
		}),
		lookupLatency: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "halcyon_" + service + "_organization_lookup_latency_seconds",
			Help:    "Organization lookup latency in seconds.",
			Buckets: prometheus.DefBuckets,
		}),
		permissionChecks: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "halcyon_" + service + "_permission_checks_total",
			Help: "Total permission checks.",
		}, []string{"result"}),
	}
}

func (m *Metrics) RecordOrganizationCreated()   { m.organizationsCreated.Inc() }
func (m *Metrics) RecordOrganizationUpdated()   { m.organizationsUpdated.Inc() }
func (m *Metrics) RecordOrganizationDeleted()   { m.organizationsDeleted.Inc() }
func (m *Metrics) RecordMembershipCreated()   { m.membershipsCreated.Inc() }
func (m *Metrics) RecordMembershipRemoved()   { m.membershipsRemoved.Inc() }
func (m *Metrics) RecordMembershipRoleChanged() { m.membershipRoleChanges.Inc() }
func (m *Metrics) RecordInvitationCreated()   { m.invitationsCreated.Inc() }
func (m *Metrics) RecordInvitationAccepted()  { m.invitationsAccepted.Inc() }
func (m *Metrics) RecordInvitationRevoked()   { m.invitationsRevoked.Inc() }

func (m *Metrics) RecordMembershipChange(delta int) {
	if delta > 0 {
		m.membershipsCreated.Add(float64(delta))
	} else if delta < 0 {
		m.membershipsRemoved.Add(float64(-delta))
	}
}

func (m *Metrics) ObserveOrganizationLookup(start time.Time) {
	m.lookupLatency.Observe(time.Since(start).Seconds())
}

func (m *Metrics) RecordPermissionCheck(granted bool) {
	result := "denied"
	if granted {
		result = "granted"
	}
	m.permissionChecks.WithLabelValues(result).Inc()
}

func (m *Metrics) RecordAuthorizationDenied(crossTenant bool) {
	m.authorizationDenied.Inc()
	if crossTenant {
		m.crossTenantDenied.Inc()
	}
}
