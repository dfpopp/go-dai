package redisDb

import (
	"testing"
)

// TestGetRedisDB 测试获取Redis数据库连接
func TestGetRedisDB(t *testing.T) {
	// 测试不存在的数据库连接
	db, err := GetRedisDB("non_existent")
	if err == nil {
		t.Error("Non-existent DB should return error")
	}
	if db != nil {
		t.Error("Non-existent DB should return nil")
	}
}

// TestRedisDb_Methods 测试Redis数据库操作方法
func TestRedisDb_Methods(t *testing.T) {
	// 创建测试Redis连接（这里我们不实际连接Redis，只测试结构）
	testDb := &RedisDb{
		Db:    nil,
		DbPre: "test_",
	}
	
	// 测试RedisDb结构
	if testDb.Db != nil {
		t.Error("RedisDb.Db should be nil initially")
	}
	if testDb.DbPre != "test_" {
		t.Error("RedisDb.DbPre should be 'test_' initially")
	}
}
