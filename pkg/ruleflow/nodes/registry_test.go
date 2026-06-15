package nodes

import (
	"context"
	"testing"

	"github.com/wjffsx/ruleflow/pkg/ruleflow/core"
)

// mockCondition for testing
type mockCondition struct {
	id string
}

func (m *mockCondition) ID() string                          { return m.id }
func (m *mockCondition) Type() string                        { return "mock" }
func (m *mockCondition) Description() string                 { return "mock condition" }
func (m *mockCondition) Evaluate(_ context.Context, _ core.DataContext) bool { return true }

// mockAction for testing
type mockAction struct {
	id string
}

func (m *mockAction) ID() string                          { return m.id }
func (m *mockAction) Type() string                        { return "mock" }
func (m *mockAction) Description() string                 { return "mock action" }
func (m *mockAction) Execute(_ context.Context, _ core.DataContext) error { return nil }

// mockNodePackage for testing RegisterPackage
type mockNodePackage struct{}

func (mockNodePackage) GetConditionFactories() map[string]core.ConditionFactory {
	return map[string]core.ConditionFactory{
		"mock_cond": func(id string, _ map[string]any) (core.Condition, error) {
			return &mockCondition{id: id}, nil
		},
	}
}

func (mockNodePackage) GetActionFactories() map[string]core.ActionFactory {
	return map[string]core.ActionFactory{
		"mock_act": func(id string, _ map[string]any) (core.Action, error) {
			return &mockAction{id: id}, nil
		},
	}
}

func TestNewEmptyRegistry(t *testing.T) {
	r := NewEmptyRegistry()
	if r == nil {
		t.Fatal("NewEmptyRegistry returned nil")
	}
	if r.conditions == nil {
		t.Error("conditions map is nil")
	}
	if r.actions == nil {
		t.Error("actions map is nil")
	}
}

func TestRegistry_RegisterCondition(t *testing.T) {
	r := NewEmptyRegistry()
	factory := func(id string, _ map[string]any) (core.Condition, error) {
		return &mockCondition{id: id}, nil
	}

	r.RegisterCondition("test_cond", factory)

	if !r.HasCondition("test_cond") {
		t.Error("condition not registered")
	}
	if r.HasCondition("unknown") {
		t.Error("unknown condition should not exist")
	}
}

func TestRegistry_RegisterAction(t *testing.T) {
	r := NewEmptyRegistry()
	factory := func(id string, _ map[string]any) (core.Action, error) {
		return &mockAction{id: id}, nil
	}

	r.RegisterAction("test_act", factory)

	if !r.HasAction("test_act") {
		t.Error("action not registered")
	}
	if r.HasAction("unknown") {
		t.Error("unknown action should not exist")
	}
}

func TestRegistry_RegisterPackage(t *testing.T) {
	r := NewEmptyRegistry()
	r.RegisterPackage(mockNodePackage{})

	if !r.HasCondition("mock_cond") {
		t.Error("package condition not registered")
	}
	if !r.HasAction("mock_act") {
		t.Error("package action not registered")
	}
}

func TestRegistry_RegisterConditions(t *testing.T) {
	r := NewEmptyRegistry()
	factories := map[string]core.ConditionFactory{
		"cond1": func(id string, _ map[string]any) (core.Condition, error) {
			return &mockCondition{id: id}, nil
		},
		"cond2": func(id string, _ map[string]any) (core.Condition, error) {
			return &mockCondition{id: id}, nil
		},
	}

	r.RegisterConditions(factories)

	if !r.HasCondition("cond1") || !r.HasCondition("cond2") {
		t.Error("conditions not registered")
	}
}

func TestRegistry_RegisterActions(t *testing.T) {
	r := NewEmptyRegistry()
	factories := map[string]core.ActionFactory{
		"act1": func(id string, _ map[string]any) (core.Action, error) {
			return &mockAction{id: id}, nil
		},
		"act2": func(id string, _ map[string]any) (core.Action, error) {
			return &mockAction{id: id}, nil
		},
	}

	r.RegisterActions(factories)

	if !r.HasAction("act1") || !r.HasAction("act2") {
		t.Error("actions not registered")
	}
}

func TestRegistry_CreateCondition(t *testing.T) {
	r := NewEmptyRegistry()
	r.RegisterCondition("test_cond", func(id string, _ map[string]any) (core.Condition, error) {
		return &mockCondition{id: id}, nil
	})

	// Test successful creation
	cond, err := r.CreateCondition("test_cond", "cond_001", nil)
	if err != nil {
		t.Fatalf("CreateCondition failed: %v", err)
	}
	if cond.ID() != "cond_001" {
		t.Errorf("expected id cond_001, got %s", cond.ID())
	}

	// Test unknown type
	_, err = r.CreateCondition("unknown", "cond_002", nil)
	if err == nil {
		t.Error("expected error for unknown type")
	}
}

func TestRegistry_CreateAction(t *testing.T) {
	r := NewEmptyRegistry()
	r.RegisterAction("test_act", func(id string, _ map[string]any) (core.Action, error) {
		return &mockAction{id: id}, nil
	})

	// Test successful creation
	act, err := r.CreateAction("test_act", "act_001", nil)
	if err != nil {
		t.Fatalf("CreateAction failed: %v", err)
	}
	if act.ID() != "act_001" {
		t.Errorf("expected id act_001, got %s", act.ID())
	}

	// Test unknown type
	_, err = r.CreateAction("unknown", "act_002", nil)
	if err == nil {
		t.Error("expected error for unknown type")
	}
}

func TestRegistry_ListConditionTypes(t *testing.T) {
	r := NewEmptyRegistry()
	r.RegisterCondition("cond1", nil)
	r.RegisterCondition("cond2", nil)
	r.RegisterCondition("cond3", nil)

	types := r.ListConditionTypes()
	if len(types) != 3 {
		t.Errorf("expected 3 types, got %d", len(types))
	}
}

func TestRegistry_ListActionTypes(t *testing.T) {
	r := NewEmptyRegistry()
	r.RegisterAction("act1", nil)
	r.RegisterAction("act2", nil)

	types := r.ListActionTypes()
	if len(types) != 2 {
		t.Errorf("expected 2 types, got %d", len(types))
	}
}

func TestRegistry_ConcurrentAccess(t *testing.T) {
	r := NewEmptyRegistry()

	// Concurrent registration
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(n int) {
			r.RegisterCondition("cond_"+string(rune(n)), nil)
			r.HasCondition("cond_"+string(rune(n)))
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}