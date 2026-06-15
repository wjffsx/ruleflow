// Package action provides builtin action nodes
package action

import (
	"context"
	"fmt"
	"strconv"

	"github.com/wjffsx/ruleflow/pkg/ruleflow/core"
)

// ─────────────────────────────────────────────
//  BitUnpackAction — 位拆解动作
// ─────────────────────────────────────────────

// BitUnpackAction 位拆解动作
// 将一个整数值拆解为多个布尔值，存储到 Tag 中
type BitUnpackAction struct {
	IDValue    string   `json:"id"`
	OutputTags []string `json:"output_tags"` // 输出标签名列表，如 ["bit0", "bit1", "bit2"]
	StartBit   int      `json:"start_bit"`   // 起始位位置
}

// NewBitUnpackAction 创建位拆解动作
func NewBitUnpackAction(id string, outputTags []string, startBit int) *BitUnpackAction {
	return &BitUnpackAction{
		IDValue:    id,
		OutputTags: outputTags,
		StartBit:   startBit,
	}
}

func (a *BitUnpackAction) Execute(_ context.Context, data core.DataContext) error {
	val := uint64(data.Value())
	for i, tag := range a.OutputTags {
		bitPos := a.StartBit + i
		bitVal := (val >> bitPos) & 1
		data.SetTag(tag, strconv.FormatUint(bitVal, 10))
	}
	return nil
}

func (a *BitUnpackAction) ID() string   { return a.IDValue }
func (a *BitUnpackAction) Type() string { return "bit_unpack" }
func (a *BitUnpackAction) Description() string {
	return fmt.Sprintf("bit_unpack to %v starting at bit %d", a.OutputTags, a.StartBit)
}

// ─────────────────────────────────────────────
//  BitPackAction — 位合并动作
// ─────────────────────────────────────────────

// BitPackAction 位合并动作
// 将多个布尔值从 Tag 中读取，合并为一个整数
type BitPackAction struct {
	IDValue     string   `json:"id"`
	InputTags   []string `json:"input_tags"`   // 输入标签名列表
	OutputField string   `json:"output_field"` // 输出到哪个字段，如 "value" 或 tag 名
	StartBit    int      `json:"start_bit"`    // 起始位位置
}

// NewBitPackAction 创建位合并动作
func NewBitPackAction(id string, inputTags []string, outputField string, startBit int) *BitPackAction {
	return &BitPackAction{
		IDValue:     id,
		InputTags:   inputTags,
		OutputField: outputField,
		StartBit:    startBit,
	}
}

func (a *BitPackAction) Execute(_ context.Context, data core.DataContext) error {
	var result uint64
	for i, tag := range a.InputTags {
		tagVal := data.GetTag(tag)
		if tagVal == "1" || tagVal == "true" {
			bitPos := a.StartBit + i
			result |= (1 << bitPos)
		}
	}

	if a.OutputField == "value" {
		data.SetValue(float64(result))
	} else {
		data.SetTag(a.OutputField, strconv.FormatUint(result, 10))
	}
	return nil
}

func (a *BitPackAction) ID() string   { return a.IDValue }
func (a *BitPackAction) Type() string { return "bit_pack" }
func (a *BitPackAction) Description() string {
	return fmt.Sprintf("bit_pack from %v to %s starting at bit %d", a.InputTags, a.OutputField, a.StartBit)
}
