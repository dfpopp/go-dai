package base

type BaseService struct {
	BaseComponent
}

// Init 初始化服务层（框架自动调用）
func (s *BaseService) Init() {
	s.BaseComponent.Init()
}
