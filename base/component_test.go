package base

import (
	"testing"
)

// TestBaseComponent_Init 测试组件初始化
func TestBaseComponent_Init(t *testing.T) {
	component := &BaseComponent{}
	component.Init()
	
	// 验证日志实例是否初始化
	// 由于日志实例是私有字段，我们通过调用日志方法来验证
	component.LogInfo("测试日志")
	component.LogError("测试错误日志")
	component.LogWarn("测试警告日志")
	
	// 测试nil指针的情况
	var nilComponent *BaseComponent
	nilComponent.LogInfo("测试nil组件日志")
	nilComponent.LogError("测试nil组件错误日志")
	nilComponent.LogWarn("测试nil组件警告日志")
}

// TestBaseComponent_LogMethods 测试日志方法
func TestBaseComponent_LogMethods(t *testing.T) {
	component := &BaseComponent{}
	component.Init()
	
	// 测试各种日志方法
	component.LogInfo("信息日志测试")
	component.LogWarn("警告日志测试")
	component.LogError("错误日志测试")
	
	// 测试nil指针的日志方法
	var nilComponent *BaseComponent
	nilComponent.LogInfo("nil组件信息日志测试")
	nilComponent.LogWarn("nil组件警告日志测试")
	nilComponent.LogError("nil组件错误日志测试")
}
