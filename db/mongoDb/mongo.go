package mongoDb

import (
	"context"
	"errors"
	"fmt"
	"github.com/dfpopp/go-dai/config"
	"github.com/dfpopp/go-dai/function"
	"github.com/dfpopp/go-dai/logger"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"
)

// Db MongoDB操作类，支持链式调用
type Db struct {
	Client        *mongo.Client              // MongoDB客户端
	Db            *mongo.Database            // 当前数据库
	DbPre         string                     //表前缀
	TxSession     mongo.Session              // 事务会话
	Collection    string                     // 当前集合名
	Filter        bson.D                     // 查询条件
	AggregatePipe mongo.Pipeline             // 聚合管道
	FindOptions   *options.FindOptions       // 查询选项
	DeleteOptions *options.DeleteOptions     // 删除选项
	UpdateOptions *options.UpdateOptions     // 更新选项
	InsertOptions *options.InsertManyOptions // 批量插入选项
	Sort          bson.D                     // 排序条件
	Skip          int64                      // 跳过条数
	Limit         int64                      // 限制条数
	Projection    bson.D                     // 字段投影（只返回指定字段）
	Data          []map[string]interface{}   // 查询结果
	Err           error                      // 错误存储
}
type DbObj struct {
	Client *mongo.Client
	DbName string
	Pre    string
}

var multiClientPool sync.Map

// InitMongoDB 初始化MongoDB连接池，支持多配置
func InitMongoDB() {
	cfgMap := config.GetMongodbConfig()
	for dbKey, cfg := range cfgMap {
		client, err := connect(cfg)
		if err != nil {
			logger.Error(fmt.Sprintf("MongoDB连接初始化失败（%s）: %v", dbKey, err))
		} else {
			multiClientPool.Store(dbKey, DbObj{Client: client, DbName: cfg.Dbname, Pre: cfg.Pre})
		}
	}
	// 注册服务退出信号，触发 mongoDb 连接关闭（优雅退出）
	registerShutdownHook()
}

// connect 建立MongoDB连接
func connect(cfg config.MongodbConfig) (*mongo.Client, error) {
	// 默认配置
	if cfg.Host == "" {
		cfg.Host = "localhost"
	}
	if cfg.Port == "" {
		cfg.Port = "27017"
	}
	cpuNum := runtime.NumCPU()
	if cfg.MaxPoolSize == 0 {
		cfg.MaxPoolSize = uint64(cpuNum) * 3
	}
	if cfg.MinPoolSize == 0 {
		cfg.MinPoolSize = uint64(cpuNum)
	}
	if cfg.MaxConnIdleTime == 0 {
		cfg.MaxConnIdleTime = 300 //5分钟
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 5
	}
	// 构建连接URI
	var uri string
	if cfg.Pwd != "" {
		uri = fmt.Sprintf("mongodb://%s:%s@%s:%s/?connect=direct", cfg.User, cfg.Pwd, cfg.Host, cfg.Port)
	} else {
		uri = fmt.Sprintf("mongodb://%s:%s/?connect=direct", cfg.Host, cfg.Port)
	}
	// 配置客户端选项
	clientOpts := options.Client().ApplyURI(uri)
	clientOpts.SetCompressors([]string{"snappy"})
	clientOpts.SetMaxPoolSize(cfg.MaxPoolSize)
	clientOpts.SetMinPoolSize(cfg.MinPoolSize)
	clientOpts.SetMaxConnIdleTime(time.Duration(cfg.MaxConnIdleTime) * time.Second)
	clientOpts.SetConnectTimeout(time.Duration(cfg.Timeout) * time.Second)
	// 建立连接
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.Timeout)*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, err
	}
	// 校验连接
	if err = client.Ping(ctx, readpref.Primary()); err != nil {
		return nil, err
	}
	return client, nil
}

// GetMongoDB 获取MongoDB操作实例
func GetMongoDB(dbKey string) (*Db, error) {
	val, ok := multiClientPool.Load(dbKey)
	if !ok {
		return nil, fmt.Errorf("MongoDB连接池[%s]未初始化", dbKey)
	}
	dbObj, ok := val.(DbObj)
	if !ok {
		return nil, fmt.Errorf("MongoDB连接池[%s]类型错误", dbKey)
	}
	// 初始化操作选项
	findOpts := options.Find()
	deleteOpts := options.Delete()
	updateOpts := options.Update()
	insertOpts := options.InsertMany()
	return &Db{
		Client:        dbObj.Client,
		Db:            dbObj.Client.Database(dbObj.DbName),
		DbPre:         dbObj.Pre,
		TxSession:     nil,
		Collection:    "",
		Filter:        nil,
		AggregatePipe: nil,
		FindOptions:   findOpts,
		DeleteOptions: deleteOpts,
		UpdateOptions: updateOpts,
		InsertOptions: insertOpts,
		Sort:          nil,
		Skip:          0,
		Limit:         0,
		Projection:    nil,
		Data:          nil,
		Err:           nil,
	}, nil
}

// Begin 开启事务（需MongoDB副本集环境）
func (m *Db) Begin(ctx context.Context) *Db {
	if m.Err != nil {
		return m
	}
	if m.TxSession != nil {
		return m
	}
	if m.Client == nil {
		m.Err = errors.New("MongoDB客户端未初始化")
		return m
	}
	// 开始一个事务的会话
	sessionOpts := options.Session().SetDefaultReadPreference(readpref.Primary())
	session, err := m.Client.StartSession(sessionOpts)
	if err != nil {
		m.Err = fmt.Errorf("开启事务失败: %v", err)
		return m
	}
	// 启动事务
	if err = session.StartTransaction(); err != nil {
		session.EndSession(ctx)
		m.Err = fmt.Errorf("启动事务失败: %v", err)
		return m
	}
	m.TxSession = session
	return m
}

// Commit 提交事务
func (m *Db) Commit(ctx context.Context) *Db {
	defer m.clearData(true)
	if m.TxSession == nil {
		m.Err = errors.New("事务未开启")
		return m
	}
	if err := m.TxSession.CommitTransaction(ctx); err != nil {
		m.Err = fmt.Errorf("提交事务失败: %v", err)
		return m
	}
	m.TxSession.EndSession(ctx)
	return m
}

// Rollback 回滚事务
func (m *Db) Rollback(ctx context.Context) *Db {
	defer m.clearData(true)
	if m.TxSession == nil {
		m.Err = errors.New("事务未开启")
		return m
	}
	if err := m.TxSession.AbortTransaction(ctx); err != nil {
		m.Err = fmt.Errorf("回滚事务失败: %v", err)
	} else {
		m.Err = nil
	}
	m.TxSession.EndSession(ctx)
	return m
}

// getTxContext 获取绑定会话的上下文（核心修正：替代SetSession）
func (m *Db) getTxContext(ctx context.Context) context.Context {
	if m.TxSession != nil {
		// 将会话绑定到上下文，所有操作通过该上下文传递事务
		return mongo.NewSessionContext(ctx, m.TxSession)
	}
	return ctx
}

// SetTable 设置操作的集合名
func (m *Db) SetTable(col string) *Db {
	if m.Err != nil {
		return m
	}
	m.Collection = m.DbPre + col
	return m
}

// SetWhere 设置查询条件（bson.D）
func (m *Db) SetWhere(filter bson.D) *Db {
	if m.Err != nil {
		return m
	}
	m.Filter = filter
	return m
}

// SetAgg 设置聚合管道
func (m *Db) SetAgg(pipeline mongo.Pipeline) *Db {
	if m.Err != nil {
		return m
	}
	m.AggregatePipe = pipeline
	return m
}

// SetSort 设置排序条件（如bson.D{{"_id", -1}}）
func (m *Db) SetSort(sort bson.D) *Db {
	if m.Err != nil {
		return m
	}
	m.Sort = sort
	m.FindOptions.SetSort(sort)
	return m
}

// SetLimit 设置查询条数限制
func (m *Db) SetLimit(limit int64) *Db {
	if m.Err != nil {
		return m
	}
	m.Limit = limit
	m.FindOptions.SetLimit(limit)
	return m
}

// SetSkip 设置跳过条数
func (m *Db) SetSkip(skip int64) *Db {
	if m.Err != nil {
		return m
	}
	m.Skip = skip
	m.FindOptions.SetSkip(skip)
	return m
}

// SetProjection 设置字段投影（指定返回/排除的字段）
func (m *Db) SetProjection(proj bson.D) *Db {
	if m.Err != nil {
		return m
	}
	m.Projection = proj
	m.FindOptions.SetProjection(proj)
	return m
}

// SetInsertOrdered 设置批量插入是否有序（默认true：有序插入，一条失败则后续都失败；false：无序插入，部分失败不影响）
func (m *Db) SetInsertOrdered(ordered bool) *Db {
	if m.Err != nil {
		return m
	}
	m.InsertOptions.SetOrdered(ordered)
	return m
}

// SetUpdateUpsert 设置更新的Upsert（不存在则插入，默认false）
func (m *Db) SetUpdateUpsert(upsert bool) *Db {
	if m.Err != nil {
		return m
	}
	m.UpdateOptions.SetUpsert(upsert)
	return m
}

// SetUpdateArrayFilters 设置更新的数组过滤条件（用于更新数组中的特定元素）
func (m *Db) SetUpdateArrayFilters(filters options.ArrayFilters) *Db {
	if m.Err != nil {
		return m
	}
	m.UpdateOptions.SetArrayFilters(filters)
	return m
}

// SetDeleteHint 设置删除的索引提示（指定使用哪个索引查询，优化性能）
func (m *Db) SetDeleteHint(hint interface{}) *Db {
	if m.Err != nil {
		return m
	}
	m.DeleteOptions.SetHint(hint)
	return m
}

// SetCollation 设置排序规则（适用于查询/更新/删除/插入的字符排序）
func (m *Db) SetCollation(collation *options.Collation) *Db {
	if m.Err != nil {
		return m
	}
	m.FindOptions.SetCollation(collation)
	m.UpdateOptions.SetCollation(collation)
	m.DeleteOptions.SetCollation(collation)
	return m
}

// FindAll 执行查询，返回多条结果
func (m *Db) FindAll(ctx context.Context) *Db {
	if m.Err != nil {
		return m
	}
	if m.Collection == "" {
		m.Err = errors.New("未指定集合名")
		return m
	}
	// 获取集合
	coll := m.Db.Collection(m.Collection)
	// 获取绑定事务的上下文
	txCtx := m.getTxContext(ctx)
	// 执行查询
	cursor, err := coll.Find(txCtx, m.Filter, m.FindOptions)
	if err != nil {
		m.Err = fmt.Errorf("查询失败: %v", err)
		return m
	}
	if cursor == nil {
		if m.Collection == "member_point" {
			fmt.Printf("没有数据")
		}
		return m
	}
	if m.Collection == "member_point" {
		fmt.Printf(">>>okkk")
	}
	defer func(cursor *mongo.Cursor, ctx context.Context) {
		closeErr := cursor.Close(ctx)
		if closeErr != nil {
			logger.Error("mongoDb 关闭结果集失败: %v", closeErr)
		}
	}(cursor, txCtx)
	if m.Collection == "member_point" {
		fmt.Printf("okkk")
	}
	// 解析结果
	var result []map[string]interface{}
	for cursor.Next(txCtx) {
		var doc map[string]interface{}
		if err := cursor.Decode(&doc); err != nil {
			m.Err = fmt.Errorf("解析文档失败: %v", err)
			return m
		}
		if m.Collection == "member_point" {
			fmt.Printf("<<<<111")
		}
		result = append(result, doc)
	}
	if m.Collection == "member_point" {
		fmt.Printf("<<<<okkk")
	}
	// 检查游标错误
	if err := cursor.Err(); err != nil {
		m.Err = fmt.Errorf("游标遍历失败: %v", err)
		return m
	}
	if m.Collection == "member_point" {
		fmt.Printf("<<<<2222")
	}
	m.Data = result
	if m.Collection == "member_point" {
		fmt.Printf("<<<<333")
	}
	return m
}

// FindCount 统计符合条件的文档数
func (m *Db) FindCount(ctx context.Context) (int64, error) {
	defer m.clearData(false)
	if m.Err != nil {
		return 0, m.Err
	}
	if m.Collection == "" {
		return 0, errors.New("未指定集合名")
	}
	coll := m.Db.Collection(m.Collection)
	txCtx := m.getTxContext(ctx)
	count, err := coll.CountDocuments(txCtx, m.Filter)
	if err != nil {
		m.Err = fmt.Errorf("计数失败: %v", err)
		return 0, m.Err
	}
	return count, nil
}

// Find 执行查询，返回单条结果
func (m *Db) Find(ctx context.Context) (string, error) {
	fmt.Println("已进入find")
	if m.Err != nil {
		return "", m.Err
	}
	if m.Db == nil {
		return "", errors.New("数据库资源不存在")
	}
	if m.Collection == "" {
		return "", errors.New("未指定集合名")
	}
	fmt.Println(function.Json_encode(m))
	if m.Collection == "member_point" {
		fmt.Printf("<<<<0000")
	}
	defer m.clearData(false)
	m.SetLimit(1)
	if m.Collection == "member_point" {
		fmt.Printf("<<<<0000")
	}
	m.FindAll(ctx)
	if m.Collection == "member_point" {
		fmt.Printf("<<<<66666")
	}
	if m.Err != nil {
		return "", m.Err
	}
	if m.Collection == "member_point" {
		fmt.Printf("<<<<777")
	}
	if len(m.Data) > 0 {
		if m.Collection == "member_point" {
			fmt.Printf("<<<<88888")
		}
		return function.Json_encode(m.Data[0]), nil
	}
	if m.Collection == "member_point" {
		fmt.Printf("<<<<9999")
	}
	return "", nil
}

// Aggregate 执行聚合查询
func (m *Db) Aggregate(ctx context.Context) *Db {
	defer m.clearData(false)
	if m.Err != nil {
		return m
	}
	if m.Collection == "" {
		m.Err = errors.New("未指定集合名")
		return m
	}
	if len(m.AggregatePipe) == 0 {
		m.Err = errors.New("聚合管道不能为空")
		return m
	}
	coll := m.Db.Collection(m.Collection)
	txCtx := m.getTxContext(ctx)
	cursor, err := coll.Aggregate(txCtx, m.AggregatePipe)
	if err != nil {
		m.Err = fmt.Errorf("聚合查询失败: %v", err)
		return m
	}
	if cursor == nil {
		return m
	}
	defer func(cursor *mongo.Cursor, ctx context.Context) {
		closeErr := cursor.Close(ctx)
		if closeErr != nil {
			logger.Error("mongoDb 关闭结果集失败: %v", closeErr)
		}
	}(cursor, txCtx)
	var result []map[string]interface{}
	for cursor.Next(txCtx) {
		var doc map[string]interface{}
		if err := cursor.Decode(&doc); err != nil {
			m.Err = fmt.Errorf("解析聚合结果失败: %v", err)
			return m
		}
		result = append(result, doc)
	}

	if err := cursor.Err(); err != nil {
		m.Err = fmt.Errorf("聚合游标遍历失败: %v", err)
		return m
	}
	m.Data = result
	return m
}

// Insert 插入单条文档
func (m *Db) Insert(ctx context.Context, doc interface{}) (primitive.ObjectID, error) {
	defer m.clearData(false)
	if m.Err != nil {
		return primitive.NilObjectID, m.Err
	}
	if m.Collection == "" {
		return primitive.NilObjectID, errors.New("未指定集合名")
	}
	if doc == nil {
		return primitive.NilObjectID, errors.New("插入文档不能为空")
	}
	coll := m.Db.Collection(m.Collection)
	txCtx := m.getTxContext(ctx)
	res, err := coll.InsertOne(txCtx, doc)
	if err != nil {
		m.Err = fmt.Errorf("插入失败: %v", err)
		return primitive.NilObjectID, m.Err
	}
	// 转换为ObjectID
	oid, ok := res.InsertedID.(primitive.ObjectID)
	if !ok {
		m.Err = errors.New("插入ID不是ObjectID类型")
		return primitive.NilObjectID, m.Err
	}
	return oid, nil
}

// InsertAll 批量插入文档
func (m *Db) InsertAll(ctx context.Context, docs []interface{}) ([]interface{}, error) {
	defer m.clearData(false)
	if m.Err != nil {
		return nil, m.Err
	}
	if m.Collection == "" {
		return nil, errors.New("未指定集合名")
	}
	if len(docs) == 0 {
		return nil, errors.New("批量插入文档不能为空")
	}

	coll := m.Db.Collection(m.Collection)
	txCtx := m.getTxContext(ctx)

	res, err := coll.InsertMany(txCtx, docs, m.InsertOptions)
	if err != nil {
		m.Err = fmt.Errorf("批量插入失败: %v", err)
		return nil, m.Err
	}
	return res.InsertedIDs, nil
}

// Update 更新文档（默认更新多条）
func (m *Db) Update(ctx context.Context, update bson.D) (int64, error) {
	defer m.clearData(false)
	if m.Err != nil {
		return 0, m.Err
	}
	if m.Collection == "" {
		return 0, errors.New("未指定集合名")
	}
	if len(update) == 0 {
		return 0, errors.New("更新条件不能为空")
	}
	if len(m.Filter) == 0 {
		return 0, errors.New("查询条件不能为空（防止全表更新）")
	}

	coll := m.Db.Collection(m.Collection)
	txCtx := m.getTxContext(ctx)
	// 构造更新操作（$set）
	res, err := coll.UpdateMany(txCtx, m.Filter, update, m.UpdateOptions)
	if err != nil {
		m.Err = fmt.Errorf("更新失败: %v", err)
		return 0, m.Err
	}
	return res.ModifiedCount, nil
}

// UpdateOne 更新单条文档
func (m *Db) UpdateOne(ctx context.Context, update bson.D) (int64, error) {
	defer m.clearData(false)
	if m.Err != nil {
		return 0, m.Err
	}
	if m.Collection == "" {
		return 0, errors.New("未指定集合名")
	}
	if len(update) == 0 {
		return 0, errors.New("更新条件不能为空")
	}
	coll := m.Db.Collection(m.Collection)
	txCtx := m.getTxContext(ctx)
	res, err := coll.UpdateOne(txCtx, m.Filter, update, m.UpdateOptions)
	if err != nil {
		m.Err = fmt.Errorf("更新单条失败: %v", err)
		return 0, m.Err
	}
	return res.ModifiedCount, nil
}

// Delete 删除文档（默认删除多条）
func (m *Db) Delete(ctx context.Context) (int64, error) {
	defer m.clearData(false)
	if m.Err != nil {
		return 0, m.Err
	}
	if m.Collection == "" {
		return 0, errors.New("未指定集合名")
	}
	if len(m.Filter) == 0 {
		return 0, errors.New("查询条件不能为空（防止全表删除）")
	}

	coll := m.Db.Collection(m.Collection)
	txCtx := m.getTxContext(ctx)

	// 核心修正：删除操作通过事务上下文传递会话，而非SetSession
	res, err := coll.DeleteMany(txCtx, m.Filter, m.DeleteOptions)
	if err != nil {
		m.Err = fmt.Errorf("删除失败: %v", err)
		return 0, m.Err
	}
	return res.DeletedCount, nil
}

// DeleteOne 删除单条文档
func (m *Db) DeleteOne(ctx context.Context) (int64, error) {
	defer m.clearData(false)
	if m.Err != nil {
		return 0, m.Err
	}
	if m.Collection == "" {
		return 0, errors.New("未指定集合名")
	}
	if len(m.Filter) == 0 {
		return 0, errors.New("查询条件不能为空")
	}

	coll := m.Db.Collection(m.Collection)
	txCtx := m.getTxContext(ctx)

	res, err := coll.DeleteOne(txCtx, m.Filter, m.DeleteOptions)
	if err != nil {
		m.Err = fmt.Errorf("删除单条失败: %v", err)
		return 0, m.Err
	}
	return res.DeletedCount, nil
}

// ToString 返回结果的字符串形式（可结合JSON序列化）和错误
func (m *Db) ToString() (string, error) {
	defer m.clearData(false)
	if m.Err != nil {
		return "", m.Err
	}
	if len(m.Data) == 0 {
		return "", nil
	}
	return function.Json_encode(m.Data), nil
}

// clearData 清理查询数据和临时配置
func (m *Db) clearData(isClearTx bool) {
	m.Collection = ""
	m.Filter = nil
	m.AggregatePipe = nil
	m.FindOptions = nil
	m.DeleteOptions = nil
	m.UpdateOptions = nil
	m.InsertOptions = nil
	m.Sort = nil
	m.Limit = 0
	m.Skip = 0
	m.Projection = nil
	m.Data = nil
	m.Err = nil
	if isClearTx {
		m.TxSession = nil
	}
}

// 注册服务退出钩子（监听信号，自动关闭 mongoDb 连接）
func registerShutdownHook() {
	sigCh := make(chan os.Signal, 1)
	// 监听常见的退出信号：Ctrl+C、kill 命令
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		<-sigCh // 等待信号
		fmt.Println("\n收到退出信号，开始关闭 Mysql 连接...")
		if err := CloseMongoDb(); err != nil {
			fmt.Printf("Mysql 连接关闭失败: %v\n", err)
		} else {
			fmt.Println("所有 Mysql 连接已关闭")
		}
		os.Exit(0)
	}()
}

// CloseMongoDb 关闭所有 mongoDb 连接（供外部调用，如服务停止时）
func CloseMongoDb() error {
	var err error
	multiClientPool.Range(func(key, value interface{}) bool {
		dbObj, ok := value.(DbObj)
		if !ok {
			err = fmt.Errorf("无效的 mongoDb 客户端对象（key: %v）", key)
			return false // 终止遍历
		}
		// 关闭客户端（会释放连接池中的所有连接）
		disconnectCtx := context.Background()
		if closeErr := dbObj.Client.Disconnect(disconnectCtx); closeErr != nil {
			// 不建议 panic，用日志记录更优雅（避免程序异常退出）
			err = fmt.Errorf("关闭 mongoDb 连接失败（dbKey: %v）: %w", key, closeErr)
			// 继续遍历，尝试关闭其他连接
			return true
		}
		fmt.Printf("mongoDb 连接已关闭（dbKey: %v）\n", key)
		return true
	})
	return err
}
