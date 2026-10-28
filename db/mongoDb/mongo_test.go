package mongoDb

import (
	"context"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// TestGetMongoDB 测试获取MongoDB连接
func TestGetMongoDB(t *testing.T) {
	// 测试不存在的数据库连接
	db, err := GetMongoDB("non_existent")
	if err == nil {
		t.Error("Non-existent DB should return error")
	}
	if db != nil {
		t.Error("Non-existent DB should return nil")
	}
}

// TestDb_Methods 测试MongoDB数据库操作方法
func TestDb_Methods(t *testing.T) {
	// 创建测试数据库连接（这里我们不实际连接数据库，只测试方法调用）
	testDb := &Db{
		Client:           nil,
		Db:               nil,
		DbPre:            "test_",
		TxSession:        nil,
		Collection:       "",
		Filter:           nil,
		AggregatePipe:    nil,
		FindOptions:      options.Find(),
		DeleteOptions:    options.DeleteMany(),
		DeleteOneOptions: options.DeleteOne(),
		UpdateOptions:    options.UpdateMany(),
		UpdateOneOptions: options.UpdateOne(),
		InsertOptions:    options.InsertMany(),
		Sort:             nil,
		Skip:             0,
		Limit:            0,
		Projection:       nil,
		Data:             nil,
		Err:              nil,
	}

	// 测试设置方法（只测试不会修改Client/Db的方法）
	testDb = testDb.SetTable("test_collection")
	testDb = testDb.SetWhere(bson.D{{Key: "_id", Value: "test"}})
	testDb = testDb.SetAgg(nil)
	testDb = testDb.SetSort(bson.D{{Key: "_id", Value: -1}})
	testDb = testDb.SetLimit(10)
	testDb = testDb.SetSkip(0)
	testDb = testDb.SetProjection(bson.D{{Key: "_id", Value: 1}, {Key: "name", Value: 1}})
	testDb = testDb.SetInsertOrdered(true)
	testDb = testDb.SetUpdateUpsert(true)
	testDb = testDb.SetUpdateArrayFilters([]interface{}{})
	testDb = testDb.SetDeleteHint("_id_1")

	// 测试事务方法
	ctx := context.Background()
	err := testDb.Begin(ctx)
	if err == nil {
		t.Error("Begin should return error when Client is nil")
	}

	err = testDb.Commit(ctx)
	if err == nil {
		t.Error("Commit should return error when TxSession is nil")
	}

	err = testDb.Rollback(ctx)
	if err == nil {
		t.Error("Rollback should return error when TxSession is nil")
	}

	// 测试查询方法
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

	testDb.Aggregate(ctx)
	if testDb.Err == nil {
		t.Error("Aggregate should return error when Db is nil")
	}

	// 测试插入方法
	_, err = testDb.Insert(ctx, map[string]interface{}{"name": "test"})
	if err == nil {
		t.Error("Insert should return error when Db is nil")
	}

	_, err = testDb.InsertAll(ctx, []interface{}{map[string]interface{}{"name": "test1"}, map[string]interface{}{"name": "test2"}})
	if err == nil {
		t.Error("InsertAll should return error when Db is nil")
	}

	// 测试更新方法
	_, err = testDb.Update(ctx, bson.D{{Key: "$set", Value: bson.D{{Key: "name", Value: "updated"}}}})
	if err == nil {
		t.Error("Update should return error when Db is nil")
	}

	_, err = testDb.UpdateOne(ctx, bson.D{{Key: "$set", Value: bson.D{{Key: "name", Value: "updated"}}}})
	if err == nil {
		t.Error("UpdateOne should return error when Db is nil")
	}

	// 测试删除方法
	_, err = testDb.Delete(ctx)
	if err == nil {
		t.Error("Delete should return error when Db is nil")
	}

	_, err = testDb.DeleteOne(ctx)
	if err == nil {
		t.Error("DeleteOne should return error when Db is nil")
	}

	// 测试ToString方法
	_, err = testDb.ToString()
	if err == nil {
		t.Error("ToString should return error when Err is set")
	}
}
