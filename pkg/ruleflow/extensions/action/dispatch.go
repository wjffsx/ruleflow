// Package action provides VPP action nodes
//
// V7 Refactoring: Updated to use vpp/types instead of core/vpp.
package action

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/wjffsx/ruleflow/pkg/ruleflow/core"
	"github.com/wjffsx/ruleflow/pkg/ruleflow/extensions/types"
)

// ─────────────────────────────────────────────
//  DispatchControlAction — VPP 调度指令下发动作
// ─────────────────────────────────────────────

// DispatchControlAction VPP 调度指令下发动作
type DispatchControlAction struct {
	IDValue    string  `json:"id"`
	Target     string  `json:"target"`      // 目标设备或设备组 ID
	Command    string  `json:"command"`     // 指令类型
	Param      float64 `json:"param"`       // 指令参数
	Protocol   string  `json:"protocol"`    // 通信协议
	TimeoutMs  int     `json:"timeout_ms"`  // 指令超时（毫秒）
	RetryCount int     `json:"retry_count"` // 重试次数

	// 外部依赖：通过适配器注入
	Dispatcher types.CommandDispatcher
}

// NewDispatchControlAction 创建调度指令下发动作
func NewDispatchControlAction(id, target, command string, param float64, protocol string, timeoutMs, retryCount int, dispatcher types.CommandDispatcher) *DispatchControlAction {
	if protocol == "" {
		protocol = "http"
	}
	if timeoutMs == 0 {
		timeoutMs = 5000
	}
	return &DispatchControlAction{
		IDValue:    id,
		Target:     target,
		Command:    command,
		Param:      param,
		Protocol:   protocol,
		TimeoutMs:  timeoutMs,
		RetryCount: retryCount,
		Dispatcher: dispatcher,
	}
}

func (a *DispatchControlAction) Execute(ctx context.Context, data core.DataContext) error {
	if a.Dispatcher == nil {
		return fmt.Errorf("dispatch_control: no CommandDispatcher configured")
	}

	target := a.Target
	// 支持从 DataContext Tag 动态解析目标
	if dynamicTarget := data.GetTag("_dispatch_target"); dynamicTarget != "" {
		target = dynamicTarget
	}

	timeout := time.Duration(a.TimeoutMs) * time.Millisecond

	var lastErr error
	retries := a.RetryCount
	if retries <= 0 {
		retries = 1
	}

	for i := 0; i < retries; i++ {
		err := a.Dispatcher.Dispatch(ctx, target, a.Command, a.Param, a.Protocol, timeout)
		if err == nil {
			data.SetTag("_dispatch_status", "ok")
			data.SetTag("_dispatch_target", target)
			return nil
		}
		lastErr = err
		select {
		case <-time.After(100 * time.Millisecond):
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	data.SetTag("_dispatch_status", "failed")
	return fmt.Errorf("dispatch_control: dispatch to %s failed after %d retries: %w", target, a.RetryCount, lastErr)
}

func (a *DispatchControlAction) ID() string   { return a.IDValue }
func (a *DispatchControlAction) Type() string { return "dispatch_control" }
func (a *DispatchControlAction) Description() string {
	return fmt.Sprintf("dispatch %s to %s", a.Command, a.Target)
}

// ─────────────────────────────────────────────
//  HTTPCommandDispatcher — HTTP 指令下发适配器
// ─────────────────────────────────────────────

// HTTPCommandDispatcher HTTP 指令下发适配器
type HTTPCommandDispatcher struct {
	BaseURL   string
	AuthToken string
	Client    *http.Client
}

// NewHTTPCommandDispatcher 创建 HTTP 指令下发适配器
func NewHTTPCommandDispatcher(baseURL, authToken string, client *http.Client) *HTTPCommandDispatcher {
	if client == nil {
		client = http.DefaultClient
	}
	return &HTTPCommandDispatcher{
		BaseURL:   baseURL,
		AuthToken: authToken,
		Client:    client,
	}
}

func (d *HTTPCommandDispatcher) Dispatch(ctx context.Context, target, command string, param float64, protocol string, timeout time.Duration) error {
	payload := map[string]any{
		"target":  target,
		"command": command,
		"param":   param,
	}
	body, _ := json.Marshal(payload)

	url := d.BaseURL + "/api/v1/dispatch"
	req, _ := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+d.AuthToken)

	resp, err := d.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("dispatch HTTP %d", resp.StatusCode)
	}
	return nil
}
