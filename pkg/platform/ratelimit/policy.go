package ratelimit

// Policy names used across services.
const (
	PolicyAuth         = "auth"
	PolicyPasswordReset = "password_reset"
	PolicyPublic       = "public"
	PolicyAPI          = "api"
	PolicyWorkflow     = "workflow"
	PolicyWebhook      = "webhook"
	PolicyHealth       = "health"
)

// PolicySet holds configured limits for named policies.
type PolicySet struct {
	Auth          Policy
	PasswordReset Policy
	Public        Policy
	API           Policy
	Workflow      Policy
	Webhook       Policy
}

// Get returns a policy by name.
func (p PolicySet) Get(name string) Policy {
	switch name {
	case PolicyAuth:
		return p.Auth
	case PolicyPasswordReset:
		return p.PasswordReset
	case PolicyPublic:
		return p.Public
	case PolicyWorkflow:
		return p.Workflow
	case PolicyWebhook:
		return p.Webhook
	default:
		return p.API
	}
}

// GatewayPolicySet returns gateway-oriented defaults from configuration.
func GatewayPolicySet(cfg Config) PolicySet {
	return PolicySet{
		Auth: Policy{Name: PolicyAuth, Limit: int64(cfg.AuthRequests), Window: cfg.AuthWindow, FailClosed: true},
		Public: Policy{Name: PolicyPublic, Limit: int64(cfg.DefaultRequests), Window: cfg.DefaultWindow},
		API: Policy{Name: PolicyAPI, Limit: int64(cfg.DefaultRequests), Window: cfg.DefaultWindow},
		Workflow: Policy{Name: PolicyWorkflow, Limit: int64(cfg.WorkflowRequests), Window: cfg.WorkflowWindow},
		Webhook: Policy{Name: PolicyWebhook, Limit: int64(cfg.DefaultRequests), Window: cfg.DefaultWindow},
		PasswordReset: Policy{Name: PolicyPasswordReset, Limit: maxInt64(1, int64(cfg.AuthRequests/2)), Window: cfg.AuthWindow, FailClosed: true},
	}
}

// AuthPolicySet returns auth-service policies.
func AuthPolicySet(cfg Config) PolicySet {
	return PolicySet{
		Auth: Policy{Name: PolicyAuth, Limit: int64(cfg.AuthRequests), Window: cfg.AuthWindow, FailClosed: true},
		API: Policy{Name: PolicyAPI, Limit: int64(cfg.AuthRequests), Window: cfg.AuthWindow, FailClosed: true},
	}
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
