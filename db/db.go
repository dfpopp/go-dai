package db

import (
	"fmt"
	"github.com/dfpopp/go-dai/db/elasticSearch"
	"github.com/dfpopp/go-dai/db/mongoDb"
	"github.com/dfpopp/go-dai/db/mysql"
	"github.com/dfpopp/go-dai/db/redisDb"
	"github.com/dfpopp/go-dai/function"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

func StartDb(dbTypeList []string) {
	// 注册服务退出信号，触发 所有数据库连接关闭（优雅退出）
	registerShutdownHook(dbTypeList)
	for _, dbType := range dbTypeList {
		switch dbType {
		case "mysql":
			mysql.InitMySQL()
			break
		case "mongodb":
			mongoDb.InitMongoDB()
			break
		case "redis":
			redisDb.InitRedis()
		case "es":
			elasticSearch.InitEs()
		}
	}
}

// 注册服务退出钩子（监听信号，自动关闭 mysql 连接）
func registerShutdownHook(dbTypeList []string) {
	sigCh := make(chan os.Signal, 1)
	// 监听常见的退出信号：Ctrl+C、kill 命令
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		<-sigCh // 等待信号
		var wg sync.WaitGroup
		wg.Add(len(dbTypeList))
		if function.InArray("mysql", dbTypeList) {
			go func() {
				defer wg.Done()
				fmt.Println("\n收到退出信号，开始关闭 Mysql 连接...")
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
				fmt.Println("\n收到退出信号，开始关闭 MongoDb 连接...")
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
				fmt.Println("\n收到退出信号，开始关闭 Redis 连接...")
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
				fmt.Println("\n收到退出信号，开始关闭 Es 连接...")
				if err := elasticSearch.CloseES(); err != nil {
					fmt.Printf("Es 连接关闭失败: %v\n", err)
				} else {
					fmt.Println("所有 ES 连接已关闭")
				}
			}()
		}
		wg.Wait()
		os.Exit(0)
	}()
}
