package redisDb

import (
	"fmt"
	"github.com/dfpopp/go-dai/config"
	"github.com/dfpopp/go-dai/function"
	"github.com/go-redis/redis"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
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
	cfgMap := config.GetRedisConfig()
	fmt.Println(function.Json_encode(cfgMap))
	for dbKey, cfg := range cfgMap {
		if cfg.MinIdleConns < 2 {
			cfg.MinIdleConns = 2
		}
		if cfg.PoolSize < 4 {
			cfg.PoolSize = 4
		}
		// 端口默认值（避免配置缺失导致 Addr 格式错误）
		if cfg.Port == "" {
			cfg.Port = "6379"
		}
		// 构建 Redis 客户端配置
		redisOpts := &redis.Options{
			Network:  "tcp",
			Addr:     fmt.Sprintf("%s:%s", cfg.Host, cfg.Port), // 格式化 Addr，避免空端口
			Password: cfg.Pwd,                                  // 空密码直接传入，适配无密码环境
			DB:       0,

			PoolSize:     cfg.PoolSize,
			MinIdleConns: cfg.MinIdleConns,
			IdleTimeout:  300 * time.Second, // 优化：连接池闲置连接超时，自动关闭过期连接（避免资源浪费）
			// v7 版本的连接超时字段名是 DialTimeout（v8 是 Timeout，注意区别！）
			DialTimeout: 5 * time.Second, // v7 用 DialTimeout 表示连接超时
		}
		// 创建客户端
		db := redis.NewClient(redisOpts)
		// 关键：测试连接有效性（捕获认证失败、网络不通等错误）
		if pingErr := db.Ping().Err(); pingErr != nil {
			// 连接失败时，关闭已创建的客户端，避免资源泄漏
			_ = db.Close()
			fmt.Println(fmt.Errorf("Redis 连接失败（dbKey: %s, addr: %s）: %w", dbKey, redisOpts.Addr, pingErr))
			return
		}
		multiDBPool.Store(dbKey, DbObj{Db: db, Pre: cfg.Pre})
	}
	// 注册服务退出信号，触发 Redis 连接关闭（优雅退出）
	registerShutdownHook()
}
func GetRedisDB(dbKey string) (*RedisDb, error) {
	val, ok := multiDBPool.Load(dbKey)
	if !ok {
		return nil, fmt.Errorf("Redis[%s]连接池未初始化", dbKey)
	}
	// 类型断言：将interface{}转为*sql.DB
	dbObj, ok := val.(DbObj)
	if !ok {
		return nil, fmt.Errorf("Redis[%s]连接池类型错误", dbKey)
	}
	return &RedisDb{
		Db:    dbObj.Db,
		DbPre: dbObj.Pre,
	}, nil
}

// 注册服务退出钩子（监听信号，自动关闭 Redis 连接）
func registerShutdownHook() {
	sigCh := make(chan os.Signal, 1)
	// 监听常见的退出信号：Ctrl+C、kill 命令
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		<-sigCh // 等待信号
		fmt.Println("\n收到退出信号，开始关闭 Redis 连接...")
		if err := CloseRedis(); err != nil {
			fmt.Printf("Redis 连接关闭失败: %v\n", err)
		} else {
			fmt.Println("所有 Redis 连接已关闭")
		}
		os.Exit(0)
	}()
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
		fmt.Printf("Redis 连接已关闭（dbKey: %v）\n", key)
		return true
	})
	return err
}
