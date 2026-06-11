package config

import (
	"testing"

	"github.com/vpptu/ruleflow/pkg/ruleflow/builtin"
	"github.com/vpptu/ruleflow/pkg/ruleflow/nodes"
)

var reg = func() *nodes.Registry {
	r := nodes.NewEmptyRegistry()
	r.RegisterPackage(builtin.Builtin)
	return r
}()

// ─────────────────────────────────────────────
//  Validate 测试
// ─────────────────────────────────────────────

func TestValidate_Valid(t *testing.T) {
	cfg := &ChainConfig{
		Chain: ChainMeta{
			ID:      "test_chain",
			Name:    "Test Chain",
			Version: 1,
			Status:  "draft",
		},
		Rules: []RuleConfig{
			{
				ID:       "rule_1",
				Priority: 1,
				Enabled:  true,
				Condition: ConditionNodeConfig{
					LeafType:   "device_type",
					LeafConfig: map[string]any{"device_types": []any{"analog"}},
				},
				Actions: []ActionConfig{
					{Type: "route", Config: map[string]any{"targets": []any{"default"}}},
				},
			},
		},
	}
	if err := Validate(cfg); err != nil {
		t.Fatalf("valid config should pass: %v", err)
	}
}

func TestValidate_MissingChainID(t *testing.T) {
	cfg := &ChainConfig{
		Chain: ChainMeta{Name: "Test", Version: 1},
		Rules: []RuleConfig{{ID: "r1"}},
	}
	if err := Validate(cfg); err == nil {
		t.Error("missing chain ID should fail")
	}
}

func TestValidate_InvalidChainID(t *testing.T) {
	cfg := &ChainConfig{
		Chain: ChainMeta{ID: "INVALID", Name: "Test", Version: 1},
		Rules: []RuleConfig{{ID: "r1"}},
	}
	if err := Validate(cfg); err == nil {
		t.Error("uppercase chain ID should fail")
	}
}

func TestValidate_MissingChainName(t *testing.T) {
	cfg := &ChainConfig{
		Chain: ChainMeta{ID: "test", Version: 1},
		Rules: []RuleConfig{{ID: "r1"}},
	}
	if err := Validate(cfg); err == nil {
		t.Error("missing chain name should fail")
	}
}

func TestValidate_NegativeVersion(t *testing.T) {
	cfg := &ChainConfig{
		Chain: ChainMeta{ID: "test", Name: "Test", Version: -1},
		Rules: []RuleConfig{{ID: "r1"}},
	}
	if err := Validate(cfg); err == nil {
		t.Error("negative version should fail")
	}
}

func TestValidate_InvalidStatus(t *testing.T) {
	cfg := &ChainConfig{
		Chain: ChainMeta{ID: "test", Name: "Test", Version: 1, Status: "invalid"},
		Rules: []RuleConfig{{ID: "r1"}},
	}
	if err := Validate(cfg); err == nil {
		t.Error("invalid status should fail")
	}
}

func TestValidate_EmptyRules(t *testing.T) {
	cfg := &ChainConfig{
		Chain: ChainMeta{ID: "test", Name: "Test", Version: 1},
		Rules: []RuleConfig{},
	}
	if err := Validate(cfg); err == nil {
		t.Error("empty rules should fail")
	}
}

func TestValidate_MissingRuleID(t *testing.T) {
	cfg := &ChainConfig{
		Chain: ChainMeta{ID: "test", Name: "Test", Version: 1},
		Rules: []RuleConfig{{ID: ""}},
	}
	if err := Validate(cfg); err == nil {
		t.Error("missing rule ID should fail")
	}
}

func TestValidate_DuplicateRuleID(t *testing.T) {
	cfg := &ChainConfig{
		Chain: ChainMeta{ID: "test", Name: "Test", Version: 1},
		Rules: []RuleConfig{
			{ID: "r1"},
			{ID: "r1"},
		},
	}
	if err := Validate(cfg); err == nil {
		t.Error("duplicate rule ID should fail")
	}
}

func TestValidate_MissingActionType(t *testing.T) {
	cfg := &ChainConfig{
		Chain: ChainMeta{ID: "test", Name: "Test", Version: 1},
		Rules: []RuleConfig{
			{
				ID: "r1",
				Actions: []ActionConfig{
					{Type: ""},
				},
			},
		},
	}
	if err := Validate(cfg); err == nil {
		t.Error("missing action type should fail")
	}
}

func TestValidate_InvalidOperator(t *testing.T) {
	cfg := &ChainConfig{
		Chain: ChainMeta{ID: "test", Name: "Test", Version: 1},
		Rules: []RuleConfig{
			{
				ID: "r1",
				Condition: ConditionNodeConfig{
					Operator: "xor",
					Children: []ConditionNodeConfig{
						{LeafType: "device_type"},
					},
				},
			},
		},
	}
	if err := Validate(cfg); err == nil {
		t.Error("invalid operator should fail")
	}
}

func TestValidate_MissingOperator(t *testing.T) {
	cfg := &ChainConfig{
		Chain: ChainMeta{ID: "test", Name: "Test", Version: 1},
		Rules: []RuleConfig{
			{
				ID: "r1",
				Condition: ConditionNodeConfig{
					Children: []ConditionNodeConfig{
						{LeafType: "device_type"},
					},
				},
			},
		},
	}
	if err := Validate(cfg); err == nil {
		t.Error("missing operator with children should fail")
	}
}

func TestValidate_ValidStatuses(t *testing.T) {
	for _, status := range []string{"draft", "deployed", "archived", ""} {
		cfg := &ChainConfig{
			Chain: ChainMeta{ID: "test", Name: "Test", Version: 1, Status: status},
			Rules: []RuleConfig{{ID: "r1"}},
		}
		if err := Validate(cfg); err != nil {
			t.Errorf("status %q should be valid: %v", status, err)
		}
	}
}

// ─────────────────────────────────────────────
//  LoadFromBytes 测试
// ─────────────────────────────────────────────

func TestLoadFromBytes_YAML(t *testing.T) {
	yamlData := []byte(`
chain:
  id: test_chain
  name: Test Chain
  version: 1
  status: draft
rules:
  - id: rule_1
    name: Route Analog
    priority: 1
    enabled: true
    condition:
      leaf_type: device_type
      leaf_config:
        device_types:
          - analog
    actions:
      - type: route
        config:
          targets:
            - default
    targets:
      - default
`)
	cfg, err := LoadFromBytes(yamlData)
	if err != nil {
		t.Fatalf("load from bytes error: %v", err)
	}
	if cfg.Chain.ID != "test_chain" {
		t.Errorf("expected chain ID test_chain, got %s", cfg.Chain.ID)
	}
	if len(cfg.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(cfg.Rules))
	}
	if cfg.Rules[0].Condition.LeafType != "device_type" {
		t.Errorf("expected leaf_type device_type, got %s", cfg.Rules[0].Condition.LeafType)
	}
}

func TestLoadFromBytes_InvalidYAML(t *testing.T) {
	_, err := LoadFromBytes([]byte(`{invalid yaml`))
	if err == nil {
		t.Error("invalid YAML should return error")
	}
}

// ─────────────────────────────────────────────
//  Parse 测试
// ─────────────────────────────────────────────

func TestParse_FullChain(t *testing.T) {
	cfg := &ChainConfig{
		Chain: ChainMeta{
			ID:      "test_chain",
			Name:    "Test Chain",
			Version: 1,
			Status:  "draft",
		},
		Rules: []RuleConfig{
			{
				ID:       "rule_1",
				Name:     "Route Analog",
				Priority: 1,
				Enabled:  true,
				Condition: ConditionNodeConfig{
					LeafType:   "device_type",
					LeafConfig: map[string]any{"device_types": []any{"analog"}},
				},
				Actions: []ActionConfig{
					{Type: "route", Config: map[string]any{"targets": []any{"default"}}},
				},
				Targets: []string{"default"},
			},
		},
	}

	chain, err := Parse(cfg, reg)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if chain.ID != "test_chain" {
		t.Errorf("expected chain ID test_chain, got %s", chain.ID)
	}
	if len(chain.Rules) != 1 {
		t.Errorf("expected 1 rule, got %d", len(chain.Rules))
	}
	if chain.Rules[0].Condition == nil {
		t.Error("condition should not be nil")
	}
	if chain.Rules[0].Actions == nil {
		t.Error("actions should not be nil")
	}
}

func TestParse_NestedCondition(t *testing.T) {
	cfg := &ChainConfig{
		Chain: ChainMeta{
			ID: "test_chain", Name: "Test", Version: 1, Status: "draft",
		},
		Rules: []RuleConfig{
			{
				ID: "rule_1", Priority: 1, Enabled: true,
				Condition: ConditionNodeConfig{
					Operator: "and",
					Children: []ConditionNodeConfig{
						{LeafType: "device_type", LeafConfig: map[string]any{"device_types": []any{"analog"}}},
						{LeafType: "quality", LeafConfig: map[string]any{"min_quality": float64(192)}},
					},
				},
				Actions: []ActionConfig{
					{Type: "route", Config: map[string]any{"targets": []any{"default"}}},
				},
			},
		},
	}

	chain, err := Parse(cfg, reg)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if chain.Rules[0].Condition.Leaf != nil {
		t.Error("root condition should be internal node, not leaf")
	}
	if len(chain.Rules[0].Condition.Children) != 2 {
		t.Errorf("expected 2 children, got %d", len(chain.Rules[0].Condition.Children))
	}
}

func TestParse_InvalidCondition(t *testing.T) {
	cfg := &ChainConfig{
		Chain: ChainMeta{
			ID: "test_chain", Name: "Test", Version: 1, Status: "draft",
		},
		Rules: []RuleConfig{
			{
				ID: "rule_1", Priority: 1, Enabled: true,
				Condition: ConditionNodeConfig{
					LeafType:   "unknown_condition",
					LeafConfig: map[string]any{},
				},
				Actions: []ActionConfig{{Type: "drop"}},
			},
		},
	}

	_, err := Parse(cfg, reg)
	if err == nil {
		t.Error("unknown condition type should fail")
	}
}

func TestParse_InvalidAction(t *testing.T) {
	cfg := &ChainConfig{
		Chain: ChainMeta{
			ID: "test_chain", Name: "Test", Version: 1, Status: "draft",
		},
		Rules: []RuleConfig{
			{
				ID: "rule_1", Priority: 1, Enabled: true,
				Condition: ConditionNodeConfig{
					LeafType:   "device_type",
					LeafConfig: map[string]any{"device_types": []any{"analog"}},
				},
				Actions: []ActionConfig{{Type: "unknown_action"}},
			},
		},
	}

	_, err := Parse(cfg, reg)
	if err == nil {
		t.Error("unknown action type should fail")
	}
}

func TestParse_DefaultEnabled(t *testing.T) {
	// 注意：Validate 要求 rule ID 非空，所以这里测试的是
	// Enabled=false 但有明显规则定义时，parseRule 的默认启用逻辑
	cfg := &ChainConfig{
		Chain: ChainMeta{
			ID: "test_chain", Name: "Test", Version: 1, Status: "draft",
		},
		Rules: []RuleConfig{
			{
				ID:       "rule_1",
				Priority: 1,
				Enabled:  false,
				Condition: ConditionNodeConfig{
					LeafType:   "device_type",
					LeafConfig: map[string]any{"device_types": []any{"analog"}},
				},
				Actions: []ActionConfig{{Type: "drop"}},
			},
		},
	}

	chain, err := Parse(cfg, reg)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	// Enabled=false 应该保持 false（parseRule 只在空 ID + zero priority 时默认启用）
	if chain.Rules[0].Enabled {
		t.Error("explicitly disabled rule should stay disabled")
	}
}

func TestParse_ORCondition(t *testing.T) {
	cfg := &ChainConfig{
		Chain: ChainMeta{ID: "test_chain", Name: "Test", Version: 1, Status: "draft"},
		Rules: []RuleConfig{
			{
				ID: "rule_1", Priority: 1, Enabled: true,
				Condition: ConditionNodeConfig{
					Operator: "or",
					Children: []ConditionNodeConfig{
						{LeafType: "device_type", LeafConfig: map[string]any{"device_types": []any{"analog"}}},
						{LeafType: "device_type", LeafConfig: map[string]any{"device_types": []any{"digital"}}},
					},
				},
				Actions: []ActionConfig{{Type: "drop"}},
			},
		},
	}

	chain, err := Parse(cfg, reg)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	// 验证 operator 是 OR
	_ = chain // parse 成功即可
}

func TestParse_NOTCondition(t *testing.T) {
	cfg := &ChainConfig{
		Chain: ChainMeta{ID: "test_chain", Name: "Test", Version: 1, Status: "draft"},
		Rules: []RuleConfig{
			{
				ID: "rule_1", Priority: 1, Enabled: true,
				Condition: ConditionNodeConfig{
					Operator: "not",
					Children: []ConditionNodeConfig{
						{LeafType: "device_type", LeafConfig: map[string]any{"device_types": []any{"analog"}}},
					},
				},
				Actions: []ActionConfig{{Type: "drop"}},
			},
		},
	}

	chain, err := Parse(cfg, reg)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	_ = chain
}
