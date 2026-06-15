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
	// 条件类型不同，不重叠
	if r1.Condition.LeafType == "" || r1.Condition.LeafType != r2.Condition.LeafType {
		return false
	}

	// 同类型条件，比较配置参数是否重叠
	return configOverlaps(r1.Condition.LeafType, r1.Condition.LeafConfig, r2.Condition.LeafConfig)
}

// configOverlaps 根据条件类型进行精细化重叠判断
func configOverlaps(leafType string, c1, c2 map[string]any) bool {
	switch leafType {
	case "value_range":
		return valueRangeOverlaps(c1, c2)
	case "device_type", "device_id", "point_name":
		return stringSetOverlaps(c1, c2, "values")
	case "quality":
		return qualityOverlaps(c1, c2)
	default:
		// 未知类型保守判断为重叠
		return true
	}
}

// valueRangeOverlaps 判断两个值范围是否重叠
func valueRangeOverlaps(c1, c2 map[string]any) bool {
	min1, max1 := getFloat(c1, "min"), getFloat(c1, "max")
	min2, max2 := getFloat(c2, "min"), getFloat(c2, "max")
	// 区间重叠判断：min1 <= max2 && min2 <= max1
	return min1 <= max2 && min2 <= max1
}

// stringSetOverlaps 判断两个字符串集合是否有交集
func stringSetOverlaps(c1, c2 map[string]any, key string) bool {
	set1 := getStringSlice(c1, key)
	set2 := getStringSlice(c2, key)
	if len(set1) == 0 || len(set2) == 0 {
		return false
	}
	for _, s1 := range set1 {
		for _, s2 := range set2 {
			if s1 == s2 {
				return true
			}
		}
	}
	return false
}

// qualityOverlaps 判断质量码条件是否重叠
func qualityOverlaps(c1, c2 map[string]any) bool {
	q1 := getInt(c1, "quality")
	q2 := getInt(c2, "quality")
	return q1 == q2
}

// getFloat 从配置中获取浮点数
func getFloat(c map[string]any, key string) float64 {
	if v, ok := c[key]; ok {
		switch n := v.(type) {
		case float64:
			return n
		case float32:
			return float64(n)
		case int:
			return float64(n)
		case int64:
			return float64(n)
		}
	}
	return 0
}

// getInt 从配置中获取整数
func getInt(c map[string]any, key string) int {
	if v, ok := c[key]; ok {
		switch n := v.(type) {
		case int:
			return n
		case int64:
			return int(n)
		case float64:
			return int(n)
		}
	}
	return 0
}

// getStringSlice 从配置中获取字符串切片
func getStringSlice(c map[string]any, key string) []string {
	if v, ok := c[key]; ok {
		if arr, ok := v.([]any); ok {
			result := make([]string, 0, len(arr))
			for _, item := range arr {
				if s, ok := item.(string); ok {
					result = append(result, s)
				}
			}
			return result
		}
		if arr, ok := v.([]string); ok {
			return arr
		}
	}
	return nil
}
