package mysql

import (
	"context"
	"testing"
)

// TestGetMysqlDB 测试获取MySQL数据库连接
func TestGetMysqlDB(t *testing.T) {
	// 测试不存在的数据库连接
	db, err := GetMysqlDB("non_existent")
	if err == nil {
		t.Error("Non-existent DB should return error")
	}
	if db != nil {
		t.Error("Non-existent DB should return nil")
	}
}

// TestMysqlDb_Methods 测试MySQL数据库操作方法
func TestMysqlDb_Methods(t *testing.T) {
	// 创建测试数据库连接（这里我们不实际连接数据库，只测试方法调用）
	testDb := &MysqlDb{
		Db: nil,
		Tx: nil,
	}
	
	// 测试链式调用方法
	testDb = testDb.SetTable("test_table")
	testDb = testDb.SetAlias("t")
	testDb = testDb.SetField("id, name")
	testDb = testDb.SetWhere("id = ?", 1)
	testDb = testDb.SetWhereOr(map[string]interface{}{"name": "test"})
	testDb = testDb.SetWhereIn("id", 1, 2, 3)
	testDb = testDb.SetOrder("id DESC")
	testDb = testDb.SetGroup("name")
	testDb = testDb.SetJoin("test_join tj", "t.id = tj.test_id", "LEFT")
	testDb = testDb.SetLimit(0, 10)
	
	// 测试事务方法
	err := testDb.ToBegin()
	if err == nil {
		t.Error("ToBegin should return error when Db is nil")
	}
	
	err = testDb.Rollback()
	if err == nil {
		t.Error("Rollback should return error when Tx is nil")
	}
	
	err = testDb.Commit()
	if err == nil {
		t.Error("Commit should return error when Tx is nil")
	}
	
	// 测试查询方法
	ctx := context.Background()
	testDb.FindAll(ctx)
	if testDb.Err == nil {
		t.Error("FindAll should return error when Db is nil")
	}
	
	_, err = testDb.FindCount(ctx)
	if err == nil {
		t.Error("FindCount should return error when Db is nil")
	}
	
	_, err = testDb.Find(ctx)
	if err == nil {
		t.Error("Find should return error when Db is nil")
	}
	
	// 测试插入方法
	_, err = testDb.Insert(ctx, map[string]interface{}{"name": "test"})
	if err == nil {
		t.Error("Insert should return error when Db is nil")
	}
	
	_, err = testDb.InsertAll(ctx, []map[string]interface{}{{"name": "test1"}, {"name": "test2"}})
	if err == nil {
		t.Error("InsertAll should return error when Db is nil")
	}
	
	// 测试更新方法
	_, err = testDb.Update(ctx, map[string]interface{}{"name": "updated"})
	if err == nil {
		t.Error("Update should return error when Db is nil")
	}
	
	_, err = testDb.UpdateBySet(ctx, "name = ?", "updated")
	if err == nil {
		t.Error("UpdateBySet should return error when Db is nil")
	}
	
	_, err = testDb.SetInc(ctx, "count = count + ?", 1)
	if err == nil {
		t.Error("SetInc should return error when Db is nil")
	}
	
	// 测试删除方法
	_, err = testDb.Delete(ctx)
	if err == nil {
		t.Error("Delete should return error when Db is nil")
	}
	
	// 测试Exec方法
	_, err = testDb.Exec(ctx, "SELECT 1")
	if err == nil {
		t.Error("Exec should return error when Db is nil")
	}
	
	// 测试ToString方法
	_, err = testDb.ToString()
	if err == nil {
		t.Error("ToString should return error when Err is set")
	}
}

// TestBatchUpdateCaseWhen 测试批量更新SQL生成
func TestBatchUpdateCaseWhen(t *testing.T) {
	// 测试空数据
	sql, err := BatchUpdateCaseWhen("test_table", "id", []string{"name"}, nil)
	if err == nil {
		t.Error("BatchUpdateCaseWhen should return error when data is nil")
	}
	if sql != "" {
		t.Error("BatchUpdateCaseWhen should return empty string when data is nil")
	}
	
	// 测试空字段
	sql, err = BatchUpdateCaseWhen("test_table", "id", []string{}, map[string]interface{}{})
	if err == nil {
		t.Error("BatchUpdateCaseWhen should return error when fields is empty")
	}
	if sql != "" {
		t.Error("BatchUpdateCaseWhen should return empty string when fields is empty")
	}
	
	// 测试正常数据
	testData := map[string]interface{}{
		"1": map[string]interface{}{"name": "test1"},
		"2": map[string]interface{}{"name": "test2"},
	}
	
	sql, err = BatchUpdateCaseWhen("test_table", "id", []string{"name"}, testData)
	if err != nil {
		t.Errorf("BatchUpdateCaseWhen should not return error: %v", err)
	}
	if sql == "" {
		t.Error("BatchUpdateCaseWhen should return non-empty string when data is provided")
	}
}
