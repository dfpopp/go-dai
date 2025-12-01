package mongoDb

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func CallSetWhere(mg *Db, ctx context.Context, coll string) {
	fmt.Println("\n--- SetWhere 调用实例 ---")
	// 设置查询条件：country=中国 且 type=tv
	result, err := mg.SetTable(coll).
		SetWhere(bson.D{
			{"country", "中国"},
			{"type", "tv"},
		}).
		FindAll(ctx).
		ToString()

	if err != nil {
		fmt.Printf("SetWhere查询失败: %v\n", err)
		return
	}
	fmt.Println(result)
}
func CallSetAgg(mg *Db, ctx context.Context, coll string) {
	fmt.Println("\n--- SetAggregatePipe 调用实例 ---")
	// 构建聚合管道：分组统计 + 排序
	pipe := mongo.Pipeline{
		bson.D{{"$group", bson.D{
			{"_id", "$country"},
			{"count", bson.M{"$sum": 1}},
		}}},
		bson.D{{"$sort", bson.D{{"count", -1}}}}, // 按数量降序
	}

	// 执行聚合查询
	result, err := mg.SetTable(coll).
		SetAgg(pipe). // 原方法名：SetAggregatePipe
		Aggregate(ctx).
		ToString()

	if err != nil {
		fmt.Printf("SetAggregatePipe聚合失败: %v\n", err)
		return
	}
	fmt.Println(result)
}
func CallSetSort(mg *Db, ctx context.Context, coll string) {
	fmt.Println("\n--- SetSort 调用实例 ---")
	// 设置排序：create_time降序，title升序
	result, err := mg.SetTable(coll).
		SetWhere(bson.D{{"country", "中国"}}).
		SetSort(bson.D{
			{"create_time", -1}, // -1=降序
			{"title", 1},        // 1=升序
		}).
		FindAll(ctx).
		ToString()

	if err != nil {
		fmt.Printf("SetSort查询失败: %v\n", err)
		return
	}
	fmt.Println(result)
}
func CallSetProjection(mg *Db, ctx context.Context, coll string) {
	fmt.Println("\n--- SetProjection 调用实例 ---")
	// 设置投影：返回title、type、country，不返回_id（_id默认返回，需显式置0）
	result, err := mg.SetTable(coll).
		SetWhere(bson.D{{"country", "中国"}}).
		SetProjection(bson.D{
			{"_id", 0},
			{"title", 1},
			{"type", 1},
			{"country", 1},
		}).
		FindAll(ctx).
		ToString()

	if err != nil {
		fmt.Printf("SetProjection查询失败: %v\n", err)
		return
	}
	fmt.Println(result)
}
func CallSetInsertOrdered(mg *Db, ctx context.Context, coll string) {
	fmt.Println("\n--- SetInsertOrdered 调用实例 ---")
	// 准备批量插入的数据
	docs := []interface{}{
		bson.M{"title": "测试电影1", "type": "movie", "country": "中国"},
		bson.M{"title": "测试电影2", "type": "movie", "country": "美国"},
		bson.M{"title": "测试电影3", "type": "movie", "country": "日本"},
	}

	// 设置为无序插入（ordered=false）
	insertedIDs, err := mg.SetTable(coll).
		SetInsertOrdered(false). // 无序插入：一条失败，其他继续
		InsertAll(ctx, docs)

	if err != nil {
		fmt.Printf("SetInsertOrdered插入失败: %v\n", err)
		return
	}
	fmt.Printf("批量插入的ID列表: %+v\n", insertedIDs)
}
func CallSetUpdateUpsert(mg *Db, ctx context.Context, coll string) {
	fmt.Println("\n--- SetUpdateUpsert 调用实例 ---")
	// 设置查询条件：title=未找到的电影
	// 设置更新内容：country=韩国，type=movie
	// 设置Upsert=true：不存在则插入
	updateCount, err := mg.SetTable(coll).
		SetWhere(bson.D{{"title", "未找到的电影"}}).
		SetUpdateUpsert(true). // 不存在则插入
		Update(ctx, bson.D{
			{"country", "韩国"},
			{"type", "movie"},
		})

	if err != nil {
		fmt.Printf("SetUpdateUpsert更新失败: %v\n", err)
		return
	}
	fmt.Printf("更新/插入的文档数: %d\n", updateCount)
}
func CallSetUpdateArrayFilters(mg *Db, ctx context.Context, coll string) {
	fmt.Println("\n--- SetUpdateArrayFilters 调用实例 ---")
	// 构建数组过滤条件：匹配actors数组中name=张三的元素
	arrayFilters := options.ArrayFilters{
		Filters: []interface{}{
			bson.D{{"actor.name", "张三"}}, // 过滤条件别名：actor
		},
	}

	// 执行更新：将匹配的数组元素的role改为"主角"
	updateCount, err := mg.SetTable(coll).
		SetWhere(bson.D{{"title", "测试电影1"}}).
		SetUpdateArrayFilters(arrayFilters). // 设置数组过滤
		Update(ctx, bson.D{
			{"actors.$[actor].role", "主角"}, // 使用别名actor匹配过滤条件
		})

	if err != nil {
		fmt.Printf("SetUpdateArrayFilters更新失败: %v\n", err)
		return
	}
	fmt.Printf("更新的文档数: %d\n", updateCount)
}
func CallSetDeleteHint(mg *Db, ctx context.Context, coll string) {
	fmt.Println("\n--- SetDeleteHint 调用实例 ---")
	// 设置删除条件：country=测试
	// 设置索引提示：使用country字段的索引（优化查询性能）
	delCount, err := mg.SetTable(coll).
		SetWhere(bson.D{{"country", "测试"}}).
		SetDeleteHint(bson.D{{"country", 1}}). // 指定使用country索引
		Delete(ctx)

	if err != nil {
		fmt.Printf("SetDeleteHint删除失败: %v\n", err)
		return
	}
	fmt.Printf("删除的文档数: %d\n", delCount)
}
func CallSetCollation(mg *Db, ctx context.Context, coll string) {
	fmt.Println("\n--- SetCollation 调用实例 ---")
	// 构建排序规则：中文拼音排序、大小写不敏感
	collation := &options.Collation{
		Locale:   "zh", // 中文语言环境
		Strength: 2,    // 忽略大小写，区分重音
	}

	// 执行查询：按title拼音升序排序
	result, err := mg.SetTable(coll).
		SetWhere(bson.D{{"title", bson.M{"$regex": "测试"}}}).
		SetSort(bson.D{{"title", 1}}).
		SetCollation(collation). // 设置字符串排序规则
		FindAll(ctx).
		ToString()

	if err != nil {
		fmt.Printf("SetCollation查询失败: %v\n", err)
		return
	}
	fmt.Printf("拼音排序后的标题列表: ")
	fmt.Println(result)
}
func GetProductCount() (string, error) {
	// 1. 获取MongoDB链式操作实例（替换为你的实例获取逻辑）
	var (
		sYear, eYear int                 // 起始年份、结束年份
		productList  []string            // 产品名称列表
		postData     map[string][]string // 前端传入的参数
	)
	mg, err := GetMongoDB("default")
	if err != nil {
		return "", fmt.Errorf("获取MongoDB实例失败: %v", err)
	}
	ctx := context.Background()
	// 2. 构建基础查询条件（原where的基础部分）
	baseWhere := bson.D{
		{"year", bson.M{"$gte": sYear, "$lte": eYear}},
		{"name", bson.M{"$in": productList}},
		{"markType", postData["markType"][0]},
	}

	// 3. 动态追加provinceId条件（原代码的if判断逻辑）
	if len(postData["provinceId"]) == 1 && postData["provinceId"][0] != "0" {
		baseWhere = append(baseWhere, bson.E{
			Key:   "provinceId",
			Value: postData["provinceId"][0],
		})
	}

	// 4. 构建聚合管道（与原pipeline完全一致）
	pipeline := mongo.Pipeline{
		bson.D{{"$match", baseWhere}}, // 匹配条件
		bson.D{{"$group", bson.D{ // 分组统计
			{"_id", "$name"},
			{"amount", bson.D{{"$sum", "$amount"}}},
			{"num", bson.D{{"$sum", "$num"}}},
		}}},
		bson.D{{"$sort", bson.D{{"_id", 1}}}}, // 按_id升序排序
		bson.D{{"$project", bson.M{ // 字段投影
			"product": "$_id",
			"amount":  "$amount",
			"num":     "$num",
		}}},
	}

	// 5. 链式调用执行聚合查询（核心）
	rows, err := mg.SetTable("product_count").SetAgg(pipeline).Aggregate(ctx).ToString()
	if err != nil {
		return "", fmt.Errorf("聚合查询失败: %v", err)
	}
	return rows, nil
}
