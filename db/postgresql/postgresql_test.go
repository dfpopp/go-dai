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

// TestPostgresDb_SetUpsert 测试PG upsert三件套的校验逻辑与ON CONFLICT ... DO UPDATE段构建（纯断言，不连库）
func TestPostgresDb_SetUpsert(t *testing.T) {
	db := &PostgresDb{ConflictCols: []string{"uid"}, UpsertCols: []string{"update_time"}, UpsertIncCols: []string{"count"}}

	// 1. 单行Insert：覆盖列 "col" = ?、累加列 "col" = "col" + ?，增量值追加values末尾
	clause, values, err := db.buildUpsertFields(map[string]interface{}{"update_time": 111, "count": 3}, false, nil)
	if err != nil {
		t.Fatalf("single row build failed: %v", err)
	}
	want := "\"update_time\" = ?, \"count\" = \"count\" + ?"
	if clause != want {
		t.Errorf("single row clause = %q, want %q", clause, want)
	}
	if len(values) != 2 || values[0] != 111 || values[1] != 3 {
		t.Errorf("single row values = %v, want [111 3]", values)
	}

	// 2. 批量InsertAll：EXCLUDED."col"逐行取本行值，不追加参数
	clause, values, err = db.buildUpsertFields(map[string]interface{}{"update_time": 111, "count": 3}, true, nil)
	if err != nil {
		t.Fatalf("batch build failed: %v", err)
	}
	want = "\"update_time\" = EXCLUDED.\"update_time\", \"count\" = \"count\" + EXCLUDED.\"count\""
	if clause != want {
		t.Errorf("batch clause = %q, want %q", clause, want)
	}
	if len(values) != 0 {
		t.Errorf("batch values = %v, want empty", values)
	}

	// 3. ON CONFLICT整段构建：冲突目标列双引号包裹，DO UPDATE SET拼接正确
	seg, values, err := db.buildUpsertConflictClause(map[string]interface{}{"update_time": 111, "count": 3}, true, nil)
	if err != nil {
		t.Fatalf("conflict clause build failed: %v", err)
	}
	want = " ON CONFLICT (\"uid\") DO UPDATE SET \"update_time\" = EXCLUDED.\"update_time\", \"count\" = \"count\" + EXCLUDED.\"count\""
	if seg != want {
		t.Errorf("conflict clause = %q, want %q", seg, want)
	}
	if len(values) != 0 {
		t.Errorf("conflict clause batch values = %v, want empty", values)
	}

	// 4. 设置了更新列但缺冲突目标列：报错（PG的DO UPDATE必须显式指定ON CONFLICT目标，fail fast）
	dbNoConflict := &PostgresDb{UpsertIncCols: []string{"count"}}
	_, _, err = dbNoConflict.buildUpsertFields(map[string]interface{}{"count": 3}, false, nil)
	if err == nil {
		t.Error("buildUpsertFields should return error when ConflictCols is empty")
	}

	// 5. 同一列同时进两个列表：双向拦截（防静默双倍累加）
	db = &PostgresDb{}
	db = db.SetUpsert("count")
	db = db.SetUpsertInc("count")
	if db.Err == nil {
		t.Error("SetUpsertInc should return error when col already in UpsertCols")
	}
	db = &PostgresDb{}
	db = db.SetUpsertInc("count")
	db = db.SetUpsert("count")
	if db.Err == nil {
		t.Error("SetUpsert should return error when col already in UpsertIncCols")
	}

	// 6. clearData清空三个upsert字段
	db = &PostgresDb{}
	db = db.SetUpsertConflict("uid").SetUpsert("update_time").SetUpsertInc("count")
	if db.Err != nil {
		t.Fatalf("set methods should not return error: %v", db.Err)
	}
	db.clearData(false)
	if len(db.ConflictCols) != 0 || len(db.UpsertCols) != 0 || len(db.UpsertIncCols) != 0 {
		t.Errorf("clearData should reset upsert fields, got conflict=%v upsert=%v inc=%v", db.ConflictCols, db.UpsertCols, db.UpsertIncCols)
	}

	// 7. 只设置冲突目标不设置更新列：生成空段（退化为普通INSERT，冲突时数据库报duplicate key显性失败）
	seg, _, err = (&PostgresDb{ConflictCols: []string{"uid"}}).buildUpsertConflictClause(map[string]interface{}{"a": 1}, false, nil)
	if err != nil || seg != "" {
		t.Errorf("conflict clause without update cols = %q, err = %v, want empty with no error", seg, err)
	}
}
