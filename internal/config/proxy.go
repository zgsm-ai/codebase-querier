package config

import (
	"errors"
	"fmt"
	"net/url"
	"time"
)

// ProxyConfig 代理配置
type ProxyConfig struct {
	Mode    string        `json:"mode" yaml:"mode"`     // 代理模式: rewrite, full_path
	Routes  []RouteConfig `json:"routes" yaml:"routes"` // 路由规则数组
	Rewrite RewriteConfig `json:"rewrite" yaml:"rewrite"`
	Headers HeadersConfig `json:"headers" yaml:"headers"`
}

// RouteConfig 路由配置
type RouteConfig struct {
	PathPrefix string       `json:"path_prefix" yaml:"path_prefix"` // 路径前缀
	Target     TargetConfig `json:"target" yaml:"target"`           // 目标服务配置
}

// TargetConfig 目标服务配置
type TargetConfig struct {
	URL     string        `json:"url" yaml:"url"`
	Timeout time.Duration `json:"timeout" yaml:"timeout"`
}

// UnmarshalYAML 自定义YAML解析方法，支持直接解析时间字符串（如"30s"）
func (t *TargetConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var aux struct {
		URL     string `json:"url" yaml:"url"`
		Timeout string `json:"timeout" yaml:"timeout"`
	}
	if err := unmarshal(&aux); err != nil {
		return err
	}

	t.URL = aux.URL

	// 如果timeout是字符串格式（如"30s"），则解析为time.Duration
	if aux.Timeout != "" {
		d, err := time.ParseDuration(aux.Timeout)
		if err != nil {
			return fmt.Errorf("invalid timeout format: %v", err)
		}
		t.Timeout = d
	} else {
		// 默认超时时间
		t.Timeout = 30 * time.Second
	}

	return nil
}

// RewriteConfig 路径重写配置
type RewriteConfig struct {
	Enabled bool          `json:"enabled" yaml:"enabled"`
	Rules   []RewriteRule `json:"rules" yaml:"rules"`
}

// RewriteRule 重写规则
type RewriteRule struct {
	From string `json:"from" yaml:"from"`
	To   string `json:"to" yaml:"to"`
}

// HeadersConfig Header配置
type HeadersConfig struct {
	PassThrough bool              `json:"pass_through" yaml:"pass_through"`
	Exclude     []string          `json:"exclude" yaml:"exclude"`
	Override    map[string]string `json:"override" yaml:"override"`
}

// 代理模式常量
const (
	ProxyModeRewrite  = "rewrite"   // 路径重写模式
	ProxyModeFullPath = "full_path" // 全路径转发模式
)

// Validate 验证代理配置
func (c *ProxyConfig) Validate() error {
	if c.Mode == "" {
		c.Mode = ProxyModeRewrite // 默认为rewrite模式，保持向后兼容
	}

	if c.Mode != ProxyModeRewrite && c.Mode != ProxyModeFullPath {
		return fmt.Errorf("invalid proxy mode: %s, must be %s or %s", c.Mode, ProxyModeRewrite, ProxyModeFullPath)
	}

	// 验证路由配置
	if len(c.Routes) == 0 {
		return errors.New("at least one route is required")
	}

	for i, route := range c.Routes {
		if route.PathPrefix == "" {
			return fmt.Errorf("route[%d] path_prefix is required", i)
		}
		if route.Target.URL == "" {
			return fmt.Errorf("route[%d] target URL is required", i)
		}
		if _, err := url.Parse(route.Target.URL); err != nil {
			return fmt.Errorf("route[%d] invalid target URL: %w", i, err)
		}
		if route.Target.Timeout <= 0 {
			c.Routes[i].Target.Timeout = 30 * time.Second
		}
	}

	// full_path模式下禁用rewrite
	if c.Mode == ProxyModeFullPath {
		c.Rewrite.Enabled = false
	}

	for _, rule := range c.Rewrite.Rules {
		if rule.From == "" {
			return errors.New("rewrite rule 'from' cannot be empty")
		}
	}

	return nil
}

// DefaultProxyConfig 返回默认配置
func DefaultProxyConfig() *ProxyConfig {
	return &ProxyConfig{
		Mode: ProxyModeRewrite, // 默认为rewrite模式，保持向后兼容
		Routes: []RouteConfig{
			{
				PathPrefix: "/",
				Target: TargetConfig{
					Timeout: 30 * time.Second,
				},
			},
		},
		Rewrite: RewriteConfig{
			Enabled: false,
		},
		Headers: HeadersConfig{
			PassThrough: true,
			Exclude:     []string{},
			Override:    make(map[string]string),
		},
	}
}
