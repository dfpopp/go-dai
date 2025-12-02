package mysql

import (
	"regexp"
	"strings"
)

var validIdentifierRegex = regexp.MustCompile(`^[a-zA-Z0-9_\s]+(\.[a-zA-Z0-9_\s]+)?$`)

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
