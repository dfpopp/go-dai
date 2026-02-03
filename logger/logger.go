package logger

import (
	"errors"
	"fmt"
	"github.com/dfpopp/go-dai/config"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Logger 日志接口
type Logger interface {
	Info(v ...interface{})  //正常日志
	Warn(v ...interface{})  //警告日志
	Error(v ...interface{}) //错误日志
	GetEnv() string         //获取当前运行环境
}

// DefaultLogger 默认日志实现
type DefaultLogger struct {
	infoLogger  *log.Logger
	warnLogger  *log.Logger
	errorLogger *log.Logger
	cfg         *config.AppConfig
	appPath     string
}

var (
	defaultLogger *DefaultLogger
	once          sync.Once
)

// 核心修改：自定义日志前缀（包含业务代码的文件和行号）
func getCallerPrefix() string {
	fileList := make([]string, 0)
	goRoot := os.Getenv("GOROOT")
	// 遍历调用栈，跳过logger包内的调用，找到业务代码位置
	for i := 0; i < 10; i++ {
		pc, filePath, lineNum, ok := runtime.Caller(i)
		if !ok {
			break
		}
		// 获取调用函数所属包名
		funcInfo := runtime.FuncForPC(pc)
		if funcInfo == nil {
			continue
		}
		funcName := funcInfo.Name()
		// 跳过logger包内的所有调用（匹配包路径关键词）
		if strings.Contains(funcName, "github.com/dfpopp/go-dai/logger") || strings.Contains(filePath, goRoot) {
			continue
		}
		// 只保留文件名+行号（如 login.go:25），也可保留完整路径
		newFilePath := getFilePath(filePath)
		fileList = append([]string{fmt.Sprintf("[%s:%d] ", newFilePath, lineNum)}, fileList...)
		if strings.Contains(filePath, "/app/") {
			break
		}
	}
	if len(fileList) > 0 {
		return strings.Join(fileList, "->")
	}
	// 未找到时返回默认前缀
	return "[unknown:0] "
}

// InitLogger 初始化日志（移除 log.Lshortfile，改用自定义前缀）
func InitLogger(appName string, appPath string) error {
	var err error
	once.Do(func() {
		cfg := config.GetAppConfig(appName)
		if cfg == nil {
			err = errors.New("应用配置未加载")
			return
		}
		logPath := cfg.Logger.Path
		today := time.Now().Format("20060102")
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

		// 关键修改：移除 log.Lshortfile，只保留时间和日期
		defaultLogger = &DefaultLogger{
			infoLogger:  log.New(infoFile, "INFO: ", log.Ldate|log.Ltime),
			warnLogger:  log.New(warnFile, "WARN: ", log.Ldate|log.Ltime),
			errorLogger: log.New(errFile, "ERROR: ", log.Ldate|log.Ltime),
			cfg:         cfg,
			appPath:     appPath,
		}
	})
	return err
}

// GetLogger 获取日志实例
func GetLogger() Logger {
	return defaultLogger
}

// Info 打印信息日志（添加业务代码位置前缀）
func (l *DefaultLogger) Info(v ...interface{}) {
	// 拼接业务代码位置前缀

	if l.cfg.Env == "prod" {
		l.infoLogger.Println(v...)
	} else {
		fmt.Println(append([]interface{}{"INFO: "}, v...)...)
	}
}

// Warn 打印警告日志（添加业务代码位置前缀）
func (l *DefaultLogger) Warn(v ...interface{}) {
	prefix := getCallerPrefix()
	newV := append([]interface{}{prefix}, v...)

	if l.cfg.Env == "prod" {
		l.warnLogger.Println(newV...)
	} else {
		fmt.Println(append([]interface{}{"WARN: "}, newV...)...)
	}
}

// Error 打印错误日志（添加业务代码位置前缀）
func (l *DefaultLogger) Error(v ...interface{}) {
	prefix := getCallerPrefix()
	newV := append([]interface{}{prefix}, v...)

	if l.cfg.Env == "prod" {
		l.errorLogger.Println(newV...)
	} else {
		fmt.Println(append([]interface{}{"ERROR: "}, newV...)...)
	}
}

func (l *DefaultLogger) GetEnv() string {
	return l.cfg.Env
}

// 全局快捷方法（无需修改，会自动调用带前缀的方法）
func Info(v ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Info(v...)
	}
}

func Warn(v ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Warn(v...)
	}
}

func Error(v ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Error(v...)
	}
}
func getFilePath(filePath string) string {
	if strings.Contains(filePath, "/go-Dai/") {
		pathList := strings.Split(filePath, "/go-Dai/")
		if len(pathList) == 2 {
			return "/go-Dai/" + pathList[1]
		} else {
			return filePath
		}
	} else if strings.Contains(filePath, defaultLogger.appPath) {
		return strings.ReplaceAll(filePath, defaultLogger.appPath, "")
	} else {
		pathList := strings.Split(filePath, "/")
		if len(pathList) > 4 {
			return strings.Join(pathList[len(pathList)-4:], "/")
		} else {
			return filePath
		}
	}

}
