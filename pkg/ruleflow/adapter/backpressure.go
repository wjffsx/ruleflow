package adapter

import (
	"github.com/vpptu/ruleflow/pkg/ruleflow/core/contract"
)

// ─────────────────────────────────────────────
//  BackpressureAdapter — 背压适配器
// ─────────────────────────────────────────────

// BackpressureManager 是 BackpressureManager 的窄接口视图
type BackpressureManager interface {
	ShouldAccept(pointType string) bool
	CurrentWorstLevel() int
}

// BackpressureAdapter 将 BackpressureManager 适配为 ruleflow 的 BackpressureIndicator
type BackpressureAdapter struct {
	mgr       BackpressureManager
	pointType string // 按设备类型区分
}

// NewBackpressureAdapter 创建背压适配器
func NewBackpressureAdapter(mgr BackpressureManager, pointType string) *BackpressureAdapter {
	return &BackpressureAdapter{mgr: mgr, pointType: pointType}
}

// ShouldAccept 委托到背压管理器
func (a *BackpressureAdapter) ShouldAccept(deviceID string) bool {
	return a.mgr.ShouldAccept(a.pointType)
}

// CurrentLevel 将背压级别映射为 ruleflow 级别
func (a *BackpressureAdapter) CurrentLevel() contract.Level {
	// 级别：0=Normal, 1=Degraded, 2=Paused, 3=Dropping
	// 两者 1:1 映射
	return contract.Level(a.mgr.CurrentWorstLevel())
}
