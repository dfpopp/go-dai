package logger

import (
	"errors"
	"fmt"
	"github.com/dfpopp/go-dai/config"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Logger 日志接口
type Logger interface {
	Info(v ...interface{})   //正常日志
	Warn(v ...interface{})   //警告日志
	Error(v ...interface{})  //错误日志
	Affair(v ...interface{}) //事务错误日志
	GetEnv() string          //获取当前运行环境
}

// DefaultLogger 默认日志实现
type DefaultLogger struct {
	infoLogger   *log.Logger
	warnLogger   *log.Logger
	errorLogger  *log.Logger
	affairLogger *log.Logger
	cfg          *config.AppConfig
}

var (
	defaultLogger *DefaultLogger
	once          sync.Once
)

// InitLogger 初始化日志（保留原有逻辑）
func InitLogger(appName string) error {
	var err error
	once.Do(func() {
		cfg := config.GetAppConfig(appName)
		if cfg == nil {
			err = errors.New("应用配置未加载")
			return
		}
		logPath := cfg.Logger.Path
		today := time.Now().Format("20060102") // 简化日期格式化（替代原有function.TimeToStr）
		infoLogPath := filepath.Join(logPath, "info", today+".log")
		warnLogPath := filepath.Join(logPath, "warn", today+".log")
		errLogPath := filepath.Join(logPath, "error", today+".log")
		affairLogPath := filepath.Join(logPath, "affair.log")

		// 创建日志目录
		dirs := []string{filepath.Dir(infoLogPath), filepath.Dir(warnLogPath), filepath.Dir(errLogPath), filepath.Dir(affairLogPath)}
		for _, dir := range dirs {
			if mkdirErr := os.MkdirAll(dir, 0755); mkdirErr != nil {
				err = fmt.Errorf("创建日志目录失败: %s, err=%v", dir, mkdirErr)
				return
			}
		}

		// 打开日志文件
		infoFile, infoErr := os.OpenFile(infoLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if infoErr != nil {
			err = fmt.Errorf("打开信息日志文件失败: %v", infoErr)
			return
		}
		// 打开警告日志文件
		warnFile, warnErr := os.OpenFile(warnLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if warnErr != nil {
			err = fmt.Errorf("打开警告日志文件失败: %v", warnErr)
			return
		}
		errFile, errErr := os.OpenFile(errLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if errErr != nil {
			err = fmt.Errorf("打开错误日志文件失败: %v", errErr)
			return
		}
		affairFile, affairErr := os.OpenFile(affairLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if affairErr != nil {
			err = fmt.Errorf("打开事务日志文件失败: %v", affairErr)
			return
		}

		// 初始化日志器
		defaultLogger = &DefaultLogger{
			infoLogger:   log.New(infoFile, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile),
			warnLogger:   log.New(warnFile, "WARN: ", log.Ldate|log.Ltime|log.Lshortfile),
			errorLogger:  log.New(errFile, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile),
			affairLogger: log.New(affairFile, "AFFAIR: ", log.Ldate|log.Ltime|log.Lshortfile),
			cfg:          cfg,
		}
	})
	return err
}

// GetLogger 获取日志实例
func GetLogger() Logger {
	return defaultLogger
}

// Info 打印信息日志（同时输出到控制台）
func (l *DefaultLogger) Info(v ...interface{}) {
	if defaultLogger.cfg.Env == "prod" {
		l.infoLogger.Println(v...)
	} else {
		fmt.Println(append([]interface{}{"INFO: "}, v...)...)
	}
}

// Warn 打印警告日志（新增实现）
func (l *DefaultLogger) Warn(v ...interface{}) {
	if defaultLogger.cfg.Env == "prod" {
		l.warnLogger.Println(v...)
	} else {
		fmt.Println(append([]interface{}{"WARN: "}, v...)...)
	}
}

// Error 打印错误日志（同时输出到控制台）
func (l *DefaultLogger) Error(v ...interface{}) {
	if defaultLogger.cfg.Env == "prod" {
		l.errorLogger.Println(v...)
	} else {
		fmt.Println(append([]interface{}{"ERROR: "}, v...)...)
	}
}

// Affair 打印事务日志（同时输出到控制台）
func (l *DefaultLogger) Affair(v ...interface{}) {
	if defaultLogger.cfg.Env == "prod" {
		l.affairLogger.Println(v...)
	} else {
		fmt.Println(append([]interface{}{"AFFAIR: "}, v...)...)
	}
}
func (l *DefaultLogger) GetEnv() string {
	return l.cfg.Env
}

// 全局快捷方法

func Info(v ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Info(v...)
	}
}
func Warn(v ...interface{}) { // 新增全局快捷方法
	if defaultLogger != nil {
		defaultLogger.Warn(v...)
	}
}
func Error(v ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Error(v...)
	}
}

func Affair(v ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Affair(v...)
	}
}
