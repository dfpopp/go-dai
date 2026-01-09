package mysql

import (
	"regexp"
	"strings"
)

var validTableRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*(\.[a-zA-Z_][a-zA-Z0-9_]*)?$`)
var validFieldRegex = regexp.MustCompile(`^(?:(?:COUNT|SUM|AVG|MIN|MAX|COUNT_DISTINCT|STDDEV|VARIANCE|MEDIAN|GROUP_CONCAT|STRING_AGG|DATE_TRUNC|DATE_PART|BIT_AND|BIT_OR|BIT_XOR)\((?:DISTINCT\s+)?(?:\*|[a-zA-Z_][a-zA-Z0-9_.]*)\)|(?:CONCAT|CONCAT_WS|TRIM|SUBSTRING|LOWER|UPPER|IF|COALESCE|ABS|ROUND|DATE_FORMAT)\((?:\s*(?:[a-zA-Z_][a-zA-Z0-9_.]*|\?)\s*,?)*\)|[a-zA-Z_][a-zA-Z0-9_.]*)(?:\s+AS\s+[a-zA-Z_][a-zA-Z0-9_]*)?$`)
var validWhereRegex = regexp.MustCompile(`^(?:\s*(?:\(\s*)?[a-zA-Z_][a-zA-Z0-9_.]*\s*[=!<>]=?\s*(?:\?|%[a-zA-Z0-9_%]*%)\s*(?:AND|OR|NOT)\s*(?:\)\s*)?)*\s*(?:\(\s*)?[a-zA-Z_][a-zA-Z0-9_.]*\s*[=!<>]=?\s*(?:\?|%[a-zA-Z0-9_%]*%)\s*(?:(?:AND|OR|NOT)\s*(?:\(\s*)?[a-zA-Z_][a-zA-Z0-9_.]*\s*[=!<>]=?\s*(?:\?|%[a-zA-Z0-9_%]*%)\s*(?:\)\s*)?)*\s*(?:\)\s*)?$`)
var validOrderRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*(\s+(asc|ASC|desc|DESC))?(,\s*[a-zA-Z_][a-zA-Z0-9_]*(\s+(asc|ASC|desc|DESC))?)*$`)
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
		if !validFieldRegex.MatchString(f) {
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
	s = regexp.MustCompile(`\s+`).ReplaceAllString(s, " ")
	s = strings.TrimSpace(s)
	// 正则校验
	return validWhereRegex.MatchString(s)
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
