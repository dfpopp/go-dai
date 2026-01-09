package elasticSearch

import (
	"compress/gzip"
	"errors"
	"fmt"
	"github.com/dfpopp/go-dai/logger"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"io"
	"regexp"
	"strings"
)

var validIndexNameRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9_\-]*$`)
var validIdentifierRegex = regexp.MustCompile(`^[a-zA-Z0-9_\s]+(\.[a-zA-Z0-9_\s]+)?$`)
var validIncRegex = regexp.MustCompile(`^[a-zA-Z0-9_=?+\-\s]+(\.[a-zA-Z0-9_=?+\-\s]+)?$`)

// 校验表名是否为合法标识符（防止注入）
func isValidIndexName(s string) bool {
	if !validIndexNameRegex.MatchString(s) {
		return false
	}
	return true
}

// 校验表名/字段名是否为合法标识符（防止注入）
func isValidIdentifier(s string) bool {
	if s == "*" { // 通配符*允许
		return true
	}
	// 多字段用逗号分隔，逐个校验
	fields := strings.Split(s, ",")
	for _, f := range fields {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		if !validIdentifierRegex.MatchString(f) {
			return false
		}
	}
	return true
}

// 校验表名/字段名是否为合法标识符（防止注入）
func isValidInc(s string) bool {
	if s == "*" { // 通配符*允许
		return true
	}
	// 多字段用逗号分隔，逐个校验
	fields := strings.Split(s, ",")
	for _, f := range fields {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		if !validIncRegex.MatchString(f) {
			return false
		}
	}
	return true
}

// 校验关联语句是否合法（防止注入）
func isValidRelation(relation string) bool {
	// 仅允许合法的JOIN关键字，且包含ON条件
	relation = strings.TrimSpace(relation)
	if relation == "" {
		return false
	}
	joinKeywords := []string{"LEFT JOIN", "RIGHT JOIN", "INNER JOIN", "JOIN"}
	hasValidJoin := false
	for _, kw := range joinKeywords {
		if strings.HasPrefix(strings.ToUpper(relation), kw) {
			hasValidJoin = true
			break
		}
	}
	// 必须包含ON条件
	hasOn := strings.Contains(strings.ToUpper(relation), " ON ")
	return hasValidJoin && hasOn
}

// DeZip 对响应体进行gzip解压（优化版：安全处理gzip.Reader关闭）
func DeZip(isGzip bool, response *esapi.Response) ([]byte, error) {
	if response.Body == nil {
		return nil, errors.New("响应体为空")
	}

	var (
		bodyReader io.Reader = response.Body
		gzReader   *gzip.Reader
		err        error
	)

	// 仅当开启gzip且响应头包含gzip时解压
	if isGzip && strings.EqualFold(response.Header.Get("Content-Encoding"), "gzip") {
		gzReader, err = gzip.NewReader(response.Body)
		if err != nil {
			return nil, fmt.Errorf("创建gzip reader失败：%w", err)
		}
		bodyReader = gzReader
	}

	// 读取所有数据（核心：先读完全部数据，再处理关闭）
	bodyBytes, err := io.ReadAll(bodyReader)
	if err != nil {
		// 读取失败时，优先关闭gzipReader（若存在），再返回错误
		if gzReader != nil {
			_ = gzReader.Close() // 忽略关闭错误，避免覆盖原读取错误
		}
		return nil, fmt.Errorf("读取响应体失败：%w", err)
	}

	// 读取成功后，关闭gzipReader（仅清理其内部缓冲区，不影响底层Reader）
	if gzReader != nil {
		if err := gzReader.Close(); err != nil {
			logger.Error("关闭gzip reader失败（不影响数据）：%v", err)
		}
	}

	return bodyBytes, nil
}

// convertToString 将任意类型转换为字符串（ES文档ID专用）
func convertToString(v interface{}) (string, error) {
	switch val := v.(type) {
	case string:
		return val, nil
	case int, int8, int16, int32, int64:
		return fmt.Sprintf("%d", val), nil
	case uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", val), nil
	case float32, float64:
		return fmt.Sprintf("%f", val), nil
	case bool:
		return fmt.Sprintf("%t", val), nil
	default:
		return "", fmt.Errorf("不支持的类型：%T", v)
	}
}
