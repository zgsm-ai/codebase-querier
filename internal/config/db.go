package config

import "time"

type Database struct {
	Driver      string
	DataSource  string
	AutoMigrate struct {
		Enable bool
	}
}

// RedisConfig Redis配置
type RedisConfig struct {
	Addr           string
	Password       string        `json:",optional"`
	DB             int           `json:",default=0"`
	PoolSize       int           `json:",default=10"`
	MinIdleConn    int           `json:",default=10"`
	ConnectTimeout time.Duration `json:",default=10s"`
	ReadTimeout    time.Duration `json:",default=10s"`
	WriteTimeout   time.Duration `json:",default=10s"`
}
