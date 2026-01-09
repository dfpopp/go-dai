package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type AppConfig struct {
	Name      string          `json:"name"`
	Env       string          `json:"env"` // dev/prod/test
	HTTP      HTTPConfig      `json:"http"`
	WebSocket WebSocketConfig `json:"websocket"`
	GRPC      GRPCConfig      `json:"grpc"`
	Logger    LoggerConfig    `json:"logger"`
}

// HTTPConfig HTTP配置
type HTTPConfig struct {
	Addr           string `json:"addr"`
	ReadTimeout    int    `json:"read_timeout"`
	WriteTimeout   int    `json:"write_timeout"`
	MaxHeaderBytes int    `json:"max_header_bytes"`
	SSL            bool   `json:"ssl"`
	SSLCertFile    string `json:"ssl_cert_file"`
	SSLKeyFile     string `json:"ssl_key_file"`
}

// WebSocketConfig WebSocket服务器配置
type WebSocketConfig struct {
	Addr             string `json:"addr"`              // 监听地址（ip:port）
	ReadTimeout      int    `json:"read_timeout"`      // 读超时（秒）
	WriteTimeout     int    `json:"write_timeout"`     // 写超时（秒）
	Path             string `json:"path"`              // WebSocket监听路径（如：/ws）
	Origin           string `json:"origin"`            // 允许的来源（* 表示允许所有）
	HandshakeTimeout int    `json:"handshake_timeout"` // 握手超时（秒）
	MaxMessageSize   int64  `json:"max_message_size"`  // 最大消息大小（字节，默认1MB）
	MaxConnections   int32  `json:"max_connections"`   // 最大连接数（默认1000）
	SSL              bool   `json:"ssl"`               //是否启用SSL/TLS（启用后为WSS，禁用为WS）
	SSLCertFile      string `json:"ssl_cert_file"`     //SSL证书路径（如：./cert/server.crt）
	SSLKeyFile       string `json:"ssl_key_file"`      //SSL密钥路径（如：./cert/server.key）
}

// GRPCConfig gRPC配置
type GRPCConfig struct {
	Addr                 string `json:"addr"`
	MaxRecvMsgSize       int    `json:"max_recv_msg_size"`
	MaxSendMsgSize       int    `json:"max_send_msg_size"`
	KeepaliveTime        int    `json:"keepalive_time"`         // 新增：保活时间（秒）
	KeepaliveTimeout     int    `json:"keepalive_timeout"`      // 新增：保活超时（秒）
	MaxConcurrentStreams uint32 `json:"max_concurrent_streams"` // 新增：最大并发流数
	Timeout              int    `json:"timeout"`
	SSL                  bool   `json:"ssl"`
	SSLCertFile          string `json:"ssl_cert_file"`
	SSLKeyFile           string `json:"ssl_key_file"`
}

// LoggerConfig 日志配置
type LoggerConfig struct {
	Path string `json:"path"`
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
	Redis   map[string]RedisConfig   `json:"redis"`
	Es      map[string]EsConfig      `json:"es"`
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
	ConnMaxIdleTime int    `json:"conn_max_idleTime"`
	ConnMaxLifetime int    `json:"conn_max_lifetime"`
}

// MongodbConfig MongoDB连接配置
type MongodbConfig struct {
	Host            string `json:"host"`
	Port            string `json:"port"`
	User            string `json:"user"`
	Pwd             string `json:"pwd"`
	Dbname          string `json:"dbname"`
	Pre             string `json:"pre"`
	Charset         string `json:"charset"`
	MaxPoolSize     uint64 `json:"max_pool_size"`      // 最大连接池大小
	MinPoolSize     uint64 `json:"min_pool_size"`      // 最小空闲连接数
	MaxConnIdleTime int    `json:"max_conn_idle_time"` // 空闲连接 多少秒后关闭
	Timeout         int    `json:"timeout"`            // 连接超时时间(秒)
}

// RedisConfig redis连接配置
type RedisConfig struct {
	Host            string `json:"host"`
	Port            string `json:"port"`
	Pwd             string `json:"pwd"`
	Pre             string `json:"pre"`
	Db              int    `json:"db_index"`       // 选中的数据库（默认 0）
	PoolSize        int    `json:"pool_size"`      // 最大连接池大小
	MinIdleConns    int    `json:"min_idle_conns"` //在启动阶段创建指定数量的Idle连接，并长期维持idle状态的连接数不少于指定数量；
	MaxConnLifetime int    `json:"max_conn_lifetime"`
	IdleTimeout     int    `json:"idle_timeout"`      //连接池闲置连接超时，自动关闭过期连接(秒)
	ReadTimeout     int    `json:"read_timeout"`      //读取超时 (秒)
	WriteTimeout    int    `json:"write_timeout"`     //写入超时 (秒)
	Timeout         int    `json:"timeout"`           //表示连接超时(秒)
	MaxRetries      int    `json:"max_retries"`       // 命令失败重试次数
	MinRetryBackoff int    `json:"min_retry_backoff"` // 最小重试间隔（毫秒）
	MaxRetryBackoff int    `json:"max_retry_backoff"` // 最大重试间隔（毫秒）
}

// EsConfig ES连接配置
type EsConfig struct {
	Host                  string `json:"host"`
	Port                  string `json:"port"`
	User                  string `json:"user"`
	Pwd                   string `json:"pwd"`
	Pre                   string `json:"pre"`
	GzipStatus            bool   `json:"gzip_status"`
	EnableTLS             bool   // 是否开启HTTPS
	InsecureTLS           bool   // 跳过TLS证书验证（测试环境用）
	MaxIdleConnNum        int    `json:"max_idle_conn_num"`          // 全局最大空闲连接
	MaxIdleConnNumPerHost int    `json:"max_idle_conn_num_per_host"` // 每个主机最大空闲连接
	IdleConnTimeout       int    `json:"idle_conn_timeout"`          //空闲连接超时释放(秒)
	MaxConnNumPerHost     int    `json:"max_conn_num_per_host"`      //每个主机最大并发连接（限制并发）
	Timeout               int    `json:"timeout"`                    // 连接建立超时（TCP握手）
	KeepAlive             int    `json:"keep_alive"`                 // 长连接保活
	ResponseHeaderTimeout int    `json:"response_header_timeout"`    //响应头超时
	TLSHandshakeTimeout   int    `json:"tls_handshake_timeout"`      // TLS握手超时
}
type PostLoadHook func() error

var (
	appConfigMap       = make(map[string]*AppConfig)
	DbConfig           *DatabaseConfig
	appConfigOnce      sync.Once
	databaseConfigOnce sync.Once
	postLoadHooks      []PostLoadHook // 存储用户注册的钩子函数
)

// RegisterPostLoadHook 注册配置加载后的钩子函数
func RegisterPostLoadHook(hook PostLoadHook) {
	postLoadHooks = append(postLoadHooks, hook)
}

// LoadAppConfig 加载应用配置
// LoadAppConfig 单例加载应用配置（使用appConfigOnce）
func LoadAppConfig(filePath string, appNames ...string) error {
	var err error
	appConfigOnce.Do(func() {
		// 读取配置文件
		data, readErr := os.ReadFile(filepath.Clean(filePath))
		if readErr != nil {
			err = readErr
			return
		}

		// 解析配置
		var cfgMap map[string]*AppConfig
		if unmarshalErr := json.Unmarshal(data, &cfgMap); unmarshalErr != nil {
			err = unmarshalErr
			return
		}

		// 加载指定应用配置
		for _, appName := range appNames {
			if cfg, ok := cfgMap[appName]; ok {
				appConfigMap[appName] = cfg
			}
		}
		// 执行配置加载后钩子（新增核心逻辑）
		if len(postLoadHooks) > 0 {
			for _, hook := range postLoadHooks {
				if hookErr := hook(); hookErr != nil {
					return
				}
			}
		}
	})
	return err
}

// LoadDatabaseConfig 加载数据库配置
func LoadDatabaseConfig(filePath string) error {
	var err error
	databaseConfigOnce.Do(func() {
		data, readErr := os.ReadFile(filepath.Clean(filePath))
		if readErr != nil {
			err = readErr
			return
		}

		var cfg DatabaseConfig
		if unmarshalErr := json.Unmarshal(data, &cfg); unmarshalErr != nil {
			err = unmarshalErr
			return
		}

		DbConfig = &cfg
	})
	return err
}

// GetAppConfig 获取应用配置
func GetAppConfig(appName string) *AppConfig {
	return appConfigMap[appName]
}

// GetDatabaseConfig 获取数据库配置
func GetDatabaseConfig() *DatabaseConfig {
	return DbConfig
}

// GetMysqlConfig 获取mysql数据库配置
func GetMysqlConfig() map[string]MySQLConfig {
	return DbConfig.MySQL
}

// GetEsConfig 获取mysql数据库配置
func GetEsConfig() map[string]EsConfig {
	return DbConfig.Es
}

// GetMongodbConfig 获取数据库配置
func GetMongodbConfig() map[string]MongodbConfig {
	return DbConfig.Mongodb
}

// GetRedisConfig 获取mysql数据库配置
func GetRedisConfig() map[string]RedisConfig {
	return DbConfig.Redis
}
