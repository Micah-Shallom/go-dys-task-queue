package main

import "time"

type TimeoutConfig struct {
	// Worker timeouts
	WorkerIdleTimeout     time.Duration // 30 seconds
	WorkerStartupTimeout  time.Duration // 5 seconds
	WorkerShutdownTimeout time.Duration // 10 seconds

	// Task processing timeouts
	TaskDispatchTimeout   time.Duration // 5 seconds
	TaskProcessingTimeout time.Duration // 30 seconds
	TaskRetryTimeout      time.Duration // 1 second

	// System timeouts
	MetricsInterval           time.Duration // 5 seconds
	HealthCheckInterval       time.Duration // 15 seconds
	AvailabilityCheckInterval time.Duration // 5 seconds
}

var DefaultTimeouts = TimeoutConfig{
	WorkerIdleTimeout:         30 * time.Second,
	WorkerStartupTimeout:      5 * time.Second,
	WorkerShutdownTimeout:     10 * time.Second,
	TaskDispatchTimeout:       5 * time.Second,
	TaskProcessingTimeout:     30 * time.Second,
	TaskRetryTimeout:          1 * time.Second,
	MetricsInterval:           5 * time.Second,
	HealthCheckInterval:       15 * time.Second,
	AvailabilityCheckInterval: 5 * time.Second,
}
