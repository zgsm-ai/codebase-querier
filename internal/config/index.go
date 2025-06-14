package config

import "time"

type IndexTaskConf struct {
	Topic         string
	EmbeddingTask EmbeddingTaskConf
	GraphTask     GraphTaskConf
}

type EmbeddingTaskConf struct {
	MaxConcurrency int  `json:",default=5"`
	Enabled        bool `json:",default=true"`
	Timeout        time.Duration
	// 滑动窗口重叠token数
	OverlapTokens     int
	MaxTokensPerChunk int
	PoolSize          int
}

type GraphTaskConf struct {
	MaxConcurrency int  `json:",default=5"`
	Enabled        bool `json:",default=true"`
	Timeout        time.Duration
	PoolSize       int
	ConfFile       string `json:",default=etc/codegraph.yaml"`
}
