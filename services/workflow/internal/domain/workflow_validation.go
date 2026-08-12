package domain

import (
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"
)

// Workflow limits (configurable via environment in application layer).
const (
	DefaultMaxWorkflowSteps     = 50
	DefaultMaxStepTimeout       = 5 * time.Minute
	DefaultMaxRetryAttempts     = 10
	DefaultMaxWorkflowNameLen   = 128
	DefaultMaxTriggerDepth      = 5
	DefaultMaxInlineOutputBytes = 64 * 1024
)

// CreateWorkflowInput carries workflow creation data.
type CreateWorkflowInput struct {
	Name  string
	Steps []StepInput
}

// StepInput carries step definition input.
type StepInput struct {
	StepKey      string          `json:"step_key"`
	Name         string          `json:"name"`
	Type         string          `json:"type"`
	Configuration json.RawMessage `json:"configuration"`
	Timeout      string          `json:"timeout"`
	RetryPolicy  *RetryPolicy    `json:"retry_policy,omitempty"`
	Dependencies []string        `json:"dependencies"`
}

// ValidateCreateWorkflow validates workflow creation input.
func ValidateCreateWorkflow(input CreateWorkflowInput, limits WorkflowLimits) error {
	name := strings.TrimSpace(input.Name)
	if name == "" || len(name) > limits.MaxWorkflowNameLen {
		return ErrInvalidWorkflowName
	}
	if len(input.Steps) == 0 {
		return fmt.Errorf("%w: at least one step required", ErrInvalidWorkflowDefinition)
	}
	if len(input.Steps) > limits.MaxWorkflowSteps {
		return ErrWorkflowLimitExceeded
	}

	seen := make(map[string]struct{}, len(input.Steps))
	for i, step := range input.Steps {
		key := strings.TrimSpace(step.StepKey)
		if key == "" {
			return fmt.Errorf("%w: step_key required at index %d", ErrInvalidWorkflowDefinition, i)
		}
		if _, ok := seen[key]; ok {
			return fmt.Errorf("%w: duplicate step_key %q", ErrInvalidWorkflowDefinition, key)
		}
		seen[key] = struct{}{}

		stepType := StepType(strings.ToUpper(strings.TrimSpace(step.Type)))
		switch stepType {
		case StepTypeHTTPAction, StepTypeEventEmit, StepTypeDelay:
		default:
			return ErrInvalidStepType
		}

		timeout := limits.DefaultStepTimeout
		if step.Timeout != "" {
			parsed, err := time.ParseDuration(step.Timeout)
			if err != nil || parsed <= 0 {
				return fmt.Errorf("%w: invalid timeout for step %q", ErrInvalidWorkflowDefinition, key)
			}
			if parsed > limits.MaxStepTimeout {
				return ErrWorkflowLimitExceeded
			}
			timeout = parsed
		}

		policy := DefaultRetryPolicy()
		if step.RetryPolicy != nil {
			policy = *step.RetryPolicy
		}
		if policy.MaxAttempts <= 0 || policy.MaxAttempts > limits.MaxRetryAttempts {
			return ErrWorkflowLimitExceeded
		}

		if err := validateStepConfiguration(stepType, step.Configuration, timeout); err != nil {
			return err
		}

		for _, dep := range step.Dependencies {
			if strings.TrimSpace(dep) == "" {
				return fmt.Errorf("%w: empty dependency for step %q", ErrInvalidWorkflowDefinition, key)
			}
		}
	}
	return nil
}

// WorkflowLimits configures validation boundaries.
type WorkflowLimits struct {
	MaxWorkflowSteps    int
	MaxStepTimeout      time.Duration
	MaxRetryAttempts    int
	MaxWorkflowNameLen  int
	DefaultStepTimeout  time.Duration
	MaxTriggerDepth     int
}

// DefaultWorkflowLimits returns safe defaults.
func DefaultWorkflowLimits() WorkflowLimits {
	return WorkflowLimits{
		MaxWorkflowSteps:   DefaultMaxWorkflowSteps,
		MaxStepTimeout:     DefaultMaxStepTimeout,
		MaxRetryAttempts:   DefaultMaxRetryAttempts,
		MaxWorkflowNameLen: DefaultMaxWorkflowNameLen,
		DefaultStepTimeout: 30 * time.Second,
		MaxTriggerDepth:    DefaultMaxTriggerDepth,
	}
}

func validateStepConfiguration(stepType StepType, raw json.RawMessage, timeout time.Duration) error {
	switch stepType {
	case StepTypeHTTPAction:
		var cfg HTTPActionConfig
		if err := json.Unmarshal(raw, &cfg); err != nil {
			return fmt.Errorf("%w: invalid HTTP_ACTION config: %v", ErrInvalidWorkflowDefinition, err)
		}
		if err := cfg.Validate(); err != nil {
			return err
		}
	case StepTypeEventEmit:
		var cfg EventEmitConfig
		if err := json.Unmarshal(raw, &cfg); err != nil {
			return fmt.Errorf("%w: invalid EVENT_EMIT config: %v", ErrInvalidWorkflowDefinition, err)
		}
		if err := cfg.Validate(); err != nil {
			return err
		}
	case StepTypeDelay:
		var cfg DelayConfig
		if err := json.Unmarshal(raw, &cfg); err != nil {
			return fmt.Errorf("%w: invalid DELAY config: %v", ErrInvalidWorkflowDefinition, err)
		}
		if err := cfg.Validate(timeout); err != nil {
			return err
		}
	}
	return nil
}

// HTTPActionConfig configures an HTTP_ACTION step.
type HTTPActionConfig struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
}

func (c HTTPActionConfig) Validate() error {
	method := strings.ToUpper(strings.TrimSpace(c.Method))
	if method != "GET" && method != "POST" && method != "PUT" && method != "PATCH" && method != "DELETE" {
		return fmt.Errorf("%w: unsupported HTTP method", ErrInvalidWorkflowDefinition)
	}
	return ValidateHTTPURL(c.URL)
}

// EventEmitConfig configures an EVENT_EMIT step.
type EventEmitConfig struct {
	EventType string          `json:"event_type"`
	Payload   json.RawMessage `json:"payload"`
}

func (c EventEmitConfig) Validate() error {
	if strings.TrimSpace(c.EventType) == "" {
		return fmt.Errorf("%w: event_type required", ErrInvalidWorkflowDefinition)
	}
	return nil
}

// DelayConfig configures a DELAY step.
type DelayConfig struct {
	Duration string `json:"duration"`
}

func (c DelayConfig) Validate(stepTimeout time.Duration) error {
	d, err := time.ParseDuration(strings.TrimSpace(c.Duration))
	if err != nil || d <= 0 {
		return fmt.Errorf("%w: invalid delay duration", ErrInvalidWorkflowDefinition)
	}
	if d > stepTimeout {
		return fmt.Errorf("%w: delay exceeds step timeout", ErrInvalidWorkflowDefinition)
	}
	return nil
}

// ValidateHTTPURL blocks SSRF to private networks and metadata endpoints.
func ValidateHTTPURL(raw string) error {
	raw = strings.TrimSpace(raw)
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return fmt.Errorf("%w: invalid url", ErrInvalidWorkflowDefinition)
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("%w: only http/https allowed", ErrInvalidWorkflowDefinition)
	}

	host := strings.ToLower(u.Hostname())
	if host == "localhost" || host == "metadata.google.internal" {
		return ErrSSRFBlocked
	}
	if ip := net.ParseIP(host); ip != nil {
		if isBlockedIP(ip) {
			return ErrSSRFBlocked
		}
	}
	return nil
}

func isBlockedIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsPrivate() {
		return true
	}
	// AWS/GCP metadata
	if ip.Equal(net.ParseIP("169.254.169.254")) {
		return true
	}
	return false
}

// DependenciesSatisfied reports whether all dependencies completed successfully.
func DependenciesSatisfied(deps []string, completed map[string]StepStatus) bool {
	for _, dep := range deps {
		status, ok := completed[dep]
		if !ok || status != StepSucceeded {
			return false
		}
	}
	return true
}

// NextRunnableStep returns the next step by order whose dependencies are satisfied.
func NextRunnableStep(steps []WorkflowStepDefinition, completed map[string]StepStatus) (*WorkflowStepDefinition, bool) {
	for i := range steps {
		step := &steps[i]
		if status, ok := completed[step.StepKey]; ok {
			if status == StepSucceeded || status == StepRunning || status == StepFailed || status == StepCancelled {
				continue
			}
		}
		if DependenciesSatisfied(step.Dependencies, completed) {
			return step, true
		}
	}
	return nil, false
}
