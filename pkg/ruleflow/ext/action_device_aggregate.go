package ext

import (
	"context"
	"strconv"

	"github.com/wjffsx/ruleflow/pkg/ruleflow/core"
)

// ─────────────────────────────────────────────
//  DeviceAggregateAction — 设备聚合动作
// ─────────────────────────────────────────────

// CategoryProvider 设备分类提供者
type CategoryProvider interface {
	GetDeviceCategory(deviceID string) string
}

// OutputMapping 聚合输出映射
type OutputMapping struct {
	Category string `json:"category" yaml:"category"`
	Output   string `json:"output" yaml:"output"`
	Target   string `json:"target" yaml:"target"`
}

// RealtimeBatchReader 批量实时数据读取接口
type RealtimeBatchReader interface {
	GetRealtimeDataBatchMulti(deviceIDs []string, pointNames []string) ([]any, error)
	GetAllRealtimeData() ([]any, error)
}

// DeviceAggregateAction 设备级聚合动作
type DeviceAggregateAction struct {
	IDValue          string
	InputPoint       string
	OutputMappings   []OutputMapping
	CategoryProvider CategoryProvider
	BatchReader      RealtimeBatchReader
}

var _ core.Action = (*DeviceAggregateAction)(nil)

func NewDeviceAggregateAction(id, inputPoint string, mappings []OutputMapping, catProvider CategoryProvider, reader RealtimeBatchReader) *DeviceAggregateAction {
	return &DeviceAggregateAction{
		IDValue:          id,
		InputPoint:       inputPoint,
		OutputMappings:   mappings,
		CategoryProvider: catProvider,
		BatchReader:      reader,
	}
}

func (a *DeviceAggregateAction) Execute(_ context.Context, data core.DataContext) error {
	if a.InputPoint == "" || len(a.OutputMappings) == 0 || a.BatchReader == nil {
		return nil
	}

	type aggResult struct {
		Sum    float64
		Target string
	}
	results := make(map[string]*aggResult)
	categoryCache := make(map[string]string, 16)

	var allData []any
	var err error
	allData, err = a.BatchReader.GetRealtimeDataBatchMulti(nil, []string{a.InputPoint})
	if err != nil {
		allData, err = a.BatchReader.GetAllRealtimeData()
		if err != nil {
			return nil // 静默失败，不影响主流程
		}
	}

	for _, dpAny := range allData {
		if dpMap, ok := dpAny.(map[string]any); ok {
			pn, _ := dpMap["point_name"].(string)
			if pn != a.InputPoint {
				continue
			}
			did, _ := dpMap["device_id"].(string)

			category := "other"
			if a.CategoryProvider != nil {
				if cat, ok := categoryCache[did]; ok {
					category = cat
				} else if cat := a.CategoryProvider.GetDeviceCategory(did); cat != "" {
					categoryCache[did] = cat
					category = cat
				}
			}

			val, _ := dpMap["value"].(float64)
			for _, mapping := range a.OutputMappings {
				if mapping.Category != "*" && mapping.Category != category {
					continue
				}
				if results[mapping.Output] == nil {
					results[mapping.Output] = &aggResult{Target: mapping.Target}
				}
				results[mapping.Output].Sum += val
			}
		}
	}

	// 将聚合结果写入 DataContext Tags
	for name, r := range results {
		data.SetTag(name, strconv.FormatFloat(r.Sum, 'f', 2, 64))
		data.SetTag(name+"_target", r.Target)
	}

	return nil
}

func (a *DeviceAggregateAction) ID() string          { return a.IDValue }
func (a *DeviceAggregateAction) Type() string        { return "device_aggregator" }
func (a *DeviceAggregateAction) Description() string { return "device aggregator" }
