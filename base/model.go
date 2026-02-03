package base

import (
	"github.com/dfpopp/go-dai/db/elasticSearch"
	"github.com/dfpopp/go-dai/db/mongoDb"
	"github.com/dfpopp/go-dai/db/mysql"
	"github.com/dfpopp/go-dai/db/redisDb"
	"github.com/dfpopp/go-dai/logger"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"strings"
)

type BaseModel struct {
	log logger.Logger // 日志实例
}

// Init 初始化服务层（框架自动调用）
func (m *BaseModel) Init() {
	m.log = logger.GetLogger()
}

func (m *BaseModel) GetMysqlDb(DbTag string) (*mysql.MysqlDb, error) {
	return mysql.GetMysqlDB(DbTag)
}
func (m *BaseModel) GetMongoDb(DbTag string) (*mongoDb.Db, error) {
	return mongoDb.GetMongoDB(DbTag)
}
func (m *BaseModel) GetESDb(DbTag string) (*elasticSearch.ESDb, error) {
	return elasticSearch.GetEsDB(DbTag)
}
func (m *BaseModel) GetRedis(DbTag string) (*redisDb.RedisDb, error) {
	return redisDb.GetRedisDB(DbTag)
}
func (m *BaseModel) StrIdToObjectID(id string) (primitive.ObjectID, error) {
	return mongoDb.IdToObjectID(id)
}
func (m *BaseModel) ObjectIDToString(id primitive.ObjectID) string {
	return mongoDb.ObjectIDToString(id)
}
func (m *BaseModel) MapToBsonD(data map[string]interface{}) bson.D {
	return mongoDb.MapToBsonD(data)
}

// LogInfo 记录服务层信息日志
func (m *BaseModel) LogInfo(content ...interface{}) {
	m.log.Info(content...)
}

// LogError 记录服务层错误日志
func (m *BaseModel) LogError(content ...interface{}) {
	m.log.Error(content...)
}

// LogWarn 记录服务层错误日志
func (m *BaseModel) LogWarn(content ...interface{}) {
	m.log.Warn(content...)
}

// StringToFulltextIndexStr 生成仅含双字符段的全文索引字符串（解决错位匹配）
// 核心规则：
//  1. 过滤仅保留数字(0-9)、字母(a-z/A-Z)、常用汉字(0x4E00-0x9FA5)；
//  2. 对所有类型（数字/字母/汉字）生成两种双字符分组：
//     - 分组1：从第0位开始，步长2（如13838→13 83）；
//     - 分组2：从第1位开始，步长2（如13838→38 3）→ 仅保留双字符段，丢弃末尾单字符；
//  3. 最终仅保留双字符段，完全丢弃所有单字符；
//  4. 所有双字符段去重后拼接，避免重复关键词；
//
// 示例：
//
//	输入："13838385687" → 分组1（0起始）：13 83 83 85 68 → 分组2（1起始）：38 38 38 56 87 → 最终：13 83 83 85 68 38 38 38 56 87；
//	输入："张三李四" → 分组1：张三 李四 → 分组2：三李 → 最终：张三 李四 三李；
//	输入："Abc12" → 分组1：Ab 12 → 分组2：bc → 最终：Ab 12 bc；
func (m *BaseModel) StringToFulltextIndexStr(input string) string {
	// 1. 转换为rune切片，兼容UTF8字符（中文/特殊字符）
	runes := []rune(input)
	if len(runes) == 0 {
		return ""
	}

	// 2. 筛选仅保留数字、字母、常用汉字，按类型分类
	var charRunes []rune
	for _, r := range runes {
		switch {
		// 数字（0-9）
		case r >= '0' && r <= '9':
			charRunes = append(charRunes, r)
		// 字母（大小写）
		case (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z'):
			charRunes = append(charRunes, r)
		// 常用汉字（Unicode基本区：0x4E00-0x9FA5）
		case r >= 0x4E00 && r <= 0x9FA5:
			charRunes = append(charRunes, r)
		// 非目标字符直接丢弃
		default:
			continue
		}
	}

	// 3. 通用函数：生成指定起始位置的双字符段（仅保留双字符，丢弃单字符）
	genDoubleCharSegments := func(rs []rune, start int) []string {
		var segments []string
		// 从start开始，步长2遍历，仅保留双字符段
		for i := start; i+1 < len(rs); i += 2 {
			// 拼接双字符
			seg := string([]rune{rs[i], rs[i+1]})
			segments = append(segments, seg)
		}
		return segments
	}

	// 4. 对每个类型生成两种错位分组（0起始+1起始），仅保留双字符段
	var allSegments []string
	// 处理字母
	charSeg0 := genDoubleCharSegments(charRunes, 0)
	charSeg1 := genDoubleCharSegments(charRunes, 1)
	allSegments = append(allSegments, charSeg0...)
	allSegments = append(allSegments, charSeg1...)
	// 5. 去重（避免重复关键词，减少索引体积）
	uniqueSegments := make(map[string]struct{})
	var finalSegments []string
	for _, seg := range allSegments {
		if _, exists := uniqueSegments[seg]; !exists {
			uniqueSegments[seg] = struct{}{}
			finalSegments = append(finalSegments, seg)
		}
	}

	// 6. 拼接为空格分隔的字符串（无末尾空格）
	return strings.Join(finalSegments, " ")
}

// StringToSearchFulltextStr 处理用户搜索输入，转换为适配MongoDB全文索引的纯双字符段（业务层专用）
// 核心规则（和存储层逻辑严格对齐）：
//  1. 过滤仅保留数字(0-9)、字母(a-z/A-Z)、常用汉字(0x4E00-0x9FA5)；
//  2. 仅保留双字符段，彻底丢弃所有单字符；
//  3. 双字符段去重，避免重复关键词干扰查询；
//  4. 不同类型（字母/数字/汉字）的双字符段用空格分隔，无末尾冗余空格；
//
// 示例：
//
//	输入："Ab123张三" → 过滤后：字母[Ab]、数字[12]、汉字[张三] → 输出："Ab 12 张三"；
//	输入："13838385687" → 过滤后：数字[13838385687] → 输出："13 83 83 85 68"；
//	输入："张1" → 仅单字符/单双混合（无完整双字符）→ 输出：""；
//	输入："!!38张三##" → 过滤后：数字[38]、汉字[张三] → 输出："38 张三"；
func (m *BaseModel) StringToSearchFulltextStr(input string) string {
	// 1. 转换为rune切片，兼容UTF8字符（中文/特殊字符）
	runes := []rune(input)
	if len(runes) == 0 {
		return ""
	}

	// 2. 筛选仅保留数字、字母、常用汉字，按类型分类
	var charRunes []rune //
	for _, r := range runes {
		switch {
		// 数字（0-9）
		case r >= '0' && r <= '9':
			charRunes = append(charRunes, r)
		// 字母（大小写）
		case (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z'):
			charRunes = append(charRunes, r)
		// 常用汉字（Unicode基本区：0x4E00-0x9FA5）
		case r >= 0x4E00 && r <= 0x9FA5:
			charRunes = append(charRunes, r)
		// 非目标字符直接丢弃
		default:
			continue
		}
	}

	// 3. 通用函数：生成纯双字符段（仅保留完整双字符，丢弃单字符）
	genPureDoubleSegments := func(rs []rune) []string {
		var segments []string
		// 仅遍历完整双字符，i+1 < len(rs) 确保无单字符
		for i := 0; i+1 < len(rs); i += 2 {
			seg := string([]rune{rs[i], rs[i+1]})
			segments = append(segments, seg)
		}
		return segments
	}

	// 4. 生成各类型的纯双字符段
	var allSegments []string
	// 双字符段
	charSegs := genPureDoubleSegments(charRunes)
	allSegments = append(allSegments, charSegs...)
	// 5. 去重（避免重复关键词，和存储层去重逻辑对齐）
	uniqueSegs := make(map[string]struct{})
	var finalSegs []string
	for _, seg := range allSegments {
		if _, exists := uniqueSegs[seg]; !exists {
			uniqueSegs[seg] = struct{}{}
			finalSegs = append(finalSegs, seg)
		}
	}
	// 6. 拼接为空格分隔的字符串（无末尾空格）
	return strings.Join(finalSegs, " ")
}
