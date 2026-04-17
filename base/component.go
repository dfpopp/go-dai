package base

import "github.com/dfpopp/go-dai/logger"

// BaseComponent 基础组件基类，提取公共功能
type BaseComponent struct {
	log logger.Logger // 日志实例
}

// Init 初始化组件（框架自动调用）
func (c *BaseComponent) Init() {
	c.log = logger.GetLogger()
}

// LogInfo 记录信息日志
func (c *BaseComponent) LogInfo(content ...interface{}) {
	if c == nil || c.log == nil {
		logger.Info(content...)
		return
	}
	c.log.Info(content...)
}

// LogError 记录错误日志
func (c *BaseComponent) LogError(content ...interface{}) {
	if c == nil || c.log == nil {
		logger.Error(content...)
		return
	}
	c.log.Error(content...)
}

// LogWarn 记录警告日志
func (c *BaseComponent) LogWarn(content ...interface{}) {
	if c == nil || c.log == nil {
		logger.Warn(content...)
		return
	}
	c.log.Warn(content...)
}
