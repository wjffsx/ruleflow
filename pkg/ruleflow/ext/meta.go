package ext

// V13 架构边界说明：
//
// 本包提供通用扩展节点，需要外部依赖注入（Storage, Publisher 等）。
//
// 命名规范：
//   - _ext 后缀：区分 builtin 和 ext 版本的同名节点
//   - 例如：alarm_notify (builtin) vs alarm_notify_ext (ext)
//   - 后缀有助于用户理解节点依赖和功能差异
//
// 与 builtin 的区别：
//   - builtin: 无外部依赖，适合简单场景
//   - ext: 需外部依赖注入，适合完整功能场景
//
// 与 extensions 的区别：
//   - ext: 通用 IoT 扩展节点
//   - extensions: VPP 业务特定节点

import (
	"github.com/wjffsx/ruleflow/pkg/ruleflow/nodes"
)

// ActionMetaList returns metadata for all ext action nodes.
// Use this for visualization editors and documentation.
func ActionMetaList() []nodes.ComponentMeta {
	return []nodes.ComponentMeta{
		{
			Type: "alarm_notify_ext", Label: "扩展告警通知", LabelEn: "Alarm Notify (Ext)",
			Category: "route_check", Icon: "alert",
			Description: "扩展告警通知：生成告警记录并触发通知",
			Fields: []nodes.ComponentField{
				{Name: "alarmType", Label: "告警类型", Type: "string", Required: true, Default: "threshold"},
				{Name: "severity", Label: "严重级别", Type: "string", Required: false, Default: "warning"},
			},
		},
		{
			Type: "quality_mark_ext", Label: "扩展质量标记", LabelEn: "Quality Mark (Ext)",
			Category: "route_check", Icon: "checkmark",
			Description: "扩展质量标记：标记数据点质量",
			Fields: []nodes.ComponentField{
				{Name: "quality", Label: "质量值", Type: "string", Required: false, Default: "GOOD"},
			},
		},
		{
			Type: "calc_node", Label: "计算节点", LabelEn: "Calculator",
			Category: "data_process", Icon: "calc",
			Description: "计算节点：使用 expr-lang 执行公式计算",
			Fields: []nodes.ComponentField{
				{Name: "formula", Label: "计算公式", Type: "string", Required: true},
				{Name: "inputs", Label: "输入字段", Type: "array", Required: false},
				{Name: "output", Label: "输出字段", Type: "string", Required: false},
			},
		},
		{
			Type: "storage_write", Label: "存储写入", LabelEn: "Storage Write",
			Category: "data_process", Icon: "save",
			Description: "存储写入：将数据写入实时库",
			Fields: []nodes.ComponentField{
				{Name: "target", Label: "路由目标", Type: "string", Required: false},
			},
		},
		{
			Type: "aggregation_write", Label: "聚合写入", LabelEn: "Aggregation Write",
			Category: "data_process", Icon: "aggregate",
			Description: "聚合写入：批量写入聚合数据",
		},
		{
			Type: "device_aggregator", Label: "设备聚合", LabelEn: "Device Aggregator",
			Category: "data_process", Icon: "device",
			Description: "设备聚合：按设备分类聚合数据",
			Fields: []nodes.ComponentField{
				{Name: "input_point", Label: "输入点", Type: "string", Required: true},
				{Name: "output_mappings", Label: "输出映射", Type: "array", Required: true},
			},
		},
		{
			Type: "status_change_log", Label: "状态变更日志", LabelEn: "Status Change Log",
			Category: "route_check", Icon: "log",
			Description: "状态变更日志：记录数据点状态变更事件",
		},
		{
			Type: "expr_switch", Label: "表达式分支", LabelEn: "Expression Switch",
			Category: "route_check", Icon: "branch",
			Description: "表达式分支：根据表达式结果分流",
			Fields: []nodes.ComponentField{
				{Name: "expression", Label: "分支表达式", Type: "string", Required: true},
			},
		},
		{
			Type: "multi_device_control", Label: "多设备联动", LabelEn: "Multi Device Control",
			Category: "route_check", Icon: "devices",
			Description: "多设备联动：向多个设备发送控制命令",
			Fields: []nodes.ComponentField{
				{Name: "targets", Label: "目标设备列表", Type: "array", Required: true},
				{Name: "command", Label: "控制命令", Type: "string", Required: true},
				{Name: "params", Label: "控制参数", Type: "map", Required: false},
			},
		},
		{
			Type: "strategy_execute", Label: "策略执行", LabelEn: "Strategy Execute",
			Category: "route_check", Icon: "strategy",
			Description: "策略执行：调用指定策略",
			Fields: []nodes.ComponentField{
				{Name: "strategy_id", Label: "策略ID", Type: "string", Required: true},
				{Name: "params", Label: "策略参数", Type: "map", Required: false},
			},
		},
		// ── 新增告警检测节点 ──
		{
			Type: "alarm_emit", Label: "告警发射", LabelEn: "Alarm Emit",
			Category: "route_check", Icon: "alert",
			Description: "告警发射：在 DataContext 上设置告警 Tag，供后续 AlarmManager 消费",
			Fields: []nodes.ComponentField{
				{Name: "severity", Label: "严重级别", Type: "string", Required: false, Default: "warning"},
				{Name: "source_type", Label: "来源类型", Type: "string", Required: false, Default: "device"},
				{Name: "message_template", Label: "消息模板", Type: "string", Required: false},
			},
		},
	}
}

// ConditionMetaList returns metadata for all ext condition nodes.
func ConditionMetaList() []nodes.ComponentMeta {
	return []nodes.ComponentMeta{
		{
			Type: "threshold_detect", Label: "阈值检测", LabelEn: "Threshold Detect",
			Category: "route_check", Icon: "threshold",
			Description: "阈值检测：比较值与阈值，支持 gt/lt/gte/lte/eq/neq/outside_range/inside_range 和持续时间",
			Fields: []nodes.ComponentField{
				{Name: "operator", Label: "运算符", Type: "string", Required: true},
				{Name: "value", Label: "阈值", Type: "number", Required: false},
				{Name: "min_value", Label: "最小值(范围)", Type: "number", Required: false},
				{Name: "duration", Label: "持续时间", Type: "string", Required: false, Default: "0s"},
			},
		},
		{
			Type: "status_detect", Label: "状态检测", LabelEn: "Status Detect",
			Category: "route_check", Icon: "status",
			Description: "状态检测：检测数据点值是否匹配期望的状态值，支持去抖窗口",
			Fields: []nodes.ComponentField{
				{Name: "expected_value", Label: "期望值", Type: "string", Required: true},
				{Name: "debounce", Label: "去抖窗口", Type: "string", Required: false},
			},
		},
		{
			Type: "rate_detect", Label: "变化率检测", LabelEn: "Rate Detect",
			Category: "route_check", Icon: "speed",
			Description: "变化率检测：检测数据点值在时间窗口内的变化率是否超过阈值",
			Fields: []nodes.ComponentField{
				{Name: "max_rate", Label: "最大变化率", Type: "number", Required: true},
				{Name: "window", Label: "检测窗口", Type: "string", Required: false, Default: "60s"},
				{Name: "direction", Label: "方向", Type: "string", Required: false, Default: "both"},
			},
		},
		{
			Type: "composite_detect", Label: "复合条件检测", LabelEn: "Composite Detect",
			Category: "route_check", Icon: "composite",
			Description: "复合条件检测：AND/OR 逻辑组合多个子条件",
			Fields: []nodes.ComponentField{
				{Name: "logic", Label: "逻辑", Type: "string", Required: false, Default: "and"},
				{Name: "conditions", Label: "子条件列表", Type: "array", Required: true},
			},
		},
	}
}
