package idempotency

// Policy controls idempotency behavior for a route.
type Policy string

const (
	Disabled Policy = "disabled"
	Optional Policy = "optional"
	Required Policy = "required"
)

// PolicyMap maps "METHOD /route/pattern" to a policy.
type PolicyMap map[string]Policy

// Lookup returns the policy for a route key, defaulting to Disabled.
func (m PolicyMap) Lookup(method, routePattern string) Policy {
	if m == nil {
		return Disabled
	}
	if p, ok := m[method+" "+routePattern]; ok {
		return p
	}
	return Disabled
}
