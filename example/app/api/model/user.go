package model

import (
	"context"
	"encoding/json"
	"github.com/dfpopp/go-dai/db/mysql"
	"golang.org/x/crypto/bcrypt"
)

// User 用户模型
type User struct {
	ID       uint64 `json:"id"`
	Username string `json:"username"`
	Password string `json:"-"` // 不返回密码
	Nickname string `json:"nickname"`
}

var ctx = context.Background()

// UserModel 用户模型操作
type UserModel struct{}

// NewUserModel 创建模型实例
func NewUserModel() *UserModel {
	return &UserModel{}
}

// GetByUsername 根据用户名查询用户
func (m *UserModel) GetByUsername(username string) (*User, error) {
	db, err := mysql.GetMysqlDB("site")
	if err != nil {
		return nil, err
	}
	resultStr, err := db.SetTable("").SetWhere("username = ?", username).Find(ctx)
	if err != nil {
		return nil, err
	}
	var user User
	err = json.Unmarshal([]byte(resultStr), &user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// CheckPassword 验证密码
func (m *UserModel) CheckPassword(user *User, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	return err == nil
}
