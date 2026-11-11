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

// TestMysqlDb_SetUpsertInc 测试SetUpsertInc的校验逻辑与ON DUPLICATE KEY UPDATE累加段构建（纯断言，不连库）
func TestMysqlDb_SetUpsertInc(t *testing.T) {
	// 1. 单行Insert：累加列生成 `col` = `col` + ?，增量值追加values末尾
	db := &MysqlDb{UpsertIncCols: []string{"count"}}
	clause, values, err := db.buildUpsertFields(map[string]interface{}{"count": 3}, false, nil)
	if err != nil {
		t.Fatalf("single row build failed: %v", err)
	}
	if clause != "`count` = `count` + ?" {
		t.Errorf("single row clause = %q, want `count` = `count` + ?", clause)
	}
	if len(values) != 1 || values[0] != 3 {
		t.Errorf("single row values = %v, want [3]", values)
	}

	// 2. 批量InsertAll：累加列生成 `col` = `col` + VALUES(`col`)，不追加参数
	db = &MysqlDb{UpsertIncCols: []string{"count"}}
	clause, values, err = db.buildUpsertFields(map[string]interface{}{"count": 3}, true, nil)
	if err != nil {
		t.Fatalf("batch build failed: %v", err)
	}
	if clause != "`count` = `count` + VALUES(`count`)" {
		t.Errorf("batch clause = %q, want `count` = `count` + VALUES(`count`)", clause)
	}
	if len(values) != 0 {
		t.Errorf("batch values = %v, want empty", values)
	}

	// 3. 覆盖列与累加列共存：覆盖列在前、累加列在后，逗号连接
	db = &MysqlDb{UpsertCols: []string{"update_time"}, UpsertIncCols: []string{"count"}}
	clause, values, err = db.buildUpsertFields(map[string]interface{}{"update_time": 111, "count": 2}, true, nil)
	if err != nil {
		t.Fatalf("mixed build failed: %v", err)
	}
	want := "`update_time` = VALUES(`update_time`), `count` = `count` + VALUES(`count`)"
	if clause != want {
		t.Errorf("mixed clause = %q, want %q", clause, want)
	}
	if len(values) != 0 {
		t.Errorf("mixed batch values = %v, want empty", values)
	}

	// 4. 同一列同时进两个列表：双向拦截（防静默双倍累加）
	db = &MysqlDb{}
	db = db.SetUpsert("count")
	db = db.SetUpsertInc("count")
	if db.Err == nil {
		t.Error("SetUpsertInc should return error when col already in UpsertCols")
	}
	db = &MysqlDb{}
	db = db.SetUpsertInc("count")
	db = db.SetUpsert("count")
	if db.Err == nil {
		t.Error("SetUpsert should return error when col already in UpsertIncCols")
	}

	// 5. 列名带反引号、空格、分号：被isValidField拦下
	for _, bad := range []string{"cou`nt", "co unt", "count;drop"} {
		db = &MysqlDb{}
		db = db.SetUpsertInc(bad)
		if db.Err == nil {
			t.Errorf("SetUpsertInc should return error for invalid col %q", bad)
		}
	}

	// 6. 连续两次SetUpsertInc：clearData必须清空，不能把上一次的累加列带到下一个查询
	db = &MysqlDb{}
	db = db.SetUpsertInc("count")
	if db.Err != nil {
		t.Fatalf("SetUpsertInc should not return error: %v", db.Err)
	}
	db.clearData(false)
	if len(db.UpsertIncCols) != 0 {
		t.Errorf("clearData should reset UpsertIncCols, got %v", db.UpsertIncCols)
	}

	// 7. 累加列不在插入数据中：报错，不发SQL
	db = &MysqlDb{UpsertIncCols: []string{"count"}}
	_, _, err = db.buildUpsertFields(map[string]interface{}{"other": 1}, false, nil)
	if err == nil {
		t.Error("buildUpsertFields should return error when inc col missing in data")
	}
}

// TestIsValidFieldQuotedIdent 成对反引号标识符（保留字列）放行、其余形态仍拦截。
// 回归背景：校验器修复生效后，`window` 等MySQL 8.0保留字列被字符白名单误判注入，
// 导致waf CC规则查询全量失败（业务侧ccRuleCols必须写`window`）。
func TestIsValidFieldQuotedIdent(t *testing.T) {
	valid := []string{
		"`window`",
		// waf_cc_rule 真实字段串（线上全量报错的回归锚点）
		"id,site_id,name,target,conditions,logic,threshold,`window`,action,scope,redirect_url,sort,status,create_time,update_time",
		"`t`.`window`",
		"`order`.id",               // bidResourceServer 真实用例（order为保留字）
		"`name`,`module`,`value`",  // unionServer 真实用例
		"`select`,`order`,`group`", // 保留字列名不应被关键字规则误伤
		"`window` AS w",
		"COUNT(*)", "id", "*", "t.id AS x",
	}
	for _, s := range valid {
		if !isValidField(s) {
			t.Errorf("isValidField(%q) = false, want true", s)
		}
	}
	invalid := []string{
		"`window",            // 落单反引号
		"win`dow",            // 反引号嵌在标识符中间
		"`win`dow`",          // 三段式落单
		"`sel``ect`",         // MySQL双反引号转义，本实现不接受
		"`a` OR 1=1",         // 剥离后仍须命中万能密码规则
		"`a`; DROP TABLE t",  // 分号/DROP
		"`a`--",              // 注释符
		"`a` UNION SELECT b", // 注入关键字
		"a;b", "id FROM t", "",
	}
	for _, s := range invalid {
		if isValidField(s) {
			t.Errorf("isValidField(%q) = true, want false", s)
		}
	}
}
