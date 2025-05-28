package config

import "time"

type IndexJob struct {
	Topic      string
	PoolSize   int
	Timeout    time.Duration
	EnableFlag int
}
