package ext

import (
	"context"
	"time"

	"github.com/wjffsx/ruleflow/pkg/ruleflow/core"
)

// ─────────────────────────────────────────────
//  HistoricalCompareCondition — 历史对比条件
// ─────────────────────────────────────────────

// HistoricalReader 历史数据读取接口（由服务层实现）
type HistoricalReader interface {
	Read(deviceID, pointName string, ago time.Duration) (float64, bool, error)
}

// HistoricalCompareCondition 历史对比条件
type HistoricalCompareCondition struct {
	IDValue   string
	Ago       string  // "1h" / "24h" / "7d"
	Operator  string  // "gt" / "lt" / "gte" / "lte"
	Threshold float64 // 变化百分比阈值
	Reader    HistoricalReader
}

var _ core.Condition = (*HistoricalCompareCondition)(nil)

func NewHistoricalCompareCondition(id, ago, operator string, threshold float64, reader HistoricalReader) *HistoricalCompareCondition {
	return &HistoricalCompareCondition{
		IDValue:   id,
		Ago:       ago,
		Operator:  operator,
		Threshold: threshold,
		Reader:    reader,
	}
}

func (c *HistoricalCompareCondition) Evaluate(_ context.Context, data core.DataContext) bool {
	if c.Reader == nil {
		return false
	}

	duration, err := time.ParseDuration(c.Ago)
	if err != nil {
		return false
	}

	histVal, ok, err := c.Reader.Read(data.DeviceID(), data.PointName(), duration)
	if !ok || err != nil {
		return false
	}

	if histVal == 0 {
		return false
	}
	changeRate := (data.Value() - histVal) / absVal(histVal)

	switch c.Operator {
	case "gt":
		return changeRate > c.Threshold
	case "lt":
		return changeRate < c.Threshold
	case "gte":
		return changeRate >= c.Threshold
	case "lte":
		return changeRate <= c.Threshold
	default:
		return false
	}
}

func (c *HistoricalCompareCondition) ID() string          { return c.IDValue }
func (c *HistoricalCompareCondition) Type() string        { return "historical_compare" }
func (c *HistoricalCompareCondition) Description() string { return "historical compare" }

func absVal(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
