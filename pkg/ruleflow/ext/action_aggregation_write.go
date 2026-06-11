package ext

import (
	"context"
	"time"

	"github.com/wjffsx/ruleflow/pkg/ruleflow/core"
)

// ─────────────────────────────────────────────
//  AggregationWriteAction — 聚合写入动作
// ─────────────────────────────────────────────

// AggregationWriteAction 批量聚合写入
type AggregationWriteAction struct {
	IDValue string
	Storage Storage
}

var _ core.Action = (*AggregationWriteAction)(nil)

func NewAggregationWriteAction(id string, storage Storage) *AggregationWriteAction {
	return &AggregationWriteAction{IDValue: id, Storage: storage}
}

func (a *AggregationWriteAction) Execute(_ context.Context, data core.DataContext) error {
	if a.Storage == nil {
		return nil
	}

	// 从 Tag（模拟 metadata）读取聚合数据
	// 约定：key_target 表示目标 deviceID, key 表示值, key_name 表示点名
	// 读取所有以 _target 结尾的 tag
	// 由于 DataContext 没有迭代 API，使用约定前缀
	ts := data.Timestamp()
	if ts == 0 {
		ts = time.Now().UnixMilli()
	}
	quality := data.Quality()

	// 使用 Raw() 获取底层数据点，尝试聚合读取
	raw := data.Raw()
	var dataPoints []any

	if dp, ok := raw.(interface{ GetGroup() string }); ok {
		// 通过 group 做聚合
		group := dp.GetGroup()
		if group != "" {
			dp := map[string]any{
				"device_id":  group,
				"point_name": data.PointName(),
				"value":      data.Value(),
				"timestamp":  ts,
				"quality":    quality,
			}
			dataPoints = append(dataPoints, dp)
		}
	}

	if len(dataPoints) == 0 {
		return nil
	}

	return a.Storage.WriteDataBatch(dataPoints)
}

func (a *AggregationWriteAction) ID() string          { return a.IDValue }
func (a *AggregationWriteAction) Type() string        { return "aggregation_write" }
func (a *AggregationWriteAction) Description() string { return "aggregation write" }
