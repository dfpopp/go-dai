package config

import (
	"fmt"
	"os"
	"time"
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
	IdleTimeout    int    `json:"idle_timeout"`
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
	MySQL      map[string]MySQLConfig      `json:"mysql"`
	Mongodb    map[string]MongodbConfig    `json:"mongodb"`
	Redis      map[string]RedisConfig      `json:"redis"`
	Es         map[string]EsConfig         `json:"es"`
	Postgresql map[string]PostgresqlConfig `json:"postgresql"`
	Clickhouse map[string]ClickhouseConfig `json:"clickhouse"`
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
	// 连接超时参数（秒）：timeout拨号/readTimeout读/writeTimeout写，应对主备切换等端点级故障
	ConnectTimeout int `json:"connect_timeout"` // 拨号超时，默认5；保证fail fast让重试有意义
	ReadTimeout    int `json:"read_timeout"`    // 读超时，默认30；必须≥业务最长查询的ctx超时，否则误杀慢查询
	WriteTimeout   int `json:"write_timeout"`   // 写超时，默认30；同上
	// 连接类瞬时错误退避重试：仅非事务路径生效，默认只重试「确定未执行」的错误（拨号失败、ErrBadConn）
	RetryMaxAttempts   int  `json:"retry_max_attempts"`    // 最大尝试次数（含首次），默认3
	RetryBackoffMs     int  `json:"retry_backoff_ms"`      // 退避基数（毫秒），指数递增100ms→200ms→400ms，默认100
	RetryOnInvalidConn bool `json:"retry_on_invalid_conn"` // 额外重试「结果未知」错误（ErrInvalidConn等），调用方必须保证操作幂等，默认false
}

// PostgresqlConfig PostgreSQL连接配置
type PostgresqlConfig struct {
	Host            string `json:"host"`
	Port            string `json:"port"`
	User            string `json:"user"`
	Pwd             string `json:"pwd"`
	Dbname          string `json:"dbname"`
	Pre             string `json:"pre"`
	SslMode         string `json:"ssl_mode"` // SSL模式（disable/require/verify-ca/verify-full），默认disable
	MaxOpenConnNum  int    `json:"max_open_conn_num"`
	MaxIdleConnNum  int    `json:"max_idle_conn_num"`
	ConnMaxIdleTime int    `json:"conn_max_idleTime"`
	ConnMaxLifetime int    `json:"conn_max_lifetime"`
}

// ClickhouseConfig ClickHouse连接配置
type ClickhouseConfig struct {
	Host            string `json:"host"`
	Port            string `json:"port"`
	User            string `json:"user"`
	Pwd             string `json:"pwd"`
	Dbname          string `json:"dbname"`
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

// ConfigChangeHook 配置变更钩子函数类型
type ConfigChangeHook func(configType string, config interface{}) error

var (
	appConfigMap       = make(map[string]*AppConfig)
	DbConfig           *DatabaseConfig
	configFileWatchers = make(map[string]*ConfigFileWatcher) // 配置文件监听器
	configHooks        = make([]ConfigChangeHook, 0)         // 配置变更钩子
)

// RegisterConfigChangeHook 注册配置变更钩子
func RegisterConfigChangeHook(hook ConfigChangeHook) {
	configHooks = append(configHooks, hook)
}

// notifyConfigChange 通知配置变更
func notifyConfigChange(configType string, config interface{}) {
	for _, hook := range configHooks {
		if err := hook(configType, config); err != nil {
			fmt.Printf("执行配置变更钩子失败: %v\n", err)
		}
	}
}

// ConfigFileWatcher 配置文件监听器
type ConfigFileWatcher struct {
	filePath      string
	lastModTime   time.Time
	watchInterval time.Duration
	stopChan      chan struct{}
	isRunning     bool
}

// NewConfigFileWatcher 创建配置文件监听器
func NewConfigFileWatcher(filePath string, interval time.Duration) *ConfigFileWatcher {
	return &ConfigFileWatcher{
		filePath:      filePath,
		watchInterval: interval,
		stopChan:      make(chan struct{}),
	}
}

// Start 启动配置文件监听
func (w *ConfigFileWatcher) Start(configType string) {
	if w.isRunning {
		return
	}
	w.isRunning = true

	// 获取初始修改时间
	info, err := os.Stat(w.filePath)
	if err == nil {
		w.lastModTime = info.ModTime()
	}

	go func() {
		ticker := time.NewTicker(w.watchInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				info, err := os.Stat(w.filePath)
				if err != nil {
					continue
				}

				// 检查文件是否被修改
				if info.ModTime().After(w.lastModTime) {
					w.lastModTime = info.ModTime()
					fmt.Printf("配置文件已修改，重新加载: %s\n", w.filePath)

					// 重新加载配置
					err := w.reloadConfig(configType)
					if err != nil {
						fmt.Printf("重新加载配置失败: %v\n", err)
						// 继续监听，不停止整个循环
						continue
					}
				}
			case <-w.stopChan:
				w.isRunning = false
				return
			}
		}
	}()
}

// Stop 停止配置文件监听
func (w *ConfigFileWatcher) Stop() {
	if w.isRunning {
		w.stopChan <- struct{}{}
	}
}

// reloadConfig 重新加载配置
func (w *ConfigFileWatcher) reloadConfig(configType string) error {
	switch configType {
	case "app":
		// 重新加载应用配置
		// 注意：这里需要重新解析配置文件并更新appConfigMap
		// 为了简化，我们可以创建一个新的加载函数
		err := LoadAppConfig(w.filePath)
		if err != nil {
			return err
		}
		// 通知配置变更
		notifyConfigChange("app", appConfigMap)

	case "database":
		// 重新加载数据库配置
		err := LoadDatabaseConfig(w.filePath)
		if err != nil {
			return err
		}
		// 通知配置变更
		notifyConfigChange("database", DbConfig)

	case "custom":
		// 重新加载自定义配置
		// 直接调用LoadCustomConfig方法，保持与app和database类型的一致性
		err := LoadCustomConfig(w.filePath)
		if err != nil {
			return err
		}
		// 通知配置变更
		notifyConfigChange("custom", customConfig)
	}
	return nil
}
