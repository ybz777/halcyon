package observability

import (
	"fmt"
	"time"

	platformconfig "github.com/halcyon-io/halcyon/pkg/platform/config"
)

// TracingConfig configures OpenTelemetry tracing.
type TracingConfig struct {
	Enabled         bool
	ServiceName     string
	ServiceVersion  string
	Environment     string
	InstanceID      string
	OTLPEndpoint    string
	OTLPProtocol    string
	Sampler         string
	SamplerArg      float64
	ResourceAttrs   map[string]string
	BatchTimeout    time.Duration
	BatchExportMax  int
	MaxQueueSize    int
	ShutdownTimeout time.Duration
}

// LoadTracingConfig reads tracing settings from environment variables.
func LoadTracingConfig(serviceName string) (TracingConfig, error) {
	return LoadTracingConfigFrom(platformconfig.FromEnv(), serviceName)
}

// LoadTracingConfigFrom reads tracing settings from a centralized env reader.
func LoadTracingConfigFrom(env *platformconfig.Env, serviceName string) (TracingConfig, error) {
	loaded, err := platformconfig.LoadObservability(env, serviceName)
	if err != nil {
		return TracingConfig{}, err
	}
	return TracingConfig{
		Enabled:         loaded.Enabled,
		ServiceName:     loaded.ServiceName,
		ServiceVersion:  loaded.ServiceVersion,
		Environment:     loaded.Environment,
		InstanceID:      loaded.InstanceID,
		OTLPEndpoint:    loaded.OTLPEndpoint,
		OTLPProtocol:    loaded.OTLPProtocol,
		Sampler:         loaded.Sampler,
		SamplerArg:      loaded.SamplerArg,
		BatchTimeout:    loaded.BatchTimeout,
		BatchExportMax:  loaded.BatchExportMax,
		MaxQueueSize:    loaded.MaxQueueSize,
		ShutdownTimeout: loaded.ShutdownTimeout,
		ResourceAttrs:   loaded.ResourceAttrs,
	}, nil
}

func (c TracingConfig) validate() error {
	switch c.Sampler {
	case "always_on", "always_off", "traceidratio", "parentbased_traceidratio",
		"parentbased_always_on", "parentbased_always_off":
	default:
		return fmt.Errorf("invalid OTEL_TRACES_SAMPLER %q", c.Sampler)
	}
	if c.SamplerArg < 0 || c.SamplerArg > 1 {
		return fmt.Errorf("OTEL_TRACES_SAMPLER_ARG must be between 0 and 1")
	}
	if c.Enabled {
		switch c.OTLPProtocol {
		case "http/protobuf", "grpc":
		default:
			return fmt.Errorf("invalid OTEL_EXPORTER_OTLP_PROTOCOL %q", c.OTLPProtocol)
		}
		if c.OTLPEndpoint == "" {
			return fmt.Errorf("OTEL_EXPORTER_OTLP_ENDPOINT is required when OTEL_ENABLED=true")
		}
	}
	return nil
}
