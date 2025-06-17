package main

import "time"

type Config struct {
	TimeoutConfig TimeoutConfig
	SizeConfig    SizeConfig
	ScaleConfig   ScaleConfig
}

var NewConfig = Config{
	TimeoutConfig: DefaultTimeouts,
	SizeConfig:    DefaultSizeConfig,
	ScaleConfig:   DefaultScaleConfig,
}

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

var DefaultTimeouts = TimeoutConfig{
	WorkerIdleTimeout:         30 * time.Second,
	WorkerStartupTimeout:      5 * time.Second,
	WorkerShutdownTimeout:     10 * time.Second,
	TaskDispatchTimeout:       5 * time.Second,
	TaskProcessingTimeout:     30 * time.Second,
	TaskRetryTimeout:          1 * time.Second,
	MetricsInterval:           1 * time.Second,
	HealthCheckInterval:       15 * time.Second,
	AvailabilityCheckInterval: 5 * time.Second,
}

// ----------------------------------------

type SizeConfig struct {
	WorkerPoolSize  int
	JobQueueSize    int
	TaskStreamSize  int
	MinWorkers      int
	MaxWorkers      int
	MaxJobPerWorker int32
}

var DefaultSizeConfig = SizeConfig{
	WorkerPoolSize:  10,
	JobQueueSize:    100000,
	TaskStreamSize:  100000, //change later
	MinWorkers:      5,
	MaxWorkers:      1000,
	MaxJobPerWorker: 500,
}

// ----------------------------------------

type ScaleConfig struct {
	ScaleUpQueueLengthThreshold   int
	ScaleDownQueueLengthThreshold int
	ScaleCheckInterval            time.Duration
}

var DefaultScaleConfig = ScaleConfig{
	ScaleUpQueueLengthThreshold:   100,
	ScaleDownQueueLengthThreshold: 50,
	ScaleCheckInterval:            5 * time.Second,
}
