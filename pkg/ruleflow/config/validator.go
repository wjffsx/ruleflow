package config

import (
	"fmt"
	"regexp"
)

var idPattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// Validate 校验规则链配置
func Validate(cfg *ChainConfig) error {
	if cfg.Chain.ID == "" {
		return fmt.Errorf("chain id is required")
	}
	if !idPattern.MatchString(cfg.Chain.ID) {
		return fmt.Errorf("chain id %q must match pattern %s", cfg.Chain.ID, idPattern.String())
	}
	if cfg.Chain.Name == "" {
		return fmt.Errorf("chain name is required")
	}
	if cfg.Chain.Version < 0 {
		return fmt.Errorf("chain version must be >= 0")
	}
	switch cfg.Chain.Status {
	case "draft", "deployed", "archived", "":
		// valid
	default:
		return fmt.Errorf("chain status %q is invalid, must be draft/deployed/archived", cfg.Chain.Status)
	}

	// ★ Phase 1 新增: pipelineType 校验
	switch cfg.Chain.PipelineType {
	case "analog", "digital", "meter", "":
		// valid
	default:
		return fmt.Errorf("chain pipeline_type %q is invalid, must be analog/digital/meter or empty", cfg.Chain.PipelineType)
	}

	// ★ Phase 1 新增: inputs 校验
	if err := validateInputs(cfg.Chain.Inputs, cfg.Chain.PipelineType); err != nil {
		return fmt.Errorf("chain inputs: %w", err)
	}

	if len(cfg.Rules) == 0 {
		return fmt.Errorf("rules cannot be empty")
	}

	// ★ Phase 1 新增: 构建 inputNames 索引用于规则校验
	inputNames := make(map[string]struct{})
	for _, inp := range cfg.Chain.Inputs {
		inputNames[inp.PointName] = struct{}{}
	}

	ruleIDs := make(map[string]struct{})
	for i, rc := range cfg.Rules {
		if rc.ID == "" {
			return fmt.Errorf("rule[%d] id is required", i)
		}
		if _, exists := ruleIDs[rc.ID]; exists {
			return fmt.Errorf("rule id %q is duplicated", rc.ID)
		}
		ruleIDs[rc.ID] = struct{}{}

		if err := validateRuleConfig(rc, i, inputNames); err != nil {
			return err
		}
	}

	return nil
}

// ★ Phase 1 新增: validateInputs 校验输入声明
func validateInputs(inputs []InputConfig, pipelineType string) error {
	pointNames := make(map[string]struct{})
	for i, inp := range inputs {
		if inp.PointName == "" {
			return fmt.Errorf("input[%d] point_name is required", i)
		}
		if _, exists := pointNames[inp.PointName]; exists {
			return fmt.Errorf("input point_name %q is duplicated", inp.PointName)
		}
		pointNames[inp.PointName] = struct{}{}

		// 类型一致性校验
		if pipelineType != "" && inp.PointType != pipelineType {
			return fmt.Errorf("input %q point_type %q does not match chain pipeline_type %q",
				inp.PointName, inp.PointType, pipelineType)
		}

		// point_type 校验
		switch inp.PointType {
		case "analog", "digital", "meter":
			// valid
		default:
			return fmt.Errorf("input %q point_type %q is invalid", inp.PointName, inp.PointType)
		}

		// data_type 校验
		switch inp.DataType {
		case "double", "int", "bool", "string":
			// valid
		default:
			return fmt.Errorf("input %q data_type %q is invalid", inp.PointName, inp.DataType)
		}
	}
	return nil
}

func validateRuleConfig(rc RuleConfig, index int, inputNames map[string]struct{}) error {
	// ★ Phase 1 新增: InputBindings 校验
	for _, binding := range rc.InputBindings {
		if _, declared := inputNames[binding]; !declared {
			return fmt.Errorf("rule %s input_binding %q is not declared in chain inputs", rc.ID, binding)
		}
	}

	// ★ Phase 1 新增: InputMode 校验
	switch rc.InputMode {
	case "single", "multi", "":
		// valid
	default:
		return fmt.Errorf("rule %s input_mode %q is invalid, must be single/multi or empty", rc.ID, rc.InputMode)
	}

	// ★ Phase 1 新增: 单输入模式下 InputBindings 长度校验
	if rc.InputMode == "single" && len(rc.InputBindings) > 1 {
		return fmt.Errorf("rule %s with input_mode=single cannot have more than 1 input_binding", rc.ID)
	}

	// 条件校验
	if err := validateConditionNode(rc.Condition, 0, MaxDepth); err != nil {
		return fmt.Errorf("rule %s condition: %w", rc.ID, err)
	}

	// 动作校验
	for i, ac := range rc.Actions {
		if ac.Type == "" {
			return fmt.Errorf("rule %s action[%d] type is required", rc.ID, i)
		}
	}

	return nil
}

const MaxDepth = 32

func validateConditionNode(cnc ConditionNodeConfig, depth, maxDepth int) error {
	if depth > maxDepth {
		return fmt.Errorf("condition tree too deep (max %d)", maxDepth)
	}

	// 叶节点
	if cnc.LeafType != "" {
		if cnc.LeafConfig == nil {
			// leaf_config 可以为空（如 limit_exceeded）
		}
		return nil
	}

	// 内部节点
	if cnc.Operator == "" && len(cnc.Children) > 0 {
		return fmt.Errorf("operator is required for non-leaf condition node")
	}

	switch cnc.Operator {
	case "and", "or", "not", "":
		// valid
	default:
		return fmt.Errorf("unknown operator %q", cnc.Operator)
	}

	for _, child := range cnc.Children {
		if err := validateConditionNode(child, depth+1, maxDepth); err != nil {
			return err
		}
	}

	return nil
}
