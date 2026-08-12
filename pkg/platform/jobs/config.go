package jobs

import (
	platformconfig "github.com/halcyon-io/halcyon/pkg/platform/config"
	"time"
)

// Config holds worker and retry configuration loaded from environment.
type Config struct {
	Enabled              bool
	WorkerCount          int
	BatchSize            int
	LeaseDuration        time.Duration
	LeaseRenewInterval   time.Duration
	PollInterval         time.Duration
	ShutdownTimeout      time.Duration
	MaxConcurrent        int
	MaxAttempts          int
	ExecutionTimeout     time.Duration
	Retry                RetryConfig
	SuccessRetentionDays int
	FailureRetentionDays int
}

// LoadConfig reads job worker configuration from environment variables.
func LoadConfig() (Config, error) {
	return LoadConfigFrom(platformconfig.FromEnv())
}

// LoadConfigFrom reads job configuration from a centralized env reader.
func LoadConfigFrom(env *platformconfig.Env) (Config, error) {
	loaded, err := platformconfig.LoadJobs(env)
	if err != nil {
		return Config{}, err
	}
	return Config{
		Enabled:              loaded.Enabled,
		WorkerCount:          loaded.WorkerCount,
		BatchSize:            loaded.BatchSize,
		LeaseDuration:        loaded.LeaseDuration,
		LeaseRenewInterval:   loaded.LeaseRenewInterval,
		PollInterval:         loaded.PollInterval,
		ShutdownTimeout:      loaded.ShutdownTimeout,
		MaxConcurrent:        loaded.MaxConcurrent,
		MaxAttempts:          loaded.MaxAttempts,
		ExecutionTimeout:     loaded.ExecutionTimeout,
		Retry: RetryConfig{
			Initial:    loaded.RetryBaseDelay,
			MaxBackoff: loaded.RetryMaxDelay,
			Multiplier: loaded.RetryMultiplier,
			Jitter:     loaded.RetryJitter,
		},
		SuccessRetentionDays: loaded.SuccessRetentionDays,
		FailureRetentionDays: loaded.FailureRetentionDays,
	}, nil
}

// Validate ensures configuration values are safe and bounded.
func (c Config) Validate() error {
	return platformconfig.Jobs{
		Enabled: c.Enabled, WorkerCount: c.WorkerCount, BatchSize: c.BatchSize,
		LeaseDuration: c.LeaseDuration, LeaseRenewInterval: c.LeaseRenewInterval,
		PollInterval: c.PollInterval, ShutdownTimeout: c.ShutdownTimeout,
		MaxConcurrent: c.MaxConcurrent, MaxAttempts: c.MaxAttempts,
		ExecutionTimeout: c.ExecutionTimeout,
		RetryBaseDelay: c.Retry.Initial, RetryMaxDelay: c.Retry.MaxBackoff,
		RetryMultiplier: c.Retry.Multiplier, RetryJitter: c.Retry.Jitter,
		SuccessRetentionDays: c.SuccessRetentionDays, FailureRetentionDays: c.FailureRetentionDays,
	}.Validate()
}

// PoolConfigFrom builds a PoolConfig from Config and runtime options.
func (c Config) PoolConfigFrom(workerID string, jobTypes []Type) PoolConfig {
	return PoolConfig{
		WorkerID:          workerID,
		PollInterval:      c.PollInterval,
		LeaseDuration:     c.LeaseDuration,
		HeartbeatInterval: c.LeaseRenewInterval,
		ShutdownTimeout:   c.ShutdownTimeout,
		MaxConcurrent:     c.MaxConcurrent,
		ClaimBatchSize:    c.BatchSize,
		ExecutionTimeout:  c.ExecutionTimeout,
		RecoveryInterval:  c.LeaseDuration,
		Retry:             c.Retry,
		JobTypes:          jobTypes,
	}
}
