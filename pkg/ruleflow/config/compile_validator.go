package config

import (
	"fmt"
)

// ─────────────────────────────────────────────
//  编译时校验器（Phase 1 新增）
// ─────────────────────────────────────────────

// CompileValidator 编译时校验器
// 校验条件/动作配置中的 point_name 引用是否在 inputs 中声明
type CompileValidator struct {
	inputNames map[string]struct{}
}

// NewCompileValidator 创建编译校验器
func NewCompileValidator(inputs []InputConfig) *CompileValidator {
	inputNames := make(map[string]struct{})
	for _, inp := range inputs {
		inputNames[inp.PointName] = struct{}{}
	}
	return &CompileValidator{inputNames: inputNames}
}

// ValidateConditionNode 校验条件节点中的 point_name 引用
func (v *CompileValidator) ValidateConditionNode(cnc ConditionNodeConfig) error {
	return v.validateConditionNodeRecursive(cnc, "")
}

func (v *CompileValidator) validateConditionNodeRecursive(cnc ConditionNodeConfig, path string) error {
	// 叶节点：校验 leaf_config.point_name
	if cnc.LeafType != "" {
		return v.validateLeafConfig(cnc.LeafType, cnc.LeafConfig, path)
	}

	// 内部节点：递归校验子节点
	for i, child := range cnc.Children {
		childPath := fmt.Sprintf("%s.children[%d]", path, i)
		if path == "" {
			childPath = fmt.Sprintf("children[%d]", i)
		}
		if err := v.validateConditionNodeRecursive(child, childPath); err != nil {
			return err
		}
	}

	return nil
}

// validateLeafConfig 校验叶节点配置
func (v *CompileValidator) validateLeafConfig(leafType string, config map[string]any, path string) error {
	// 需要校验 point_name 的条件类型
	pointNameRequiredTypes := []string{
		"point_value",     // 点值条件
		"point_quality",   // 点质量条件
		"point_compare",   // 点比较条件
		"point_threshold", // 点阈值条件
		"point_limit",     // 点限值条件
		"point_range",     // 点范围条件
		"point_change",    // 点变化条件
	}

	// 检查是否是需要 point_name 的条件类型
	needsPointName := false
	for _, t := range pointNameRequiredTypes {
		if leafType == t {
			needsPointName = true
			break
		}
	}

	if !needsPointName {
		return nil
	}

	// 校验 point_name 是否存在且已声明
	pointName, ok := config["point_name"].(string)
	if !ok || pointName == "" {
		return fmt.Errorf("condition at %s: point_name is required for type %s", path, leafType)
	}

	if _, declared := v.inputNames[pointName]; !declared {
		return fmt.Errorf("condition at %s: point_name %q is not declared in chain inputs", path, pointName)
	}

	return nil
}

// ValidateActionConfig 校验动作配置中的 point_name 引用
func (v *CompileValidator) ValidateActionConfig(ac ActionConfig) error {
	// 需要校验 point_name 的动作类型
	pointNameRequiredTypes := []string{
		"set_value",       // 设置值
		"set_quality",     // 设置质量
		"set_output",      // 设置输出
		"calc_expression", // 计算表达式
		"transform",       // 数据转换
	}

	// 检查是否是需要 point_name 的动作类型
	needsPointName := false
	for _, t := range pointNameRequiredTypes {
		if ac.Type == t {
			needsPointName = true
			break
		}
	}

	if !needsPointName {
		return nil
	}

	// 校验 point_name 是否存在且已声明
	pointName, ok := ac.Config["point_name"].(string)
	if !ok || pointName == "" {
		return fmt.Errorf("action type %s: point_name is required", ac.Type)
	}

	if _, declared := v.inputNames[pointName]; !declared {
		return fmt.Errorf("action type %s: point_name %q is not declared in chain inputs", ac.Type, pointName)
	}

	return nil
}

// ValidateRule 校验规则配置
func (v *CompileValidator) ValidateRule(rc RuleConfig) error {
	// 1. 校验条件树
	if err := v.ValidateConditionNode(rc.Condition); err != nil {
		return fmt.Errorf("rule %s: %w", rc.ID, err)
	}

	// 2. 校验动作链
	for i, ac := range rc.Actions {
		if err := v.ValidateActionConfig(ac); err != nil {
			return fmt.Errorf("rule %s action[%d]: %w", rc.ID, i, err)
		}
	}

	return nil
}

// ValidateChain 校验整个规则链配置
func (v *CompileValidator) ValidateChain(cfg *ChainConfig) error {
	for _, rc := range cfg.Rules {
		if err := v.ValidateRule(rc); err != nil {
			return err
		}
	}
	return nil
}

// ─────────────────────────────────────────────
//  辅助函数
// ─────────────────────────────────────────────

// ExtractPointNames 从条件配置中提取所有 point_name
func ExtractPointNames(cnc ConditionNodeConfig) []string {
	var names []string
	extractPointNamesRecursive(cnc, &names)
	return names
}

func extractPointNamesRecursive(cnc ConditionNodeConfig, names *[]string) {
	if cnc.LeafType != "" {
		if pointName, ok := cnc.LeafConfig["point_name"].(string); ok && pointName != "" {
			*names = append(*names, pointName)
		}
		return
	}

	for _, child := range cnc.Children {
		extractPointNamesRecursive(child, names)
	}
}

// ExtractActionPointNames 从动作配置中提取所有 point_name
func ExtractActionPointName(ac ActionConfig) string {
	if pointName, ok := ac.Config["point_name"].(string); ok {
		return pointName
	}
	return ""
}
