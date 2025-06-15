package main

import "time"

type TimeoutConfig struct {
	WorkerIdleTimeout         time.Duration
	WorkerStartupTimeout      time.Duration
	WorkerShutdownTimeout     time.Duration
	TaskDispatchTimeout       time.Duration
	TaskProcessingTimeout     time.Duration
	TaskRetryTimeout          time.Duration
	MetricsInterval           time.Duration
	HealthCheckInterval       time.Duration
	AvailabilityCheckInterval time.Duration
}

type SizeConfig struct {
	WorkerPoolSize int
	JobQueueSize   int
	TaskStreamSize int
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

var DefaultSizeConfig = SizeConfig{
	WorkerPoolSize: 10,
	JobQueueSize:   5000,
	TaskStreamSize: 1000,
}
