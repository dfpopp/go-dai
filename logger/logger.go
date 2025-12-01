package logger

import (
	"github.com/dfpopp/go-dai/config"
	"log"
	"os"
	"path/filepath"
)

var (
	infoLogger  *log.Logger
	errorLogger *log.Logger
)

// InitLogger 初始化日志
func InitLogger() {
	cfg := config.LoadAppConfig("config/app.json")
	logPath := cfg.Logger.Path

	// 创建日志目录
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		panic("创建日志目录失败: " + err.Error())
	}

	// 打开日志文件
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		panic("打开日志文件失败: " + err.Error())
	}

	// 初始化日志器
	infoLogger = log.New(logFile, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	errorLogger = log.New(logFile, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
}

// Info 打印信息日志
func Info(v ...interface{}) {
	infoLogger.Println(v...)
	log.Println(v...) // 同时输出到控制台
}

// Error 打印错误日志
func Error(v ...interface{}) {
	errorLogger.Println(v...)
	log.Println(v...) // 同时输出到控制台
}
