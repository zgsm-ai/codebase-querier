package config

import "time"

// VectorStoreConf 向量数据库配置
type VectorStoreConf struct {
	Type string // 向量数据库类型
	// 通用配置
	Timeout    time.Duration // 操作超时时间
	MaxRetries int           // 最大重试次数
	Embedder   EmbedderConf
	Rerank     RerankConf
	Retriever  RetrieverConf
	// 具体实现配置
	Weaviate *WeaviateConf // Weaviate配置
}

// WeaviateConf Weaviate向量数据库配置
type WeaviateConf struct {
	Host      string // HTTP端点
	APIKey    string // API密钥
	IndexName string // 索引名称
	BatchSize int    // 批处理大小
	Namespace string //
	ClassName string
}

// EmbedderConf 嵌入模型配置
type EmbedderConf struct {
	// 通用配置
	Timeout       time.Duration
	MaxRetries    int
	BatchSize     int
	Model         string `json:"model" yaml:"model"`                         // 模型名称（如text-embedding-ada-002）
	APIKey        string `json:"apiKey" yaml:"apiKey"`                       // API密钥
	APIBase       string `json:"apiBase,omitempty" yaml:"apiBase,omitempty"` // API基础URL
	StripNewLines bool
}

type RetrieverConf struct {
	NumDocuments   int
	Namespace      string
	ScoreThreshold float32
}

// RerankConf 嵌入模型配置
type RerankConf struct {
	// 通用配置
	Timeout    time.Duration
	MaxRetries int
	BatchSize  int
	Model      string `json:"model" yaml:"model"`                         // 模型名称（如text-embedding-ada-002）
	APIKey     string `json:"apiKey" yaml:"apiKey"`                       // API密钥
	APIBase    string `json:"apiBase,omitempty" yaml:"apiBase,omitempty"` // API基础URL
}
