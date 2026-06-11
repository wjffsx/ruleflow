// Package datacontext 提供规则引擎的 DataContext 实现。
//
// V12 架构边界说明：
//   - 本包提供 DataContext 的内部实现，用于测试和示例场景
//   - 生产环境推荐使用 adapter 包的 DataContextAdapter（零拷贝适配器）
//   - 本包实现不依赖外部数据模型，可独立使用
//
// 职责边界：
//   - adapter: 外部数据点适配（零拷贝，依赖 DataPoint 接口）
//   - datacontext: 内部实现（MapDataContext, MultiDataContext）
//
// 遵循架构设计 V7：
//   - core 子包只保留接口与契约
//   - 具体实现位于此包
//   - 通过类型别名保持 API 向后兼容
package datacontext
