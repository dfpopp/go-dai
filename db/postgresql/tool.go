package postgresql

import (
	"regexp"
	"strings"
)

var validTableRegex = regexp.MustCompile(`^` +
	// 库名部分（可选）：支持 `库名` 或 库名 格式
	`(?:` +
	`(?:` +
	// 带反引号的库名：`开头 + 合法字符 + `结尾
	// 合法字符：字母/下划线开头，后跟字母/数字/下划线
	"`[a-zA-Z_][a-zA-Z0-9_]*`" +
	`|` +
	// 不带反引号的库名
	`[a-zA-Z_][a-zA-Z0-9_]*` +
	`)` +
	// 库名和表名之间的点
	`\.` +
	`)?` +
	// 表名部分（必填）：支持 `表名` 或 表名 格式
	`(?:` +
	// 带反引号的表名
	"`[a-zA-Z_][a-zA-Z0-9_]*`" +
	`|` +
	// 不带反引号的表名
	`[a-zA-Z_][a-zA-Z0-9_]*` +
	`)` +
	`$`)

var validFieldRegex = regexp.MustCompile(`(?i)(?:
	# 第一部分：注入风险特征（匹配到即非法）
	\b(UNION|SELECT|INSERT|UPDATE|DELETE|DROP|ALTER|CREATE|REPLACE|EXEC|EXECUTE)\b|  # 危险关键字
	--|/\*|\*/|#|                                                                   # 注释符
	;|                                                                              # 语句分隔符
	\bOR\s+1\s*=|\bAND\s+1\s*=|                                                     # 万能密码注入
	\bWHERE\b|\bFROM\b|\bJOIN\b|                                                    # 非法子句关键字
	# 第二部分：非法字符（匹配到即非法，仅允许[\w\s().,'"]）
	[^a-zA-Z0-9_\s().,'"]
)`)
var validWhereRegex = regexp.MustCompile(`(?i)
    (?:--|#|;|\|\|)                          # 注释符、分号、管道符（终止语句/拼接）
    |(?:UNION\s+ALL\s+SELECT|UNION\s+SELECT) # UNION注入
    |(?:INSERT\s+INTO|UPDATE\s+|DELETE\s+FROM|DROP\s+|ALTER\s+|TRUNCATE\s+) # 危险操作
    |(?:EXEC\s+|XP_|\s+OR\s+\d+\s*=\s*\d+|AND\s+\d+\s*=\s*\d+) # 逻辑注入/执行命令
    |(?:CHAR\s*\(|ASCII\s*\(|CONCAT\s*\(|GROUP_CONCAT\s*\()    # 字符拼接函数
    |(?:SELECT\s+.+?\s+FROM\s+.+?)          # 嵌套查询
    |(?:['"]).*?['"]                        # 单双引号（参数化查询不应出现）
    |(?:\$\{|\}\$)                          # 模板注入
`)
var validOrderRegex = regexp.MustCompile(`^(?:` +
	// 单个排序项：支持`字段名`/`字段名`.`子字段名` 格式，可选asc/desc
	"`?[a-zA-Z_][a-zA-Z0-9_]*`?(?:\\.`?[a-zA-Z_][a-zA-Z0-9_]*`?)*" +
	// 可选的排序方向（asc/desc，大小写不敏感）
	"(?:\\s+(?:asc|ASC|desc|DESC))?" +
	// 多个排序项的分隔符（逗号+可选空格）+ 重复的排序项
	")(?:,\\s*(?:`?[a-zA-Z_][a-zA-Z0-9_]*`?(?:\\.`?[a-zA-Z_][a-zA-Z0-9_]*`?)*" +
	"(?:\\s+(?:asc|ASC|desc|DESC))?)?)*$")
var validGroupRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*(\.[a-zA-Z_][a-zA-Z0-9_]*)?(,\s*[a-zA-Z_][a-zA-Z0-9_]*(\.[a-zA-Z_][a-zA-Z0-9_]*)?)*$`)
var validIncRegex = regexp.MustCompile(`^[a-zA-Z0-9_=?+\-\s]+(\.[a-zA-Z0-9_=?+\-\s]+)?$`)

// 校验表名（防止注入）
func isValidTable(s string) bool {
	if s == "" {
		return false
	}
	// 正则校验
	return validTableRegex.MatchString(strings.TrimSpace(s))
}

// 校验字段名是否为合法标识符（防止注入）
func isValidField(s string) bool {
	if s == "*" { // 通配符*允许
		return true
	}
	if s == "" {
		return false
	}
	// 多字段用逗号分隔，逐个校验
	fields := strings.Split(s, ",")
	for _, f := range fields {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		if validFieldRegex.MatchString(f) {
			return false
		}
	}
	return true
}

// 校验查询条件是否为合法标识符（防止注入）
func isValidWhere(s string) bool {
	if s == "" { // 空表达式合法（无WHERE子句）
		return true
	}
	// 预处理：移除多余空格，统一为大写（方便匹配）
	s = strings.TrimSpace(s)
	s = strings.ToUpper(s)
	// 正则校验
	return !validWhereRegex.MatchString(s)
}

// 校验order条件是否为合法标识符（防止注入）
func isValidOrder(s string) bool {
	if s == "" { // 空表达式合法（无WHERE子句）
		return true
	}
	s = regexp.MustCompile(`\s+`).ReplaceAllString(s, " ")
	s = strings.TrimSpace(s)
	// 正则校验
	return validOrderRegex.MatchString(s)
}

// 校验group条件是否为合法标识符（防止注入）
func isValidGroup(s string) bool {
	if s == "" { // 空表达式合法（无WHERE子句）
		return true
	}
	s = regexp.MustCompile(`\s+`).ReplaceAllString(s, " ")
	s = strings.TrimSpace(s)
	// 正则校验
	return validGroupRegex.MatchString(s)
}

// 校验表名/字段名是否为合法标识符（防止注入）
func isValidInc(s string) bool {
	if s == "*" { // 通配符*允许
		return true
	}
	if s == "" {
		return false
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
