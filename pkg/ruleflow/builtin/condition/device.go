// Package condition provides builtin condition nodes
package condition

import (
	"context"
	"fmt"

	"github.com/wjffsx/ruleflow/pkg/ruleflow/core"
)

// ─────────────────────────────────────────────
//  设备相关条件
// ─────────────────────────────────────────────

// DeviceTypeCondition 设备类型条件
type DeviceTypeCondition struct {
	IDValue     string              `json:"id"`
	DeviceTypes []string            `json:"device_types"`
	typeSet     map[string]struct{} // 预编译哈希集
}

func NewDeviceTypeCondition(id string, types []string) *DeviceTypeCondition {
	typeSet := make(map[string]struct{}, len(types))
	for _, t := range types {
		typeSet[t] = struct{}{}
	}
	return &DeviceTypeCondition{IDValue: id, DeviceTypes: types, typeSet: typeSet}
}

func (c *DeviceTypeCondition) Evaluate(_ context.Context, data core.DataContext) bool {
	_, ok := c.typeSet[data.PointType()] // O(1) 查找
	return ok
}

func (c *DeviceTypeCondition) ID() string   { return c.IDValue }
func (c *DeviceTypeCondition) Type() string { return "device_type" }
func (c *DeviceTypeCondition) Description() string {
	return fmt.Sprintf("device type in %v", c.DeviceTypes)
}

// DeviceIDCondition 设备 ID 条件
type DeviceIDCondition struct {
	IDValue   string   `json:"id"`
	DeviceIDs []string `json:"device_ids"`
	idSet     map[string]struct{}
}

func NewDeviceIDCondition(id string, ids []string) *DeviceIDCondition {
	idSet := make(map[string]struct{}, len(ids))
	for _, d := range ids {
		idSet[d] = struct{}{}
	}
	return &DeviceIDCondition{IDValue: id, DeviceIDs: ids, idSet: idSet}
}

func (c *DeviceIDCondition) Evaluate(_ context.Context, data core.DataContext) bool {
	_, ok := c.idSet[data.DeviceID()]
	return ok
}

func (c *DeviceIDCondition) ID() string          { return c.IDValue }
func (c *DeviceIDCondition) Type() string        { return "device_id" }
func (c *DeviceIDCondition) Description() string { return fmt.Sprintf("device ID in %v", c.DeviceIDs) }
