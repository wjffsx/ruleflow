package ext

import (
	"context"
	"fmt"

	"github.com/wjffsx/ruleflow/pkg/ruleflow/core"
)

// ─────────────────────────────────────────────
//  MultiDeviceControlAction — 多设备联动动作
// ─────────────────────────────────────────────

// DeviceExecutor 设备执行器接口（由服务层实现）
type DeviceExecutor interface {
	Execute(deviceID string, command string, params map[string]any) error
}

// MultiDeviceControlAction 多设备联动控制动作
type MultiDeviceControlAction struct {
	IDValue  string
	Targets  []string       // 目标设备列表
	Command  string         // 控制命令
	Params   map[string]any // 控制参数
	Executor DeviceExecutor // 注入的设备执行器
}

var _ core.Action = (*MultiDeviceControlAction)(nil)

func NewMultiDeviceControlAction(id string, targets []string, command string, params map[string]any, executor DeviceExecutor) *MultiDeviceControlAction {
	return &MultiDeviceControlAction{
		IDValue:  id,
		Targets:  targets,
		Command:  command,
		Params:   params,
		Executor: executor,
	}
}

func (a *MultiDeviceControlAction) Execute(_ context.Context, data core.DataContext) error {
	if a.Executor == nil {
		return nil
	}

	var errs []error
	for _, target := range a.Targets {
		if err := a.Executor.Execute(target, a.Command, a.Params); err != nil {
			errs = append(errs, fmt.Errorf("device %s: %w", target, err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("%d/%d devices failed", len(errs), len(a.Targets))
	}
	return nil
}

func (a *MultiDeviceControlAction) ID() string          { return a.IDValue }
func (a *MultiDeviceControlAction) Type() string        { return "multi_device_control" }
func (a *MultiDeviceControlAction) Description() string { return "multi device control" }
