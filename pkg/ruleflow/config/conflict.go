package config

import "fmt"

// ─────────────────────────────────────────────
//  冲突检测
// ─────────────────────────────────────────────

// Conflict 冲突描述
type Conflict struct {
	Type     string   `json:"type"` // "action_conflict" / "condition_overlap"
	RuleIDs  []string `json:"rule_ids"`
	Message  string   `json:"message"`
	Severity string   `json:"severity"` // "warning" / "error"
}

// DetectConflicts 检测规则链内的潜在冲突
func DetectConflicts(cfg *ChainConfig) []Conflict {
	var conflicts []Conflict

	for i, r1 := range cfg.Rules {
		for j, r2 := range cfg.Rules {
			if i >= j {
				continue
			}

			// 1. 动作冲突：两条规则对同一数据点执行矛盾动作
			if hasActionConflict(r1, r2) {
				conflicts = append(conflicts, Conflict{
					Type:     "action_conflict",
					RuleIDs:  []string{r1.ID, r2.ID},
					Message:  fmt.Sprintf("规则 %s 和 %s 可能对同一数据点产生矛盾修改", r1.ID, r2.ID),
					Severity: "warning",
				})
			}

			// 2. 条件重叠：两条规则条件完全相同，高优先级可能屏蔽低优先级
			if hasConditionOverlap(r1, r2) {
				conflicts = append(conflicts, Conflict{
					Type:     "condition_overlap",
					RuleIDs:  []string{r1.ID, r2.ID},
					Message:  fmt.Sprintf("规则 %s 和 %s 条件重叠，优先级 %d 可能屏蔽 %d", r1.ID, r2.ID, r1.Priority, r2.Priority),
					Severity: "warning",
				})
			}
		}
	}

	return conflicts
}

// hasActionConflict 检测两条规则是否有动作冲突
func hasActionConflict(r1, r2 RuleConfig) bool {
	// 检测 transform 动作冲突：两条规则都执行 transform
	hasTransform1 := false
	hasTransform2 := false
	for _, a := range r1.Actions {
		if a.Type == "transform" {
			hasTransform1 = true
			break
		}
	}
	for _, a := range r2.Actions {
		if a.Type == "transform" {
			hasTransform2 = true
			break
		}
	}
	if hasTransform1 && hasTransform2 {
		return true
	}

	// 检测 drop + 其他动作冲突
	hasDrop1 := false
	hasDrop2 := false
	for _, a := range r1.Actions {
		if a.Type == "drop" {
			hasDrop1 = true
			break
		}
	}
	for _, a := range r2.Actions {
		if a.Type == "drop" {
			hasDrop2 = true
			break
		}
	}
	// 一条 drop 另一条 transform/route，可能冲突
	if (hasDrop1 && !hasDrop2 && len(r2.Actions) > 0) ||
		(hasDrop2 && !hasDrop1 && len(r1.Actions) > 0) {
		return true
	}

	return false
}

// hasConditionOverlap 检测两条规则条件是否重叠
func hasConditionOverlap(r1, r2 RuleConfig) bool {
	// 简化检测：如果两条规则的条件类型和配置完全相同
	if r1.Condition.LeafType != "" && r1.Condition.LeafType == r2.Condition.LeafType {
		return true
	}
	return false
}
