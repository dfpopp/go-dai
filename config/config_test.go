package config

import (
	"encoding/json"
	"os"
	"testing"
	"time"
)

// TestConfigAccessor 测试配置访问器
func TestConfigAccessor(t *testing.T) {
	// 创建测试配置
	testConfig := map[string]interface{}{
		"string_value": "test",
		"int_value":    123,
		"bool_value":   true,
		"float_value":  123.456,
		"nested": map[string]interface{}{
			"inner_value": "inner",
		},
		"array": []interface{}{1, 2, 3},
	}
	
	// 创建配置访问器
	accessor := NewConfigAccessor(testConfig)
	
	// 测试Get方法
	innerAccessor := accessor.Get("nested")
	if !innerAccessor.Exists() {
		t.Error("Nested config should exist")
	}
	
	// 测试不存在的配置
	notExistAccessor := accessor.Get("not_exist")
	if notExistAccessor.Exists() {
		t.Error("Non-existent config should not exist")
	}
	
	// 测试String方法
	if accessor.Get("string_value").String() != "test" {
		t.Error("String value should be 'test'")
	}
	
	// 测试Int方法
	if accessor.Get("int_value").Int() != 123 {
		t.Error("Int value should be 123")
	}
	
	// 测试Bool方法
	if !accessor.Get("bool_value").Bool() {
		t.Error("Bool value should be true")
	}
	
	// 测试Float方法
	if accessor.Get("float_value").Float() != 123.456 {
		t.Error("Float value should be 123.456")
	}
	
	// 测试Map方法
	if accessor.Get("nested").Map() == nil {
		t.Error("Map value should not be nil")
	}
	
	// 测试Slice方法
	if accessor.Get("array").Slice() == nil {
		t.Error("Slice value should not be nil")
	}
	
	// 测试GetAll方法
	allConfig := accessor.GetAll()
	if allConfig == nil {
		t.Error("GetAll should return non-nil map")
	}
	if allConfig["string_value"] != "test" {
		t.Error("GetAll should return correct string value")
	}
}

// TestMemoryConfig 测试内存配置
func TestMemoryConfig(t *testing.T) {
	// 创建内存配置
	memConfig := NewMemoryConfig()
	
	// 测试Get方法
	if memConfig.Get("test") != nil {
		t.Error("Non-existent key should return nil")
	}
	
	// 测试Exists方法
	if memConfig.Exists("test") {
		t.Error("Non-existent key should not exist")
	}
	
	// 测试GetAll方法
	if memConfig.GetAll() == nil {
		t.Error("GetAll should return non-nil map")
	}
	if len(memConfig.GetAll()) != 0 {
		t.Error("Empty memory config should return empty map")
	}
	
	// 测试GetString方法
	if memConfig.GetString("test") != "" {
		t.Error("Non-existent key should return empty string")
	}
	
	// 测试GetInt方法
	if memConfig.GetInt("test") != 0 {
		t.Error("Non-existent key should return 0")
	}
	
	// 测试GetBool方法
	if memConfig.GetBool("test") {
		t.Error("Non-existent key should return false")
	}
	
	// 测试GetFloat64方法
	if memConfig.GetFloat64("test") != 0 {
		t.Error("Non-existent key should return 0")
	}
}

// TestConfigFileWatcher 测试配置文件监听器
func TestConfigFileWatcher(t *testing.T) {
	// 创建临时配置文件
	testConfig := AppConfig{
		Name: "test",
		Env:  "test",
	}
	
	configData, err := json.Marshal(testConfig)
	if err != nil {
		t.Fatalf("Failed to marshal test config: %v", err)
	}
	
	// 创建临时文件
	tempFile, err := os.CreateTemp("", "test-config-")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tempFilePath := tempFile.Name()
	defer os.Remove(tempFilePath)
	
	// 写入测试配置
	_, err = tempFile.Write(configData)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}
	tempFile.Close()
	
	// 创建配置文件监听器
	watcher := NewConfigFileWatcher(tempFilePath, 1*time.Second)
	defer watcher.Stop()
	
	// 启动监听器
	watcher.Start("app")
	
	// 等待一段时间
	time.Sleep(2 * time.Second)
	
	// 停止监听器
	watcher.Stop()
}

// TestLoadCustomConfig 测试加载自定义配置
func TestLoadCustomConfig(t *testing.T) {
	// 创建临时配置文件
	testConfig := map[string]interface{}{
		"test_key": "test_value",
	}
	
	configData, err := json.Marshal(testConfig)
	if err != nil {
		t.Fatalf("Failed to marshal test config: %v", err)
	}
	
	// 创建临时文件
	tempFile, err := os.CreateTemp("", "test-custom-config-")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tempFilePath := tempFile.Name()
	defer os.Remove(tempFilePath)
	
	// 写入测试配置
	_, err = tempFile.Write(configData)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}
	tempFile.Close()
	
	// 加载自定义配置
	err = LoadCustomConfig(tempFilePath)
	if err != nil {
		t.Fatalf("Failed to load custom config: %v", err)
	}
	
	// 等待一段时间，让配置加载完成
	time.Sleep(1 * time.Second)
	
	// 测试获取配置
	if customConfig == nil {
		t.Error("Custom config should not be nil")
	}
	
	// 获取配置访问器
	accessor := customConfig.GetConfig()
	if accessor == nil {
		t.Error("Config accessor should not be nil")
	}
	
	// 测试配置值
	if accessor.Get("test_key").String() != "test_value" {
		t.Error("Custom config value should be 'test_value'")
	}
	
	// 停止配置监听
	customConfig.Stop()
}

// TestRegisterConfigChangeHook 测试注册配置变更钩子
func TestRegisterConfigChangeHook(t *testing.T) {
	// 定义测试变量
	hookCalled := false
	var hookConfigType string
	var hookConfig interface{}
	
	// 注册配置变更钩子
	RegisterConfigChangeHook(func(configType string, config interface{}) error {
		hookCalled = true
		hookConfigType = configType
		hookConfig = config
		return nil
	})
	
	// 创建测试配置
	testConfig := AppConfig{
		Name: "test",
		Env:  "test",
	}
	
	// 通知配置变更
	notifyConfigChange("test", testConfig)
	
	// 验证钩子是否被调用
	if !hookCalled {
		t.Error("Config change hook should be called")
	}
	
	if hookConfigType != "test" {
		t.Error("Config type should be 'test'")
	}
	
	if hookConfig != testConfig {
		t.Error("Config should be the test config")
	}
}

// TestStopConfigFileWatchers 测试停止配置文件监听器
func TestStopConfigFileWatchers(t *testing.T) {
	// 停止所有配置文件监听器
	StopConfigFileWatchers()
	
	// 验证监听器是否被清空
	if len(configFileWatchers) != 0 {
		t.Error("Config file watchers should be empty")
	}
}
