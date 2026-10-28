package postgresql

import (
	"context"
	"strings"
	"testing"
)

// TestGetPostgresDB 测试获取PostgreSQL数据库连接
func TestGetPostgresDB(t *testing.T) {
	// 测试不存在的数据库连接
	db, err := GetPostgresDB("non_existent")
	if err == nil {
		t.Error("Non-existent DB should return error")
	}
	if db != nil {
		t.Error("Non-existent DB should return nil")
	}
}

// TestPostgresDb_Methods 测试PostgreSQL数据库操作方法
func TestPostgresDb_Methods(t *testing.T) {
	// 创建测试数据库连接（这里我们不实际连接数据库，只测试方法调用）
	testDb := &PostgresDb{
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
	// PostgreSQL版本应使用双引号标识符
	if !strings.Contains(sql, `"name"`) {
		t.Errorf("BatchUpdateCaseWhen should use double-quoted identifiers, got: %s", sql)
	}
}

// TestConvertPlaceholders 测试占位符转换（? → $n）
func TestConvertPlaceholders(t *testing.T) {
	cases := []struct {
		input  string
		expect string
	}{
		{"SELECT * FROM users", "SELECT * FROM users"},
		{"id = ?", "id = $1"},
		{"id = ? AND name = ? AND status = ?", "id = $1 AND name = $2 AND status = $3"},
		{"IN (?, ?, ?)", "IN ($1, $2, $3)"},
	}
	for _, c := range cases {
		got := convertPlaceholders(c.input)
		if got != c.expect {
			t.Errorf("convertPlaceholders(%q) = %q, expect %q", c.input, got, c.expect)
		}
	}
}

// TestBuildLimit 测试LIMIT语法转换（skip,num → LIMIT n OFFSET m）
func TestBuildLimit(t *testing.T) {
	db := &PostgresDb{}
	if got := db.buildLimit(); got != " LIMIT 500" {
		t.Errorf("default buildLimit = %q, expect ` LIMIT 500`", got)
	}

	db.Limit = "10"
	if got := db.buildLimit(); got != " LIMIT 10" {
		t.Errorf("buildLimit(10) = %q, expect ` LIMIT 10`", got)
	}

	db.Limit = "20,10"
	if got := db.buildLimit(); got != " LIMIT 10 OFFSET 20" {
		t.Errorf("buildLimit(20,10) = %q, expect ` LIMIT 10 OFFSET 20`", got)
	}
}

// TestQuoteTable 测试表名引用（支持schema.table分段引用）
func TestQuoteTable(t *testing.T) {
	if got := quoteTable("users"); got != `"users"` {
		t.Errorf("quoteTable(users) = %q, expect `\"users\"`", got)
	}
	if got := quoteTable("public.users"); got != `"public"."users"` {
		t.Errorf("quoteTable(public.users) = %q, expect `\"public\".\"users\"`", got)
	}
}

// TestDeleteWithLimit 测试PostgreSQL不支持DELETE LIMIT
func TestDeleteWithLimit(t *testing.T) {
	// Db为nil时Delete先返回连接错误；此处仅验证Limit错误提示逻辑在连接正常时才触发
	// 由于无法在单测中建立真实连接，此处仅验证空Limit的Delete返回连接池错误
	db := &PostgresDb{Db: nil}
	ctx := context.Background()
	_, err := db.Delete(ctx)
	if err == nil {
		t.Error("Delete should return error when Db is nil")
	}
}
