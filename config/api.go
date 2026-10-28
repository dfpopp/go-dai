package config

// GetAppConfig 获取应用配置
func GetAppConfig(appName string) *AppConfig {
	return appConfigMap[appName]
}

// GetDatabaseConfig 获取数据库配置
func GetDatabaseConfig() *DatabaseConfig {
	return DbConfig
}

// GetCustomConfig 获取自定义配置
func GetCustomConfig() *CustomConfig {
	configMu.RLock()
	defer configMu.RUnlock()
	return customConfig
}

// GetMysqlConfig 获取mysql数据库配置
func GetMysqlConfig() map[string]MySQLConfig {
	if DbConfig == nil {
		return nil
	}
	return DbConfig.MySQL
}

// GetEsConfig 获取ES数据库配置
func GetEsConfig() map[string]EsConfig {
	if DbConfig == nil {
		return nil
	}
	return DbConfig.Es
}

// GetMongodbConfig 获取MongoDB数据库配置
func GetMongodbConfig() map[string]MongodbConfig {
	if DbConfig == nil {
		return nil
	}
	return DbConfig.Mongodb
}

// GetRedisConfig 获取Redis数据库配置
func GetRedisConfig() map[string]RedisConfig {
	if DbConfig == nil {
		return nil
	}
	return DbConfig.Redis
}

// GetPostgresqlConfig 获取PostgreSQL数据库配置
func GetPostgresqlConfig() map[string]PostgresqlConfig {
	if DbConfig == nil {
		return nil
	}
	return DbConfig.Postgresql
}

// GetClickhouseConfig 获取ClickHouse数据库配置
func GetClickhouseConfig() map[string]ClickhouseConfig {
	if DbConfig == nil {
		return nil
	}
	return DbConfig.Clickhouse
}

// GetGlobalConfig 获取全局配置实例
func GetGlobalConfig() Config {
	if globalConfig == nil {
		globalConfig = NewMemoryConfig()
	}
	return globalConfig
}
