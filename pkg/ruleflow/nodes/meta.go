// Package nodes provides node metadata for visualization/editors.
//
// V7 Refactoring: Moved from registry/registry.go GetComponentMeta.
// Component metadata belongs to the nodes layer, not the core registry.
package nodes

// ComponentMeta component metadata for visualization editors
type ComponentMeta struct {
	Type        string           `json:"type"`
	Label       string           `json:"label"`
	LabelEn     string           `json:"label_en,omitempty"`
	Category    string           `json:"category"` // condition / action
	Icon        string           `json:"icon,omitempty"`
	Description string           `json:"description"`
	Fields      []ComponentField `json:"fields"`
}

// ComponentField component field definition
type ComponentField struct {
	Name     string   `json:"name"`
	Label    string   `json:"label"`
	Type     string   `json:"type"` // string / number / bool / array / map
	Required bool     `json:"required"`
	Default  string   `json:"default,omitempty"`
	Options  []Option `json:"options,omitempty"`
}

// Option enum option
type Option struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// GetBuiltinMeta returns metadata for all builtin nodes.
// Use this for visualization editors and documentation.
func GetBuiltinMeta() []ComponentMeta {
	return []ComponentMeta{
		// Condition components
		{Type: "device_type", Label: "设备类型", Category: "condition", Description: "按设备类型过滤", Fields: []ComponentField{
			{Name: "device_types", Label: "设备类型列表", Type: "array", Required: true},
		}},
		{Type: "point_name", Label: "数据点名", Category: "condition", Description: "按数据点名过滤", Fields: []ComponentField{
			{Name: "point_names", Label: "数据点名列表", Type: "array", Required: true},
		}},
		{Type: "point_name_pattern", Label: "数据点名模式", Category: "condition", Description: "按正则模式匹配数据点名", Fields: []ComponentField{
			{Name: "pattern", Label: "正则表达式", Type: "string", Required: true},
		}},
		{Type: "value_range", Label: "值范围", Category: "condition", Description: "按数值范围过滤", Fields: []ComponentField{
			{Name: "min_value", Label: "最小值", Type: "number"},
			{Name: "max_value", Label: "最大值", Type: "number"},
		}},
		{Type: "quality", Label: "质量", Category: "condition", Description: "按质量码过滤", Fields: []ComponentField{
			{Name: "min_quality", Label: "最低质量", Type: "number", Required: true},
		}},
		{Type: "limit_exceeded", Label: "越限", Category: "condition", Description: "越限状态条件", Fields: []ComponentField{}},
		{Type: "device_id", Label: "设备ID", Category: "condition", Description: "按设备ID过滤", Fields: []ComponentField{
			{Name: "device_ids", Label: "设备ID列表", Type: "array", Required: true},
		}},
		{Type: "fqn_prefix", Label: "FQN前缀", Category: "condition", Description: "按FQN前缀过滤", Fields: []ComponentField{
			{Name: "prefixes", Label: "前缀列表", Type: "array", Required: true},
		}},
		{Type: "time_window", Label: "时间窗口", Category: "condition", Description: "判断时间是否在指定窗口内", Fields: []ComponentField{
			{Name: "start", Label: "起始时间", Type: "string", Required: true},
			{Name: "end", Label: "结束时间", Type: "string", Required: true},
			{Name: "timezone", Label: "时区", Type: "string"},
			{Name: "weekdays", Label: "星期", Type: "array"},
		}},
		{Type: "value_in", Label: "离散值匹配", Category: "condition", Description: "判断值是否在离散值列表中", Fields: []ComponentField{
			{Name: "values", Label: "值列表", Type: "array", Required: true},
		}},
		{Type: "state_change", Label: "状态变化", Category: "condition", Description: "检测数据点值的变化", Fields: []ComponentField{
			{Name: "from", Label: "旧值", Type: "number"},
			{Name: "to", Label: "新值", Type: "number"},
		}},
		{Type: "duration", Label: "持续时间", Category: "condition", Description: "条件连续满足指定时长", Fields: []ComponentField{
			{Name: "duration", Label: "持续时间", Type: "string", Required: true},
		}},
		{Type: "trend", Label: "趋势判断", Category: "condition", Description: "判断值在时间窗口内的变化趋势", Fields: []ComponentField{
			{Name: "direction", Label: "方向", Type: "string", Required: true, Options: []Option{
				{Value: "increasing", Label: "上升"},
				{Value: "decreasing", Label: "下降"},
			}},
			{Name: "window", Label: "时间窗口", Type: "string", Required: true},
			{Name: "threshold", Label: "变化阈值", Type: "number", Required: true},
		}},
		{Type: "periodic", Label: "周期判断", Category: "condition", Description: "条件在连续N个周期内满足", Fields: []ComponentField{
			{Name: "period", Label: "周期", Type: "string", Required: true},
			{Name: "count", Label: "连续次数", Type: "number", Required: true},
		}},
		{Type: "dynamic_threshold", Label: "动态阈值", Category: "condition", Description: "从Tag读取阈值进行比较", Fields: []ComponentField{
			{Name: "operator", Label: "比较运算符", Type: "string", Required: true, Options: []Option{
				{Value: "gt", Label: ">"},
				{Value: "lt", Label: "<"},
				{Value: "gte", Label: ">="},
				{Value: "lte", Label: "<="},
				{Value: "eq", Label: "=="},
				{Value: "neq", Label: "!="},
			}},
			{Name: "source", Label: "阈值来源", Type: "string", Required: true},
		}},
		// Action components
		{Type: "transform", Label: "值变换", Category: "action", Description: "缩放/偏移/单位转换", Fields: []ComponentField{
			{Name: "scale", Label: "缩放系数", Type: "number"},
			{Name: "offset", Label: "偏移量", Type: "number"},
			{Name: "unit", Label: "目标单位", Type: "string"},
		}},
		{Type: "rename", Label: "重命名", Category: "action", Description: "重命名数据点", Fields: []ComponentField{
			{Name: "point_name", Label: "新名称", Type: "string", Required: true},
		}},
		{Type: "tag", Label: "标签", Category: "action", Description: "添加标签", Fields: []ComponentField{
			{Name: "tags", Label: "标签键值对", Type: "map", Required: true},
		}},
		{Type: "drop", Label: "丢弃", Category: "action", Description: "丢弃数据点", Fields: []ComponentField{}},
		{Type: "route", Label: "路由", Category: "action", Description: "路由到目标通道", Fields: []ComponentField{
			{Name: "targets", Label: "目标列表", Type: "array", Required: true},
		}},
		{Type: "limit_check", Label: "越限检测", Category: "action", Description: "检测是否超过上下限", Fields: []ComponentField{}},
		{Type: "quality_mark", Label: "质量标记", Category: "action", Description: "根据越限结果设置质量码", Fields: []ComponentField{
			{Name: "good_value", Label: "正常质量码", Type: "number", Required: true},
			{Name: "bad_value", Label: "异常质量码", Type: "number", Required: true},
		}},
		{Type: "alarm_notify", Label: "告警通知", Category: "action", Description: "发送告警通知", Fields: []ComponentField{
			{Name: "severity", Label: "严重级别", Type: "string", Required: true, Options: []Option{
				{Value: "info", Label: "信息"},
				{Value: "warning", Label: "警告"},
				{Value: "critical", Label: "严重"},
			}},
			{Name: "message", Label: "告警消息", Type: "string", Required: true},
		}},
		{Type: "delay", Label: "延时执行", Category: "action", Description: "延时执行动作", Fields: []ComponentField{
			{Name: "duration", Label: "延时时间", Type: "string", Required: true},
		}},
	}
}
