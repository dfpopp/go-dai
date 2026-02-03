package base

import "github.com/dfpopp/go-dai/logger"

type BaseService struct {
	log logger.Logger // 日志实例
}

// Init 初始化服务层（框架自动调用）
func (s *BaseService) Init() {
	s.log = logger.GetLogger()
}

// LogInfo 记录服务层信息日志
func (s *BaseService) LogInfo(content ...interface{}) {
	s.log.Info(content...)
}

// LogError 记录服务层错误日志
func (s *BaseService) LogError(content ...interface{}) {
	s.log.Error(content...)
}

// LogWarn 记录服务层错误日志
func (s *BaseService) LogWarn(content ...interface{}) {
	s.log.Warn(content...)
}
