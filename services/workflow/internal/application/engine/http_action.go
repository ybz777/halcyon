package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/halcyon-io/halcyon/services/workflow/internal/domain"
)

// HTTPActionExecutor executes HTTP_ACTION steps.
type HTTPActionExecutor struct {
	client *http.Client
}

func NewHTTPActionExecutor() *HTTPActionExecutor {
	return &HTTPActionExecutor{client: &http.Client{}}
}

func (e *HTTPActionExecutor) Type() domain.StepType { return domain.StepTypeHTTPAction }

func (e *HTTPActionExecutor) Execute(ctx context.Context, step domain.WorkflowStepDefinition, execCtx StepContext) (StepResult, error) {
	var cfg domain.HTTPActionConfig
	if err := json.Unmarshal(step.Configuration, &cfg); err != nil {
		return StepResult{}, domain.PermanentExecutionError(err)
	}
	if err := cfg.Validate(); err != nil {
		return StepResult{}, domain.PermanentExecutionError(err)
	}

	reqCtx, cancel := context.WithTimeout(ctx, step.Timeout)
	defer cancel()

	var body io.Reader
	if cfg.Body != "" {
		body = strings.NewReader(cfg.Body)
	}
	req, err := http.NewRequestWithContext(reqCtx, strings.ToUpper(cfg.Method), cfg.URL, body)
	if err != nil {
		return StepResult{}, domain.PermanentExecutionError(err)
	}
	for k, v := range cfg.Headers {
		if isForbiddenHeader(k) {
			continue
		}
		req.Header.Set(k, v)
	}

	resp, err := e.client.Do(req)
	if err != nil {
		if reqCtx.Err() == context.DeadlineExceeded {
			return StepResult{}, domain.TimeoutExecutionError(err)
		}
		return StepResult{}, domain.TransientExecutionError(err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, domain.DefaultMaxInlineOutputBytes))
	inline := string(respBody)
	if len(inline) > domain.DefaultMaxInlineOutputBytes {
		inline = inline[:domain.DefaultMaxInlineOutputBytes]
	}

	if resp.StatusCode >= 500 {
		return StepResult{}, domain.TransientExecutionError(fmt.Errorf("http %d", resp.StatusCode))
	}
	if resp.StatusCode >= 400 {
		return StepResult{}, domain.PermanentExecutionError(fmt.Errorf("http %d", resp.StatusCode))
	}

	return StepResult{Output: &domain.OutputReference{
		Type: "inline", Location: "response", ContentType: resp.Header.Get("Content-Type"),
		Size: int64(len(respBody)), Inline: inline,
	}}, nil
}

func isForbiddenHeader(name string) bool {
	switch strings.ToLower(name) {
	case "authorization", "cookie", "set-cookie", "proxy-authorization":
		return true
	default:
		return false
	}
}

// DelayExecutor schedules DELAY steps without blocking.
type DelayExecutor struct{}

func NewDelayExecutor() *DelayExecutor { return &DelayExecutor{} }

func (e *DelayExecutor) Type() domain.StepType { return domain.StepTypeDelay }

func (e *DelayExecutor) Execute(ctx context.Context, step domain.WorkflowStepDefinition, execCtx StepContext) (StepResult, error) {
	var cfg domain.DelayConfig
	if err := json.Unmarshal(step.Configuration, &cfg); err != nil {
		return StepResult{}, domain.PermanentExecutionError(err)
	}
	if err := cfg.Validate(step.Timeout); err != nil {
		return StepResult{}, domain.PermanentExecutionError(err)
	}
	d, _ := time.ParseDuration(cfg.Duration)
	return StepResult{Output: &domain.OutputReference{
		Type: "delay", Location: d.String(), Inline: fmt.Sprintf("delayed %s", d),
	}}, nil
}
