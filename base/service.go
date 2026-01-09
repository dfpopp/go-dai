package base

import "github.com/dfpopp/go-dai/logger"

type BaseService struct {
	Log         logger.Logger // 日志实例
	ServiceName string        // 服务名称（应用层赋值）
}

// Init 初始化服务层（框架自动调用）
func (s *BaseService) Init() {
	s.Log = logger.GetLogger()
	if s.ServiceName == "" {
		s.ServiceName = "未知服务"
	}
	logger.Info(s.ServiceName, "服务初始化成功")
}

// LogInfo 记录服务层信息日志
func (s *BaseService) LogInfo(content ...interface{}) {
	logContent := append([]interface{}{"[" + s.ServiceName + "]"}, content...)
	s.Log.Info(logContent...)
}

// LogError 记录服务层错误日志
func (s *BaseService) LogError(content ...interface{}) {
	logContent := append([]interface{}{"[" + s.ServiceName + "]"}, content...)
	s.Log.Error(logContent...)
}
