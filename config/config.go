package config

import (
	"encoding/json"
	"os"
	"sync"
	"time"
)

// AppConfig 单个应用配置
type AppConfig struct {
	Port string `json:"port"`
	Name string `json:"name"`
}

// LoggerConfig 日志配置
type LoggerConfig struct {
	Level string `json:"level"`
	Path  string `json:"path"`
}

// GlobalAppConfig 应用全局配置
type GlobalAppConfig struct {
	Apps   map[string]AppConfig `json:"apps"`
	Logger LoggerConfig         `json:"logger"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	MySQL   map[string]MySQLConfig   `json:"mysql"`
	Mongodb map[string]MongodbConfig `json:"mongodb"`
	Redis   map[string]RedisConfig   `json:"redisDb"`
}

// MySQLConfig MySQL连接配置
type MySQLConfig struct {
	Host            string `json:"host"`
	Port            string `json:"port"`
	User            string `json:"user"`
	Pwd             string `json:"pwd"`
	Dbname          string `json:"dbname"`
	Charset         string `json:"charset"`
	Pre             string `json:"pre"`
	MaxOpenConnNum  int    `json:"max_open_conn_num"`
	MaxIdleConnNum  int    `json:"max_idle_conn_num"`
	ConnMaxLifetime int    `json:"conn_max_lifetime"`
}

// MongodbConfig MongoDB连接配置
type MongodbConfig struct {
	Host        string        `json:"host"`
	Port        string        `json:"port"`
	User        string        `json:"user"`
	Pwd         string        `json:"pwd"`
	Dbname      string        `json:"dbname"`
	Pre         string        `json:"pre"`
	Charset     string        `json:"charset"`
	MaxPoolSize uint64        `json:"max_pool_size"` // 最大连接池大小
	Timeout     time.Duration `json:"timeout"`       // 连接超时时间(秒)
}

// RedisConfig redis连接配置
type RedisConfig struct {
	Host         string `json:"host"`
	Port         string `json:"port"`
	Pwd          string `json:"pwd"`
	Pre          string `json:"pre"`
	PoolSize     int    `json:"pool_size"`      // 最大连接池大小
	MinIdleConns int    `json:"min_idle_conns"` //在启动阶段创建指定数量的Idle连接，并长期维持idle状态的连接数不少于指定数量；
	IdleTimeout  int    `json:"idle_timeout"`   //连接池闲置连接超时，自动关闭过期连接(秒)
	Timeout      int    `json:"timeout"`        //表示连接超时(秒)
}
type PostLoadHook func(*GlobalAppConfig) error

var (
	appConfig     *GlobalAppConfig
	dbConfig      *DatabaseConfig
	appOnce       sync.Once
	databaseOnce  sync.Once
	postLoadHooks []PostLoadHook // 存储用户注册的钩子函数
)

// RegisterPostLoadHook 注册配置加载后的钩子函数
func RegisterPostLoadHook(hook PostLoadHook) {
	postLoadHooks = append(postLoadHooks, hook)
}

// LoadAppConfig 加载应用配置（JSON）
func LoadAppConfig(path string) *GlobalAppConfig {
	appOnce.Do(func() {
		file, err := os.Open(path)
		if err != nil {
			panic("加载应用配置失败: " + err.Error())
		}
		defer file.Close()
		appConfig = &GlobalAppConfig{}
		if err := json.NewDecoder(file).Decode(appConfig); err != nil {
			panic("解析应用配置失败: " + err.Error())
		}

		// 2. 执行用户注册的钩子函数（扩展逻辑）
		for _, hook := range postLoadHooks {
			if err := hook(appConfig); err != nil {
				panic("执行配置钩子函数失败: " + err.Error())
			}
		}
	})
	return appConfig
}

// LoadDatabaseConfig 加载数据库配置（JSON）
func LoadDatabaseConfig(path string) *DatabaseConfig {
	databaseOnce.Do(func() {
		file, err := os.Open(path)
		if err != nil {
			panic("加载数据库配置失败: " + err.Error())
		}
		defer file.Close()
		decoder := json.NewDecoder(file)
		if err := decoder.Decode(&dbConfig); err != nil {
			panic("解析数据库配置失败: " + err.Error())
		}
	})
	return dbConfig
}

// GetAppConfig 根据应用名获取配置
func GetAppConfig(appName string) AppConfig {
	return appConfig.Apps[appName]
}

// GetMysqlConfig 获取mysql数据库配置
func GetMysqlConfig() map[string]MySQLConfig {
	return dbConfig.MySQL
}

// GetMongodbConfig 获取数据库配置
func GetMongodbConfig() map[string]MongodbConfig {
	return dbConfig.Mongodb
}

// GetRedisConfig 获取mysql数据库配置
func GetRedisConfig() map[string]RedisConfig {
	return dbConfig.Redis
}
