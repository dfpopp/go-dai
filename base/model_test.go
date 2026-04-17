package base

import (
	"testing"
)

// TestBaseModel_Init 测试模型初始化
func TestBaseModel_Init(t *testing.T) {
	model := &BaseModel{}
	model.Init()
	
	// 验证组件是否初始化
	model.LogInfo("测试模型日志")
}

// TestBaseModel_StringToFulltextIndexStr 测试全文索引字符串生成
func TestBaseModel_StringToFulltextIndexStr(t *testing.T) {
	model := &BaseModel{}
	model.Init()
	
	// 测试空字符串
	if result := model.StringToFulltextIndexStr(""); result != "" {
		t.Error("Empty string should return empty string")
	}
	
	// 测试数字字符串
	result := model.StringToFulltextIndexStr("13838385687")
	if result == "" {
		t.Error("Number string should return non-empty string")
	}
	
	// 测试中文字符串
	result = model.StringToFulltextIndexStr("张三李四")
	if result == "" {
		t.Error("Chinese string should return non-empty string")
	}
	
	// 测试混合字符串
	result = model.StringToFulltextIndexStr("Abc12")
	if result == "" {
		t.Error("Mixed string should return non-empty string")
	}
}

// TestBaseModel_StringToSearchFulltextStr 测试搜索全文索引字符串生成
func TestBaseModel_StringToSearchFulltextStr(t *testing.T) {
	model := &BaseModel{}
	model.Init()
	
	// 测试空字符串
	if result := model.StringToSearchFulltextStr(""); result != "" {
		t.Error("Empty string should return empty string")
	}
	
	// 测试数字字符串
	result := model.StringToSearchFulltextStr("13838385687")
	if result == "" {
		t.Error("Number string should return non-empty string")
	}
	
	// 测试中文字符串
	result = model.StringToSearchFulltextStr("张三李四")
	if result == "" {
		t.Error("Chinese string should return non-empty string")
	}
	
	// 测试混合字符串
	result = model.StringToSearchFulltextStr("Abc12")
	if result == "" {
		t.Error("Mixed string should return non-empty string")
	}
}
