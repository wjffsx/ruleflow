package config

import (
	"fmt"
	"os"

	"github.com/wjffsx/ruleflow/pkg/ruleflow/core"
	"github.com/wjffsx/ruleflow/pkg/ruleflow/nodes"
	"gopkg.in/yaml.v3"
)

// ChainConfig YAML/JSON 规则链配置格式
type ChainConfig struct {
	Chain ChainMeta    `yaml:"chain" json:"chain"`
	Rules []RuleConfig `yaml:"rules" json:"rules"`
}

// ChainMeta 规则链元信息
type ChainMeta struct {
	ID          string `yaml:"id" json:"id"`
	Name        string `yaml:"name" json:"name"`
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
	Root        bool   `yaml:"root" json:"root"`
	Version     int    `yaml:"version" json:"version"`
	Status      string `yaml:"status" json:"status"`
	// ★ Phase 1 新增
	PipelineType string                 `yaml:"pipeline_type,omitempty" json:"pipeline_type,omitempty"`
	Inputs       []InputConfig          `yaml:"inputs,omitempty" json:"inputs,omitempty"`
	Outputs      []core.RuleChainOutput `yaml:"outputs,omitempty" json:"outputs,omitempty"`
}

// ★ Phase 1 新增: InputConfig 输入配置
type InputConfig struct {
	PointName   string `yaml:"point_name" json:"point_name"`
	DisplayName string `yaml:"display_name,omitempty" json:"display_name,omitempty"`
	PointType   string `yaml:"point_type" json:"point_type"`
	DataType    string `yaml:"data_type" json:"data_type"`
	Unit        string `yaml:"unit,omitempty" json:"unit,omitempty"`
	Group       string `yaml:"group,omitempty" json:"group,omitempty"`
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
}

// RuleConfig 规则配置
type RuleConfig struct {
	ID          string `yaml:"id" json:"id"`
	Name        string `yaml:"name" json:"name"`
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
	Priority    int    `yaml:"priority" json:"priority"`
	Enabled     bool   `yaml:"enabled" json:"enabled"`
	// ★ Phase 1 新增
	InputBindings []string            `yaml:"input_bindings,omitempty" json:"input_bindings,omitempty"`
	InputMode     string              `yaml:"input_mode,omitempty" json:"input_mode,omitempty"`
	Condition     ConditionNodeConfig `yaml:"condition" json:"condition"`
	Actions       []ActionConfig      `yaml:"actions" json:"actions"`
	Targets       []string            `yaml:"targets,omitempty" json:"targets,omitempty"`
}

// ConditionNodeConfig 条件节点配置
type ConditionNodeConfig struct {
	ID         string                `yaml:"id,omitempty" json:"id,omitempty"`
	Operator   string                `yaml:"operator,omitempty" json:"operator,omitempty"`
	Children   []ConditionNodeConfig `yaml:"children,omitempty" json:"children,omitempty"`
	LeafType   string                `yaml:"leaf_type,omitempty" json:"leaf_type,omitempty"`
	LeafConfig map[string]any        `yaml:"leaf_config,omitempty" json:"leaf_config,omitempty"`
}

// ActionConfig 动作配置
type ActionConfig struct {
	Type   string         `yaml:"type" json:"type"`
	Config map[string]any `yaml:"config,omitempty" json:"config,omitempty"`
}

// LoadFromFile 从 YAML/JSON 文件加载规则链配置
func LoadFromFile(path string) (*ChainConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file %s: %w", path, err)
	}

	var cfg ChainConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	return &cfg, nil
}

// LoadFromBytes 从字节加载规则链配置
func LoadFromBytes(data []byte) (*ChainConfig, error) {
	var cfg ChainConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return &cfg, nil
}

// Parse 将配置解析为 RuleChain（需要 Registry 创建条件/动作实例）
//
// 内部实现两阶段解析：
//  1. ParseToIntermediate — 仅做字段映射，零依赖
//  2. parseIntermediate — 应用层调用 registry 创建实例
//
// 设计动机（V7 §5.4.1）：解耦 config 与 registry 的强耦合，config 层不再依赖业务 registry。
// Parse 仍接受 registry 以保持向后兼容；推荐新代码直接调用 ParseToIntermediate + 自行创建实例。
func Parse(cfg *ChainConfig, reg *nodes.Registry) (*core.RuleChain, error) {
	inter, err := ParseToIntermediate(cfg)
	if err != nil {
		return nil, err
	}
	return parseIntermediate(inter, reg)
}

// IntermediateChain 是两阶段解析的中间表示（不包含 Condition/Action 实例）。
// 应用层可基于此自行决定如何创建节点实例。
type IntermediateChain struct {
	ID           string
	Name         string
	Description  string
	Root         bool
	Version      int
	Status       string
	PipelineType string
	Inputs       []core.RuleChainInput
	Outputs      []core.RuleChainOutput
	Rules        []IntermediateRule
}

// IntermediateRule 规则的中间表示。
type IntermediateRule struct {
	ID            string
	Name          string
	Description   string
	Priority      int
	Enabled       bool
	InputBindings []string
	InputMode     string
	Condition     ConditionNodeConfig
	Actions       []ActionConfig
	Targets       []string
}

// ParseToIntermediate 第一阶段：仅做字段映射，不创建实例。
//
// 适用场景：
//   - 仅需展示/校验配置（如编辑器的预览）
//   - 应用层希望自行决定 Condition/Action 实例化策略
//   - 单元测试 config 解析逻辑（不依赖 registry）
//
// 此函数不依赖 registry，可在 config 层独立调用。
func ParseToIntermediate(cfg *ChainConfig) (*IntermediateChain, error) {
	if err := Validate(cfg); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	var inputs []core.RuleChainInput
	for _, inp := range cfg.Chain.Inputs {
		inputs = append(inputs, core.RuleChainInput{
			PointName:   inp.PointName,
			DisplayName: inp.DisplayName,
			PointType:   inp.PointType,
			DataType:    inp.DataType,
			Unit:        inp.Unit,
			Group:       inp.Group,
			Description: inp.Description,
		})
	}

	inter := &IntermediateChain{
		ID:           cfg.Chain.ID,
		Name:         cfg.Chain.Name,
		Description:  cfg.Chain.Description,
		Root:         cfg.Chain.Root,
		Version:      cfg.Chain.Version,
		Status:       cfg.Chain.Status,
		PipelineType: cfg.Chain.PipelineType,
		Inputs:       inputs,
		Outputs:      cfg.Chain.Outputs,
	}

	for _, rc := range cfg.Rules {
		inter.Rules = append(inter.Rules, IntermediateRule{
			ID:            rc.ID,
			Name:          rc.Name,
			Description:   rc.Description,
			Priority:      rc.Priority,
			Enabled:       rc.Enabled,
			InputBindings: rc.InputBindings,
			InputMode:     rc.InputMode,
			Condition:     rc.Condition,
			Actions:       rc.Actions,
			Targets:       rc.Targets,
		})
	}

	return inter, nil
}

// parseIntermediate 第二阶段：从中间表示创建 RuleChain（含实例化）。
func parseIntermediate(inter *IntermediateChain, reg *nodes.Registry) (*core.RuleChain, error) {
	chain := &core.RuleChain{
		ID:           inter.ID,
		Name:         inter.Name,
		Description:  inter.Description,
		Root:         inter.Root,
		Version:      inter.Version,
		Status:       inter.Status,
		PipelineType: inter.PipelineType,
		Inputs:       inter.Inputs,
		Outputs:      inter.Outputs,
	}

	for _, ir := range inter.Rules {
		rule, err := materializeRule(ir, reg)
		if err != nil {
			return nil, fmt.Errorf("materialize rule %s: %w", ir.ID, err)
		}
		chain.Rules = append(chain.Rules, rule)
	}

	return chain, nil
}

// materializeRule 从 IntermediateRule 创建 Rule（含实例化条件/动作）。
func materializeRule(ir IntermediateRule, reg *nodes.Registry) (*core.Rule, error) {
	condition, err := materializeCondition(ir.Condition, reg)
	if err != nil {
		return nil, fmt.Errorf("materialize condition: %w", err)
	}

	var actions []core.Action
	for i, ac := range ir.Actions {
		action, err := reg.CreateAction(ac.Type, fmt.Sprintf("%s_action_%d", ir.ID, i), ac.Config)
		if err != nil {
			return nil, fmt.Errorf("create action %s: %w", ac.Type, err)
		}
		actions = append(actions, action)
	}

	enabled := ir.Enabled
	if !ir.Enabled && ir.Priority == 0 && ir.ID == "" {
		// 默认启用
		enabled = true
	}

	return &core.Rule{
		ID:            ir.ID,
		Name:          ir.Name,
		Description:   ir.Description,
		Priority:      ir.Priority,
		Enabled:       enabled,
		InputBindings: ir.InputBindings,
		InputMode:     ir.InputMode,
		Condition:     condition,
		Actions:       &core.ActionChain{Actions: actions},
		Targets:       ir.Targets,
	}, nil
}

// materializeCondition 递归实例化条件节点。
func materializeCondition(cnc ConditionNodeConfig, reg *nodes.Registry) (*core.ConditionNode, error) {
	node := &core.ConditionNode{
		ID:         cnc.ID,
		LeafType:   cnc.LeafType,
		LeafConfig: cnc.LeafConfig,
	}

	// 叶节点：创建条件实例
	if cnc.LeafType != "" {
		condition, err := reg.CreateCondition(cnc.LeafType, cnc.ID, cnc.LeafConfig)
		if err != nil {
			return nil, fmt.Errorf("create condition %s: %w", cnc.LeafType, err)
		}
		node.Leaf = condition
		return node, nil
	}

	// 内部节点：解析逻辑运算符
	switch cnc.Operator {
	case "and":
		node.Operator = core.OpAnd
	case "or":
		node.Operator = core.OpOr
	case "not":
		node.Operator = core.OpNot
	default:
		node.Operator = core.OpAnd
	}

	// 递归解析子节点
	for _, child := range cnc.Children {
		childNode, err := materializeCondition(child, reg)
		if err != nil {
			return nil, err
		}
		node.Children = append(node.Children, childNode)
	}

	return node, nil
}
