package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Config 配置管理接口
type Config interface {
	// 获取配置值
	Get(key string) interface{}
	GetString(key string) string
	GetInt(key string) int
	GetBool(key string) bool
	GetFloat64(key string) float64

	// 检查配置是否存在
	Exists(key string) bool

	// 批量获取配置
	GetAll() map[string]interface{}
}

// MemoryConfig 内存配置实现
type MemoryConfig struct {
	configs sync.Map
}

// CustomConfig 自定义配置
type CustomConfig struct {
	config   map[string]interface{}
	filePath string
	watcher  *ConfigFileWatcher
	mu       sync.RWMutex
}

// ConfigAccessor 配置访问器
type ConfigAccessor struct {
	value interface{}
}

// NewConfigAccessor 创建配置访问器
func NewConfigAccessor(value interface{}) *ConfigAccessor {
	return &ConfigAccessor{value: value}
}

// Get 获取子配置
func (a *ConfigAccessor) Get(key string) *ConfigAccessor {
	if a.value == nil {
		return NewConfigAccessor(nil)
	}

	if m, ok := a.value.(map[string]interface{}); ok {
		v, exists := m[key]
		if exists {
			return NewConfigAccessor(v)
		}
	}

	return NewConfigAccessor(nil)
}

// String 获取字符串值
func (a *ConfigAccessor) String() string {
	if a.value == nil {
		return ""
	}

	if s, ok := a.value.(string); ok {
		return s
	}

	return fmt.Sprintf("%v", a.value)
}

// Int 获取整数值
func (a *ConfigAccessor) Int() int {
	if a.value == nil {
		return 0
	}

	switch v := a.value.(type) {
	case int:
		return v
	case float64:
		return int(v)
	case float32:
		return int(v)
	case int64:
		return int(v)
	case int32:
		return int(v)
	}

	return 0
}

// Bool 获取布尔值
func (a *ConfigAccessor) Bool() bool {
	if a.value == nil {
		return false
	}

	if b, ok := a.value.(bool); ok {
		return b
	}

	return false
}

// Float 获取浮点值
func (a *ConfigAccessor) Float() float64 {
	if a.value == nil {
		return 0
	}

	switch v := a.value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case int32:
		return float64(v)
	}

	return 0
}

// Map 获取映射值
func (a *ConfigAccessor) Map() map[string]interface{} {
	if a.value == nil {
		return nil
	}

	if m, ok := a.value.(map[string]interface{}); ok {
		return m
	}

	return nil
}

// Slice 获取切片值
func (a *ConfigAccessor) Slice() []interface{} {
	if a.value == nil {
		return nil
	}

	if s, ok := a.value.([]interface{}); ok {
		return s
	}

	return nil
}

// Exists 检查值是否存在
func (a *ConfigAccessor) Exists() bool {
	return a.value != nil
}

// GetAll 获取所有键值对
func (a *ConfigAccessor) GetAll() map[string]interface{} {
	if a.value == nil {
		return nil
	}

	if m, ok := a.value.(map[string]interface{}); ok {
		return m
	}

	return nil
}

// GetConfig 获取配置访问器
func (c *CustomConfig) GetConfig() *ConfigAccessor {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return NewConfigAccessor(c.config)
}

// Stop 停止配置监听
func (c *CustomConfig) Stop() {
	if c.watcher != nil {
		c.watcher.Stop()
	}
}

// NewMemoryConfig 创建内存配置实例
func NewMemoryConfig() *MemoryConfig {
	return &MemoryConfig{}
}

// Get 获取配置值
func (c *MemoryConfig) Get(key string) interface{} {
	value, exists := c.configs.Load(key)
	if !exists {
		return nil
	}
	return value
}

// GetString 获取字符串类型配置值
func (c *MemoryConfig) GetString(key string) string {
	value := c.Get(key)
	if value == nil {
		return ""
	}
	str, ok := value.(string)
	if !ok {
		return ""
	}
	return str
}

// GetInt 获取整数类型配置值
func (c *MemoryConfig) GetInt(key string) int {
	value := c.Get(key)
	if value == nil {
		return 0
	}
	intVal, ok := value.(int)
	if !ok {
		return 0
	}
	return intVal
}

// GetBool 获取布尔类型配置值
func (c *MemoryConfig) GetBool(key string) bool {
	value := c.Get(key)
	if value == nil {
		return false
	}
	boolVal, ok := value.(bool)
	if !ok {
		return false
	}
	return boolVal
}

// GetFloat64 获取浮点数类型配置值
func (c *MemoryConfig) GetFloat64(key string) float64 {
	value := c.Get(key)
	if value == nil {
		return 0
	}
	floatVal, ok := value.(float64)
	if !ok {
		return 0
	}
	return floatVal
}

// Exists 检查配置是否存在
func (c *MemoryConfig) Exists(key string) bool {
	_, exists := c.configs.Load(key)
	return exists
}

// GetAll 批量获取配置
func (c *MemoryConfig) GetAll() map[string]interface{} {
	configs := make(map[string]interface{})
	c.configs.Range(func(key, value interface{}) bool {
		configs[key.(string)] = value
		return true
	})
	return configs
}

// 全局配置实例
var (
	globalConfig Config
	customConfig *CustomConfig
	configMu     sync.RWMutex
)

// LoadAppConfig 加载应用配置
func LoadAppConfig(filePath string, appNames ...string) error {
	// 读取配置文件
	data, readErr := os.ReadFile(filepath.Clean(filePath))
	if readErr != nil {
		return readErr
	}

	// 解析配置
	var cfgMap map[string]*AppConfig
	if unmarshalErr := json.Unmarshal(data, &cfgMap); unmarshalErr != nil {
		return unmarshalErr
	}

	// 加载指定应用配置
	if len(appNames) > 0 {
		for _, appName := range appNames {
			if cfg, ok := cfgMap[appName]; ok {
				appConfigMap[appName] = cfg
			}
		}
	} else {
		// 如果没有指定应用名，加载所有应用配置
		for appName, cfg := range cfgMap {
			appConfigMap[appName] = cfg
		}
	}

	// 启动配置文件监听
	if _, exists := configFileWatchers[filePath]; !exists {
		watcher := NewConfigFileWatcher(filePath, 5*time.Second)
		configFileWatchers[filePath] = watcher
		watcher.Start("app")
	}

	return nil
}

// LoadDatabaseConfig 加载数据库配置
func LoadDatabaseConfig(filePath string) error {
	data, readErr := os.ReadFile(filepath.Clean(filePath))
	if readErr != nil {
		return readErr
	}

	var cfg DatabaseConfig
	if unmarshalErr := json.Unmarshal(data, &cfg); unmarshalErr != nil {
		return unmarshalErr
	}

	DbConfig = &cfg

	// 启动配置文件监听
	if _, exists := configFileWatchers[filePath]; !exists {
		watcher := NewConfigFileWatcher(filePath, 5*time.Second)
		configFileWatchers[filePath] = watcher
		watcher.Start("database")
	}

	return nil
}

// LoadCustomConfig 加载自定义配置
func LoadCustomConfig(filePath string) error {
	data, readErr := os.ReadFile(filepath.Clean(filePath))
	if readErr != nil {
		return readErr
	}

	var config map[string]interface{}
	if unmarshalErr := json.Unmarshal(data, &config); unmarshalErr != nil {
		return unmarshalErr
	}

	cfg := &CustomConfig{
		config:   config,
		filePath: filePath,
	}

	// 启动配置文件监听
	watcher := NewConfigFileWatcher(filePath, 5*time.Second)
	cfg.watcher = watcher

	// 这里需要为自定义配置添加监听逻辑
	// 为了简化，我们可以在现有的ConfigFileWatcher中添加对custom类型的支持
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		var lastModTime time.Time
		info, err := os.Stat(filePath)
		if err == nil {
			lastModTime = info.ModTime()
		}

		for range ticker.C {
			info, err := os.Stat(filePath)
			if err != nil {
				continue
			}

			// 检查文件是否被修改
			if info.ModTime().After(lastModTime) {
				lastModTime = info.ModTime()
				fmt.Printf("自定义配置文件已修改，重新加载: %s\n", filePath)

				// 重新加载配置
				data, readErr := os.ReadFile(filepath.Clean(filePath))
				if readErr != nil {
					fmt.Printf("读取自定义配置文件失败: %v\n", readErr)
					continue
				}

				var newConfig map[string]interface{}
				if unmarshalErr := json.Unmarshal(data, &newConfig); unmarshalErr != nil {
					fmt.Printf("解析自定义配置文件失败: %v\n", unmarshalErr)
					continue
				}

				// 更新配置
				cfg.mu.Lock()
				cfg.config = newConfig
				cfg.mu.Unlock()

				// 通知配置变更
				notifyConfigChange("custom", cfg)
			}
		}
	}()

	// 存储自定义配置
	configMu.Lock()
	customConfig = cfg
	configMu.Unlock()

	return nil
}

// StopConfigFileWatchers 停止所有配置文件监听器
func StopConfigFileWatchers() {
	for _, watcher := range configFileWatchers {
		watcher.Stop()
	}
	configFileWatchers = make(map[string]*ConfigFileWatcher)

	// 停止自定义配置的监听器
	configMu.RLock()
	if customConfig != nil && customConfig.watcher != nil {
		customConfig.watcher.Stop()
	}
	configMu.RUnlock()
}
