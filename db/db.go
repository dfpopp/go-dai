package db

import (
	"fmt"
	"sync"

	"github.com/dfpopp/go-dai/db/clickhouse"
	"github.com/dfpopp/go-dai/db/elasticSearch"
	"github.com/dfpopp/go-dai/db/mongoDb"
	"github.com/dfpopp/go-dai/db/mysql"
	"github.com/dfpopp/go-dai/db/postgresql"
	"github.com/dfpopp/go-dai/db/redisDb"
	"github.com/dfpopp/go-dai/function"
)

func StartDb(dbTypeList []string) {
	for _, dbType := range dbTypeList {
		switch dbType {
		case "mysql":
			mysql.InitMySQL()
		case "mongodb":
			mongoDb.InitMongoDB()
		case "redis":
			redisDb.InitRedis()
		case "es":
			elasticSearch.InitEs()
		case "postgresql":
			postgresql.InitPostgres()
		case "clickhouse":
			clickhouse.InitClickhouse()
		}
	}
}

// CloseDb 关闭所有数据库连接
func CloseDb(dbTypeList []string) {
	var wg sync.WaitGroup
	wg.Add(len(dbTypeList))
	if function.InArray("mysql", dbTypeList) {
		go func() {
			defer wg.Done()
			fmt.Println("\n开始关闭 Mysql 连接...")
			if err := mysql.CloseMysql(); err != nil {
				fmt.Printf("Mysql 连接关闭失败: %v\n", err)
			} else {
				fmt.Println("所有 Mysql 连接已关闭")
			}
		}()
	}
	if function.InArray("mongodb", dbTypeList) {
		go func() {
			defer wg.Done()
			fmt.Println("\n开始关闭 MongoDb 连接...")
			if err := mongoDb.CloseMongoDb(); err != nil {
				fmt.Printf("MongoDb 连接关闭失败: %v\n", err)
			} else {
				fmt.Println("所有 MongoDb 连接已关闭")
			}
		}()
	}
	if function.InArray("redis", dbTypeList) {
		go func() {
			defer wg.Done()
			fmt.Println("\n开始关闭 Redis 连接...")
			if err := redisDb.CloseRedis(); err != nil {
				fmt.Printf("Redis 连接关闭失败: %v\n", err)
			} else {
				fmt.Println("所有 Redis 连接已关闭")
			}
		}()
	}
	if function.InArray("es", dbTypeList) {
		go func() {
			defer wg.Done()
			fmt.Println("\n开始关闭 Es 连接...")
			if err := elasticSearch.CloseES(); err != nil {
				fmt.Printf("Es 连接关闭失败: %v\n", err)
			} else {
				fmt.Println("所有 ES 连接已关闭")
			}
		}()
	}
	if function.InArray("postgresql", dbTypeList) {
		go func() {
			defer wg.Done()
			fmt.Println("\n开始关闭 Postgresql 连接...")
			if err := postgresql.ClosePostgres(); err != nil {
				fmt.Printf("Postgresql 连接关闭失败: %v\n", err)
			} else {
				fmt.Println("所有 Postgresql 连接已关闭")
			}
		}()
	}
	if function.InArray("clickhouse", dbTypeList) {
		go func() {
			defer wg.Done()
			fmt.Println("\n开始关闭 Clickhouse 连接...")
			if err := clickhouse.CloseClickhouse(); err != nil {
				fmt.Printf("Clickhouse 连接关闭失败: %v\n", err)
			} else {
				fmt.Println("所有 Clickhouse 连接已关闭")
			}
		}()
	}
	wg.Wait()
}
