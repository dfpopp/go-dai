package db

import (
	"testing"
)

// TestStartDb 测试启动数据库
func TestStartDb(t *testing.T) {
	// 测试启动空数据库列表
	StartDb([]string{})
	
	// 测试启动单个数据库
	StartDb([]string{"mysql"})
	
	// 测试启动多个数据库
	StartDb([]string{"mysql", "mongodb", "redis", "es"})
}
