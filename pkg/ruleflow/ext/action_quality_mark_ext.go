package ext

import (
	"context"
	"strconv"
	"strings"

	"github.com/vpptu/ruleflow/pkg/ruleflow/core"
)

// ─────────────────────────────────────────────
//  QualityMarkExtAction — 质量标记动作（扩展版）
// ─────────────────────────────────────────────

// V13 架构边界说明：
//   - 本节点是 ext 版本，需要外部依赖 DataQualityStorage
//   - 根据配置或 Tag 设置质量值，并同步写存储
//   - 适合完整质量管理场景
//
// 与 builtin 版本的区别：
//   - builtin: 根据越限状态设置质量，无外部依赖
//   - ext: 根据配置/Tag 设置质量，并同步写存储
//
// 使用场景：
//   - 完整质量管理：需要同步写存储
//   - 灵活质量配置：支持 Tag 动态设置质量

// DataQualityStorage 质量写回接口质量写回接口
type DataQualityStorage interface {
	UpdateDataQuality(deviceID, pointName string, quality int) error
}

// QualityMarkExtAction 标记数据点质量
type QualityMarkExtAction struct {
	IDValue string
	Quality string // 默认 "GOOD"
	Store   DataQualityStorage
}

var _ core.Action = (*QualityMarkExtAction)(nil)

func NewQualityMarkExtAction(id, quality string, store DataQualityStorage) *QualityMarkExtAction {
	return &QualityMarkExtAction{IDValue: id, Quality: quality, Store: store}
}

func (a *QualityMarkExtAction) Execute(_ context.Context, data core.DataContext) error {
	qualityStr := a.Quality
	if q := data.GetTag("quality"); q != "" {
		qualityStr = q
	}
	if qualityStr == "" {
		qualityStr = "GOOD"
	}

	// 设置 DataContext 质量值
	q := 0
	switch strings.ToUpper(qualityStr) {
	case "GOOD":
		q = 1
	case "BAD":
		q = 0
	default:
		if qi, err := strconv.Atoi(qualityStr); err == nil {
			q = qi
		}
	}
	data.SetQuality(q)
	data.SetTag("_vppt_quality", qualityStr)

	// 同步写存储（异步由调用方控制）
	if a.Store != nil {
		deviceID := data.DeviceID()
		pointName := data.PointName()
		if deviceID != "" && pointName != "" {
			_ = a.Store.UpdateDataQuality(deviceID, pointName, q)
		}
	}

	return nil
}

func (a *QualityMarkExtAction) ID() string          { return a.IDValue }
func (a *QualityMarkExtAction) Type() string        { return "quality_mark_ext" }
func (a *QualityMarkExtAction) Description() string { return "quality mark (ext)" }
