package redisDb

import (
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/dfpopp/go-dai/config"
	"github.com/dfpopp/go-dai/logger"
	"github.com/go-redis/redis"
)

// 该文件为mysql基本操作类，支持链式操作，在执行findAll()后必须调用ToString()才能返回想要的结果和错误信息
// 全局多数据库连接池
var multiDBPool sync.Map

type RedisDb struct {
	Db    *redis.Client // 复用全局数据库连接池
	DbPre string        //表前缀
}
type DbObj struct {
	Db  *redis.Client // 复用全局数据库连接池
	Pre string
}

// InitRedis 初始化Redis连接池
func InitRedis() {
	initRedisConnections()

	// 注册配置变更钩子，当数据库配置变更时重新初始化连接
	config.RegisterConfigChangeHook(func(configType string, config interface{}) error {
		if configType == "database" {
			logger.Info("数据库配置变更，重新初始化Redis连接...")
			return reinitRedisConnections()
		}
		return nil
	})
}

// initRedisConnections 初始化Redis连接
func initRedisConnections() {
	cfgMap := config.GetRedisConfig()
	for dbKey, cfg := range cfgMap {
		createRedisConnection(dbKey, cfg)
	}
}

// createRedisConnection 创建Redis连接
func createRedisConnection(dbKey string, cfg config.RedisConfig) {
	cpuNum := runtime.NumCPU()
	if cfg.MinIdleConns == 0 {
		cfg.MinIdleConns = cpuNum
	}
	if cfg.PoolSize == 0 {
		cfg.PoolSize = cpuNum * 3
	}
	if cfg.MaxConnLifetime == 0 {
		cfg.MaxConnLifetime = 1800
	}
	if cfg.IdleTimeout == 0 {
		cfg.IdleTimeout = 300
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 10
	}
	if cfg.ReadTimeout == 0 {
		cfg.ReadTimeout = 5
	}
	if cfg.WriteTimeout == 0 {
		cfg.WriteTimeout = 5
	}
	if cfg.MaxRetries == 0 {
		cfg.MaxRetries = 2
	}
	if cfg.MinRetryBackoff == 0 {
		cfg.MinRetryBackoff = 100 //单位毫秒
	}
	if cfg.MaxRetryBackoff == 0 {
		cfg.MaxRetryBackoff = 1 //单位秒
	}
	// 端口默认值（避免配置缺失导致 Addr 格式错误）
	if cfg.Port == "" {
		cfg.Port = "6379"
	}
	// 构建 Redis 客户端配置
	redisOpts := &redis.Options{
		Network:      "tcp",
		Addr:         fmt.Sprintf("%s:%s", cfg.Host, cfg.Port), // 格式化 Addr，避免空端口
		Password:     cfg.Pwd,                                  // 空密码直接传入，适配无密码环境
		DB:           0,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		MaxConnAge:   time.Duration(cfg.MaxConnLifetime) * time.Second,
		IdleTimeout:  time.Duration(cfg.IdleTimeout) * time.Second, // 优化：连接池闲置连接超时，自动关闭过期连接（避免资源浪费）
		// v7 版本的连接超时字段名是 DialTimeout（v8 是 Timeout，注意区别！）
		DialTimeout:     time.Duration(cfg.Timeout) * time.Second,              // v7 用 DialTimeout 表示连接超时
		ReadTimeout:     time.Duration(cfg.ReadTimeout) * time.Second,          // 读取超时
		WriteTimeout:    time.Duration(cfg.WriteTimeout) * time.Second,         // 写入超时
		MaxRetries:      cfg.MaxRetries,                                        // 命令失败重试次数
		MinRetryBackoff: time.Duration(cfg.MinRetryBackoff) * time.Millisecond, // 最小重试间隔
		MaxRetryBackoff: time.Duration(cfg.MaxRetryBackoff) * time.Second,      // 最大重试间隔
	}
	// 创建客户端
	db := redis.NewClient(redisOpts)
	// 关键：测试连接有效性（捕获认证失败、网络不通等错误）
	if pingErr := db.Ping().Err(); pingErr != nil {
		// 连接失败时，关闭已创建的客户端，避免资源泄漏
		_ = db.Close()
		logger.Error("Redis 连接失败:", "dbKey", dbKey, "addr", redisOpts.Addr, "error", pingErr)
		return
	}
	multiDBPool.Store(dbKey, DbObj{Db: db, Pre: cfg.Pre})
	logger.Info("Redis 连接成功:", "dbKey", dbKey)
}

// reinitRedisConnections 重新初始化Redis连接（平滑切换）
func reinitRedisConnections() error {
	cfgMap := config.GetRedisConfig()

	// 存储需要关闭的旧连接
	oldConnections := make(map[string]DbObj)

	// 先收集所有旧连接
	multiDBPool.Range(func(key, value interface{}) bool {
		dbKey := key.(string)
		dbObj := value.(DbObj)
		oldConnections[dbKey] = dbObj
		return true
	})

	// 创建新连接
	for dbKey, cfg := range cfgMap {
		createRedisConnection(dbKey, cfg)
		// 从旧连接映射中移除已更新的连接
		delete(oldConnections, dbKey)
	}

	// 延迟关闭旧连接（给现有操作30秒时间完成）
	go func() {
		time.Sleep(30 * time.Second)
		for dbKey, dbObj := range oldConnections {
			if err := dbObj.Db.Close(); err != nil {
				logger.Error("关闭旧Redis连接失败:", "dbKey", dbKey, "error", err)
			} else {
				logger.Info("旧Redis连接已关闭:", "dbKey", dbKey)
			}
			multiDBPool.Delete(dbKey)
		}
	}()

	return nil
}
func GetRedisDB(dbKey string) (*RedisDb, error) {
	val, ok := multiDBPool.Load(dbKey)
	if !ok {
		return nil, fmt.Errorf("redis[%s]连接池未初始化", dbKey)
	}
	// 类型断言：将interface{}转为*sql.DB
	dbObj, ok := val.(DbObj)
	if !ok {
		return nil, fmt.Errorf("redis[%s]连接池类型错误", dbKey)
	}
	return &RedisDb{
		Db:    dbObj.Db,
		DbPre: dbObj.Pre,
	}, nil
}

// CloseRedis 关闭所有 Redis 连接（供外部调用，如服务停止时）
func CloseRedis() error {
	var err error
	multiDBPool.Range(func(key, value interface{}) bool {
		dbObj, ok := value.(DbObj)
		if !ok {
			err = fmt.Errorf("无效的 Redis 客户端对象（key: %v）", key)
			return false // 终止遍历
		}
		// 关闭客户端（会释放连接池中的所有连接）
		if closeErr := dbObj.Db.Close(); closeErr != nil {
			err = fmt.Errorf("关闭 Redis 连接失败（dbKey: %v）: %w", key, closeErr)
			// 继续遍历，尝试关闭其他连接
			return true
		}
		logger.Info("Redis 连接已关闭", "dbKey", key)
		return true
	})
	return err
}
