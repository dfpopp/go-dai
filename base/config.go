package base

import (
	"sync"
)

// Config 配置管理接口
type Config interface {
	// 获取配置值

	Get(key string) interface{}
	GetString(key string) string
	GetInt(key string) int
	GetBool(key string) bool
	GetFloat64(key string) float64

	// 设置配置值

	Set(key string, value interface{})

	// 检查配置是否存在

	Exists(key string) bool

	// 移除配置

	Remove(key string)

	// 批量获取配置

	GetAll() map[string]interface{}
}

// MemoryConfig 内存配置实现
type MemoryConfig struct {
	configs sync.Map
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

// Set 设置配置值
func (c *MemoryConfig) Set(key string, value interface{}) {
	c.configs.Store(key, value)
}

// Exists 检查配置是否存在
func (c *MemoryConfig) Exists(key string) bool {
	_, exists := c.configs.Load(key)
	return exists
}

// Remove 移除配置
func (c *MemoryConfig) Remove(key string) {
	c.configs.Delete(key)
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
var globalConfig Config

// SetGlobalConfig 设置全局配置实例（框架层调用）
func SetGlobalConfig(config Config) {
	globalConfig = config
}

// GetGlobalConfig 获取全局配置实例
func GetGlobalConfig() Config {
	if globalConfig == nil {
		globalConfig = NewMemoryConfig()
	}
	return globalConfig
}
