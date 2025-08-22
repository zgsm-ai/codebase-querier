package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zgsm-ai/codebase-indexer/internal/config"
)

// SmartProxyHandler 智能代理处理器
// 根据请求头中的 X-Costrict-Version 字段和配置来决定转发策略：
// 1. 如果请求头里有 X-Costrict-Version 字段，则用 port_manager 转发
// 2. 否则如果配置了转发到固定地址，则转发到固定地址
// 3. 否则转发到 port_manager
type SmartProxyHandler struct {
	dynamicProxyHandler *DynamicProxyHandler
	staticProxyHandler  *ProxyHandler
	proxyConfig         *config.ProxyConfig
}

// NewSmartProxyHandler 创建智能代理处理器
func NewSmartProxyHandler(cfg *config.ProxyConfig) *SmartProxyHandler {
	handler := &SmartProxyHandler{
		dynamicProxyHandler: NewDynamicProxyHandler(cfg),
		proxyConfig:         cfg,
	}

	// 如果配置了 ForwardURL，创建静态代理处理器
	if cfg.ForwardURL != "" {
		staticConfig := &ProxyConfig{
			Mode: cfg.Mode,
			Target: TargetConfig{
				URL:     cfg.ForwardURL,
				Timeout: 30 * time.Second,
			},
			Rewrite: RewriteConfig{
				Enabled: cfg.Rewrite.Enabled,
				Rules:   make([]RewriteRule, len(cfg.Rewrite.Rules)),
			},
			Headers: HeadersConfig{
				PassThrough: cfg.Headers.PassThrough,
				Exclude:     cfg.Headers.Exclude,
				Override:    cfg.Headers.Override,
			},
		}

		// 复制重写规则
		for i, rule := range cfg.Rewrite.Rules {
			staticConfig.Rewrite.Rules[i] = RewriteRule{
				From: rule.From,
				To:   rule.To,
			}
		}

		handler.staticProxyHandler = NewProxyHandler(staticConfig)
		logx.Infof("Created static proxy handler for forward URL: %s", cfg.ForwardURL)
	}

	logx.Infof("Created smart proxy handler")
	return handler
}

// ServeHTTP 处理智能代理请求
func (h *SmartProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 检查请求头中是否有 X-Costrict-Version 字段
	costrictVersion := r.Header.Get("X-Costrict-Version")
	if costrictVersion != "" {
		logx.Infof("Request contains X-Costrict-Version header: %s, using dynamic proxy (port_manager)", costrictVersion)
		h.dynamicProxyHandler.ServeHTTP(w, r)
		return
	}

	// 如果没有 X-Costrict-Version 字段，检查是否配置了 ForwardURL
	if h.staticProxyHandler != nil {
		logx.Infof("No X-Costrict-Version header found, using static proxy to forward URL: %s", h.proxyConfig.ForwardURL)
		h.staticProxyHandler.ServeHTTP(w, r)
		return
	}

	// 否则使用 port_manager 转发
	logx.Infof("No X-Costrict-Version header and no forward URL configured, using dynamic proxy (port_manager)")
	h.dynamicProxyHandler.ServeHTTP(w, r)
}

// HealthCheck 健康检查
func (h *SmartProxyHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	type HealthStatus struct {
		DynamicProxy map[string]interface{} `json:"dynamic_proxy"`
		StaticProxy  map[string]interface{} `json:"static_proxy,omitempty"`
		ForwardURL   string                 `json:"forward_url,omitempty"`
		Strategy     string                 `json:"strategy"`
	}

	healthStatus := &HealthStatus{
		Strategy: "smart",
	}

	// 检查动态代理健康状态
	dynamicHealth := h.checkDynamicProxyHealth(r)
	healthStatus.DynamicProxy = dynamicHealth

	// 如果有静态代理，检查其健康状态
	if h.staticProxyHandler != nil {
		staticHealth := h.checkStaticProxyHealth(r)
		healthStatus.StaticProxy = staticHealth
		healthStatus.ForwardURL = h.proxyConfig.ForwardURL
	}

	response := map[string]interface{}{
		"status": "ok",
		"proxy":  healthStatus,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// checkDynamicProxyHealth 检查动态代理健康状态
func (h *SmartProxyHandler) checkDynamicProxyHealth(r *http.Request) map[string]interface{} {
	// 创建一个临时请求来测试动态代理
	tempReq, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		return map[string]interface{}{
			"healthy": false,
			"error":   fmt.Sprintf("Failed to create test request: %v", err),
		}
	}

	// 复制原始请求的头部，特别是 clientId
	tempReq.Header = r.Header.Clone()

	// 创建响应记录器
	recorder := &responseRecorder{
		statusCode: http.StatusOK,
		headers:    make(http.Header),
		body:       &bytes.Buffer{},
	}

	// 执行动态代理的健康检查
	h.dynamicProxyHandler.HealthCheck(recorder, tempReq)

	return map[string]interface{}{
		"healthy":     recorder.statusCode >= 200 && recorder.statusCode < 300,
		"status_code": recorder.statusCode,
		"response":    recorder.body.String(),
	}
}

// checkStaticProxyHealth 检查静态代理健康状态
func (h *SmartProxyHandler) checkStaticProxyHealth(r *http.Request) map[string]interface{} {
	ctx := r.Context()
	healthy, duration, err := h.staticProxyHandler.proxyLogic.HealthCheck(ctx)

	result := map[string]interface{}{
		"healthy":          healthy,
		"response_time_ms": duration.Milliseconds(),
	}

	if err != nil {
		result["error"] = err.Error()
	}

	return result
}

// Close 关闭处理器
func (h *SmartProxyHandler) Close() error {
	var errs []error

	// 关闭动态代理处理器
	if h.dynamicProxyHandler != nil {
		if err := h.dynamicProxyHandler.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close dynamic proxy handler: %w", err))
		}
	}

	// 关闭静态代理处理器
	if h.staticProxyHandler != nil {
		if err := h.staticProxyHandler.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close static proxy handler: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing smart proxy handlers: %v", errs)
	}

	return nil
}

// responseRecorder 用于记录响应的记录器
type responseRecorder struct {
	statusCode int
	headers    http.Header
	body       *bytes.Buffer
}

func (r *responseRecorder) Header() http.Header {
	return r.headers
}

func (r *responseRecorder) Write(data []byte) (int, error) {
	return r.body.Write(data)
}

func (r *responseRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
}
