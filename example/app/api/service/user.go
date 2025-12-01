package service

import (
	"errors"
	"github.com/dfpopp/go-dai/example/app/api/model"
)

// UserService 用户服务
type UserService struct {
	userModel *model.UserModel
}

// NewUserService 创建服务实例
func NewUserService() *UserService {
	return &UserService{
		userModel: model.NewUserModel(),
	}
}

// Login 用户登录业务
func (s *UserService) Login(username, password string) (*model.User, error) {
	// 1. 查询用户
	user, err := s.userModel.GetByUsername(username)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("用户名不存在")
	}

	// 2. 验证密码
	if !s.userModel.CheckPassword(user, password) {
		return nil, errors.New("密码错误")
	}

	return user, nil
}
