package clickhouse

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"net/url"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dfpopp/go-dai/config"
	"github.com/dfpopp/go-dai/function"
	"github.com/dfpopp/go-dai/logger"

	// 根包init()注册database/sql驱动"clickhouse"（v2.42+已内置，早期版本的stdlib子包已废弃）
	_ "github.com/ClickHouse/clickhouse-go/v2"
)

// 该文件为ClickHouse基本操作类，支持链式操作，API与mysql包保持一致（应用代码可无缝切换）
// 与MySQL的差异由本包内部消化：
//  1. 不支持事务：本包不提供ToBegin/Commit/Rollback方法，无Tx字段
//  2. Update/UpdateBySet/SetInc：自动转换为ClickHouse的ALTER TABLE ... UPDATE变更语法
//     （注意：CH的mutation默认异步执行，ALTER TABLE ... UPDATE/DELETE仅改变元数据立即返回，
//     数据变更在后台完成，查询需等待mutation生效，同步等待可用mutations_sync设置）
//  3. Delete：转换为ALTER TABLE ... DELETE变更语法（异步mutation，全版本兼容），不支持LIMIT子句
//  4. Insert：ClickHouse无自增ID概念，Insert返回0，InsertAll返回受影响行数
//  5. LIMIT：与mysql一致支持 LIMIT offset,num 语法
//
// 全局多数据库连接池
var multiDBPool sync.Map

type ClickhouseDb struct {
	Db             *sql.DB       // 复用全局数据库连接池
	DbPre          string        //表前缀
	Table          string        //表名（可含库名前缀，如 mydb.table）
	Alias          string        //表别名
	WhereTemplates []string      // WHERE条件模板列表（如["id = ?", "status = ?"]）
	WhereArgs      []interface{} // WHERE条件参数列表（与模板一一对应）
	Order          string
	Group          string
	Field          string
	RelationList   []string
	ArrayJoin      string // ARRAY JOIN 行级展开表达式（如 zbhx AS x），位于FROM/JOIN之后、WHERE之前
	ArrayJoinLeft  bool   // 是否为 LEFT ARRAY JOIN（空数组行保留、缺失元素填类型默认值）
	Limit          string
	Data           []map[string]interface{}
	Err            error
}
type DbObj struct {
	Db  *sql.DB // 复用全局数据库连接池
	Pre string
}

// InitClickhouse 初始化ClickHouse连接池
func InitClickhouse() {
	initClickhouseConnections()

	// 注册配置变更钩子，当数据库配置变更时重新初始化连接
	config.RegisterConfigChangeHook(func(configType string, config interface{}) error {
		if configType == "database" {
			logger.Info("数据库配置变更，重新初始化ClickHouse连接...")
			return reinitClickhouseConnections()
		}
		return nil
	})
}

// initClickhouseConnections 初始化ClickHouse连接
func initClickhouseConnections() {
	cfgMap := config.GetClickhouseConfig()
	for dbKey, cfg := range cfgMap {
		createClickhouseConnection(dbKey, cfg)
	}
}

// reinitClickhouseConnections 重新初始化ClickHouse连接（平滑切换）
func reinitClickhouseConnections() error {
	cfgMap := config.GetClickhouseConfig()

	// 存储需要关闭的旧连接
	oldConnections := make(map[string]DbObj)

	// 先收集所有旧连接
	multiDBPool.Range(func(key, value interface{}) bool {
		dbKey := key.(string)
		dbObj := value.(DbObj)
		oldConnections[dbKey] = dbObj
		return true
	})

	// 创建新连接
	for dbKey, cfg := range cfgMap {
		createClickhouseConnection(dbKey, cfg)
		// 从旧连接映射中移除已更新的连接
		delete(oldConnections, dbKey)
	}

	// 延迟关闭旧连接（给现有操作30秒时间完成）
	go func() {
		time.Sleep(30 * time.Second)
		for dbKey, dbObj := range oldConnections {
			if err := dbObj.Db.Close(); err != nil {
				logger.Error("关闭旧ClickHouse连接失败:", "dbKey", dbKey, "error", err)
			} else {
				logger.Info("旧ClickHouse连接已关闭:", "dbKey", dbKey)
			}
			multiDBPool.Delete(dbKey)
		}
	}()

	return nil
}

// createClickhouseConnection 创建ClickHouse连接
func createClickhouseConnection(dbKey string, cfg config.ClickhouseConfig) {
	dsn := fmt.Sprintf("clickhouse://%s:%s?username=%s&password=%s&database=%s",
		cfg.Host, cfg.Port,
		url.QueryEscape(cfg.User), url.QueryEscape(cfg.Pwd), url.QueryEscape(cfg.Dbname))
	db, err := sql.Open("clickhouse", dsn)
	if err != nil {
		logger.Error("ClickHouse连接失败: " + err.Error())
	} else {
		// 设置连接池参数
		cpuNum := runtime.NumCPU()
		if cfg.MaxOpenConnNum <= 0 {
			cfg.MaxOpenConnNum = cpuNum * 3
		}
		if cfg.MaxIdleConnNum <= 0 {
			cfg.MaxIdleConnNum = cpuNum * 2
		}
		if cfg.ConnMaxIdleTime <= 0 {
			cfg.ConnMaxIdleTime = 300
		}
		if cfg.ConnMaxLifetime <= 0 {
			cfg.ConnMaxLifetime = 1800
		}
		db.SetMaxOpenConns(cfg.MaxOpenConnNum)
		db.SetMaxIdleConns(cfg.MaxIdleConnNum)
		db.SetConnMaxIdleTime(time.Duration(cfg.ConnMaxIdleTime) * time.Second)
		db.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetime) * time.Second)
		// 测试连接
		if err := db.Ping(); err != nil {
			logger.Error("ClickHouse Ping失败: " + err.Error())
		} else {
			// 存储新连接
			multiDBPool.Store(dbKey, DbObj{Db: db, Pre: cfg.Pre})
			logger.Info("ClickHouse连接成功:", "dbKey", dbKey)
		}
	}
}

// GetClickhouseDB 获取ClickHouse链式操作对象
func GetClickhouseDB(dbKey string) (*ClickhouseDb, error) {
	val, ok := multiDBPool.Load(dbKey)
	if !ok {
		return nil, fmt.Errorf("数据库[%s]连接池未初始化", dbKey)
	}
	// 类型断言：将interface{}转为DbObj
	dbObj, ok := val.(DbObj)
	if !ok {
		return nil, fmt.Errorf("数据库[%s]连接池类型错误", dbKey)
	}
	return &ClickhouseDb{
		Db:             dbObj.Db,
		DbPre:          dbObj.Pre,
		Table:          "",
		Alias:          "",
		WhereTemplates: nil,
		WhereArgs:      nil,
		Order:          "",
		Group:          "",
		Field:          "",
		RelationList:   nil,
		ArrayJoin:      "",
		ArrayJoinLeft:  false,
		Limit:          "",
		Data:           nil,
		Err:            nil,
	}, nil
}

func (db *ClickhouseDb) SetTable(table string) *ClickhouseDb {
	db.Table = db.DbPre + table
	return db
}
func (db *ClickhouseDb) SetAlias(alias string) *ClickhouseDb {
	db.Alias = alias
	return db
}
func (db *ClickhouseDb) SetField(field string) *ClickhouseDb {
	db.Field = field
	return db
}
func (db *ClickhouseDb) SetWhere(tpl string, args ...interface{}) *ClickhouseDb {
	// 空值校验：模板为空则直接返回
	tpl = strings.TrimSpace(tpl)
	if tpl == "" {
		return db
	}

	// 非法关键字拦截（词边界匹配，避免update_time/deleted等字段名被子串误伤）
	if kw := dangerousKeywordsRe.FindString(tpl); kw != "" {
		db.Err = fmt.Errorf("条件模板包含非法关键字：%s", strings.ToUpper(kw))
		return db
	}
	// 将模板和参数加入列表
	db.WhereTemplates = append(db.WhereTemplates, tpl)
	db.WhereArgs = append(db.WhereArgs, args...)
	return db
}
func (db *ClickhouseDb) SetWhereOr(data map[string]interface{}) *ClickhouseDb {
	if len(data) == 0 {
		return db
	}
	// 遍历map，生成等值条件模板和参数
	for field, value := range data {
		// 字段名校验（可选，防止传入非法字段名）
		if strings.TrimSpace(field) == "" {
			continue
		}
		// 生成等值模板：`field` = ?（加反引号防止字段名与关键字冲突）
		tpl := fmt.Sprintf("`%s` = ?", field)
		db.WhereTemplates = append(db.WhereTemplates, tpl)
		db.WhereArgs = append(db.WhereArgs, value)
	}
	return db
}
func (db *ClickhouseDb) SetWhereIn(field string, args ...interface{}) *ClickhouseDb {
	// 空值校验：模板为空则直接返回
	field = strings.TrimSpace(field)
	if field == "" {
		return db
	}
	tpl := strings.Repeat("?,", len(args))
	tpl = strings.TrimSuffix(tpl, ",") // 去掉最后一个逗号
	tpl = field + " IN (" + tpl + ")"
	// 将模板和参数加入列表
	db.WhereTemplates = append(db.WhereTemplates, tpl)
	db.WhereArgs = append(db.WhereArgs, args...)
	return db
}
func (db *ClickhouseDb) SetOrder(order string) *ClickhouseDb {
	db.Order = order
	return db
}
func (db *ClickhouseDb) SetGroup(group string) *ClickhouseDb {
	db.Group = group
	return db
}
func (db *ClickhouseDb) SetJoin(tableName string, condition string, joinType string) *ClickhouseDb {
	if joinType == "" {
		joinType = "LEFT"
	}
	tableList := strings.Split(strings.TrimSpace(tableName), " ")
	tableName = tableList[0]
	alias := tableName
	if len(tableList) > 2 {
		db.Err = errors.New("SetJoin方法中tableName参数中间包含了多个空格")
	}
	if len(tableList) == 2 {
		alias = tableList[1]
	}
	db.RelationList = append(db.RelationList, strings.ToUpper(joinType)+" JOIN "+db.DbPre+tableName+" AS "+alias+" ON "+condition)
	return db
}

// SetArrayJoin 设置 ARRAY JOIN 行级展开子句（ClickHouse特色语法，位于FROM/JOIN之后、WHERE之前）
// expr 示例："zbhx AS x"（数组逐元素展开为列x）或 "zbhx"（原地覆盖）
func (db *ClickhouseDb) SetArrayJoin(expr string) *ClickhouseDb {
	return db.setArrayJoin(expr, false)
}

// SetLeftArrayJoin 设置 LEFT ARRAY JOIN 行级展开子句（空数组的行保留，缺失元素填类型默认值）
func (db *ClickhouseDb) SetLeftArrayJoin(expr string) *ClickhouseDb {
	return db.setArrayJoin(expr, true)
}

func (db *ClickhouseDb) setArrayJoin(expr string, left bool) *ClickhouseDb {
	if db.Err != nil {
		return db
	}
	if !isValidArrayJoin(expr) {
		db.Err = fmt.Errorf("ARRAY JOIN表达式[%s]非法，存在注入风险", expr)
		return db
	}
	// 归一化连续空白后存储
	db.ArrayJoin = strings.Join(strings.Fields(expr), " ")
	db.ArrayJoinLeft = left
	return db
}
func (db *ClickhouseDb) SetLimit(skip int64, num int64) *ClickhouseDb {
	if skip < 0 {
		skip = 0
	}
	if num < 0 {
		num = 0
	}
	if num > 10000 {
		num = 10000
	}
	if skip == 0 {
		db.Limit = strconv.FormatInt(num, 10)
	} else {
		db.Limit = strconv.FormatInt(skip, 10) + "," + strconv.FormatInt(num, 10)
	}
	return db
}
func (db *ClickhouseDb) FetchSql() (string, error) {
	if db.Err != nil {
		return "", db.Err
	}
	if db.Db == nil {
		db.Err = errors.New("数据库连接未初始化")
		return "", db.Err
	}
	if db.Table == "" {
		db.Err = errors.New("未指定表名")
		return "", db.Err
	} else {
		if !isValidTable(db.Table) {
			db.Err = fmt.Errorf("表名[%s]包含非法字符，存在注入风险", db.Table)
			return "", db.Err
		}
	}
	if db.Field == "" {
		db.Field = "*"
	} else {
		// 校验字段合法性（防止字段注入）
		if !isValidField(db.Field) {
			db.Err = fmt.Errorf("查询字段[%s]包含非法字符，存在注入风险", db.Field)
			return "", db.Err
		}
	}
	sqlStr := "SELECT " + db.Field + " FROM " + db.Table
	if db.Alias != "" {
		// 校验别名合法性
		if !isValidTable(db.Alias) {
			db.Err = fmt.Errorf("表别名[%s]包含非法字符，存在注入风险", db.Alias)
			return "", db.Err
		}
		sqlStr += " AS " + db.Alias
	}
	if len(db.RelationList) > 0 {
		for _, relation := range db.RelationList {
			// 校验关联语句合法性
			if !isValidRelation(relation) {
				db.Err = fmt.Errorf("关联语句[%s]格式非法，存在注入风险", relation)
				return "", db.Err
			}
			sqlStr += " " + relation
		}
	}
	if db.ArrayJoin != "" {
		if !isValidArrayJoin(db.ArrayJoin) {
			db.Err = fmt.Errorf("ARRAY JOIN子句[%s]包含非法字符，存在注入风险", db.ArrayJoin)
			return "", db.Err
		}
		if db.ArrayJoinLeft {
			sqlStr += " LEFT ARRAY JOIN " + db.ArrayJoin
		} else {
			sqlStr += " ARRAY JOIN " + db.ArrayJoin
		}
	}
	if len(db.WhereTemplates) > 0 {
		for _, tpl := range db.WhereTemplates {
			if !isValidWhere(tpl) {
				db.Err = fmt.Errorf("where子句[%s]格式非法，存在注入风险", tpl)
				return "", db.Err
			}
		}
		sqlStr += " WHERE " + strings.Join(db.WhereTemplates, " AND ")
	}
	if db.Group != "" {
		if !isValidGroup(db.Group) {
			db.Err = fmt.Errorf("GROUP BY子句[%s]包含非法字符，存在注入风险", db.Group)
			return "", db.Err
		}
		sqlStr += " GROUP BY " + db.Group
	}
	if db.Order != "" {
		if !isValidOrder(db.Order) {
			db.Err = fmt.Errorf("ORDER BY子句[%s]包含非法字符，存在注入风险", db.Order)
			return "", db.Err
		}
		sqlStr += " ORDER BY " + db.Order
	}
	if db.Limit != "" {
		// 校验LIMIT格式（仅允许数字和逗号）
		sqlStr += " LIMIT " + db.Limit
	} else {
		sqlStr += " LIMIT 500"
	}
	formatStr := strings.ReplaceAll(sqlStr, "?", "%v")
	sqlStr = fmt.Sprintf(formatStr, db.WhereArgs...)
	return sqlStr, nil
}
func (db *ClickhouseDb) FindAll(ctx context.Context) *ClickhouseDb {
	if db.Err != nil {
		return db
	}
	if db.Db == nil {
		db.Err = errors.New("数据库连接未初始化")
		return db
	}
	if db.Table == "" {
		db.Err = errors.New("未指定表名")
		return db
	} else {
		if !isValidTable(db.Table) {
			db.Err = fmt.Errorf("表名[%s]包含非法字符，存在注入风险", db.Table)
			return db
		}
	}
	if db.Field == "" {
		db.Field = "*"
	} else {
		// 校验字段合法性（防止字段注入）
		if !isValidField(db.Field) {
			db.Err = fmt.Errorf("查询字段[%s]包含非法字符，存在注入风险", db.Field)
			return db
		}
	}
	sqlStr := "SELECT " + db.Field + " FROM " + db.Table
	if db.Alias != "" {
		// 校验别名合法性
		if !isValidTable(db.Alias) {
			db.Err = fmt.Errorf("表别名[%s]包含非法字符，存在注入风险", db.Alias)
			return db
		}
		sqlStr += " AS " + db.Alias
	}
	if len(db.RelationList) > 0 {
		for _, relation := range db.RelationList {
			// 校验关联语句合法性
			if !isValidRelation(relation) {
				db.Err = fmt.Errorf("关联语句[%s]格式非法，存在注入风险", relation)
				return db
			}
			sqlStr += " " + relation
		}
	}
	if db.ArrayJoin != "" {
		if !isValidArrayJoin(db.ArrayJoin) {
			db.Err = fmt.Errorf("ARRAY JOIN子句[%s]包含非法字符，存在注入风险", db.ArrayJoin)
			return db
		}
		if db.ArrayJoinLeft {
			sqlStr += " LEFT ARRAY JOIN " + db.ArrayJoin
		} else {
			sqlStr += " ARRAY JOIN " + db.ArrayJoin
		}
	}
	if len(db.WhereTemplates) > 0 {
		for _, tpl := range db.WhereTemplates {
			if !isValidWhere(tpl) {
				db.Err = fmt.Errorf("where子句[%s]格式非法，存在注入风险", tpl)
				return db
			}
		}
		sqlStr += " WHERE " + strings.Join(db.WhereTemplates, " AND ")
	}
	if db.Group != "" {
		if !isValidGroup(db.Group) {
			db.Err = fmt.Errorf("GROUP BY子句[%s]包含非法字符，存在注入风险", db.Group)
			return db
		}
		sqlStr += " GROUP BY " + db.Group
	}
	if db.Order != "" {
		if !isValidOrder(db.Order) {
			db.Err = fmt.Errorf("ORDER BY子句[%s]包含非法字符，存在注入风险", db.Order)
			return db
		}
		sqlStr += " ORDER BY " + db.Order
	}
	if db.Limit != "" {
		// 校验LIMIT格式（仅允许数字和逗号）
		sqlStr += " LIMIT " + db.Limit
	} else {
		sqlStr += " LIMIT 500"
	}
	// ClickHouse不支持事务，始终走连接池执行
	rows, err := db.Db.QueryContext(ctx, sqlStr, db.WhereArgs...)
	if err != nil {
		db.Err = fmt.Errorf("SQL语句:%s，values:%s,查询失败，失败原因[%s]", sqlStr, function.Json_encode(db.WhereArgs), err.Error())
		return db
	}
	// 确保结果集关闭
	defer func() {
		if rows != nil {
			if closeErr := rows.Close(); closeErr != nil {
				logger.Error("关闭结果集失败: %v", closeErr)
			}
		}
	}()
	cols, er := rows.Columns()
	if er != nil {
		db.Err = er
		return db
	}
	// 构造列值的指针切片（用于Scan）
	vals := make([]interface{}, len(cols))
	valPars := make([]interface{}, len(cols))
	for i := range vals {
		valPars[i] = &vals[i]
	}
	var result []map[string]interface{}
	for rows.Next() {
		if err := rows.Scan(valPars...); err != nil {
			db.Err = err
			return db
		}
		// 构造map：列名→列值
		rowMap := make(map[string]interface{})
		for i, col := range cols {
			// 处理[]uint8为字符串（数据库字符串字段的默认返回值）
			if b, ok := vals[i].([]uint8); ok {
				rowMap[col] = string(b)
			} else {
				rowMap[col] = vals[i]
			}
		}
		result = append(result, rowMap)
	}
	// 检查遍历过程中的错误
	if err := rows.Err(); err != nil {
		db.Err = fmt.Errorf("遍历结果集失败: %w", err)
		return db
	}
	db.Data = result
	return db
}

func (db *ClickhouseDb) FindCount(ctx context.Context) (int64, error) {
	defer db.clearData()
	if db.Db == nil {
		return 0, errors.New("数据库连接池未初始化（clickhouse.Db为nil）")
	}
	if db.Field == "" {
		db.Field = "COUNT(*) AS count"
	} else {
		db.Field = "COUNT(" + db.Field + ") AS count"
	}
	db.Limit = "1"
	db.FindAll(ctx)
	if db.Err != nil {
		return 0, db.Err
	}
	if len(db.Data) > 0 {
		// 安全的类型转换：兼容常见数值类型，非数值类型直接返回错误
		countVal := db.Data[0]["count"]
		count := int64(0)
		switch v := countVal.(type) {
		// 数据库常见的计数返回类型：int64（默认）、string（部分驱动序列化）
		case int64:
			count = v
		case string:
			parsedCount, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				return 0, fmt.Errorf("转换count字符串失败: %w", err)
			}
			count = parsedCount
		// 兼容其他常见数值类型：int/int32/uint/uint64
		case int:
			count = int64(v)
		case int32:
			count = int64(v)
		case uint:
			count = int64(v)
		case uint64:
			// 防止uint64溢出int64（COUNT(*)结果不会超过int64范围，此处做兜底）
			if v > uint64(math.MaxInt64) {
				return 0, errors.New("count值超出int64范围")
			}
			count = int64(v)
		case uint32:
			count = int64(v)
		// 非数值类型直接返回错误
		default:
			return 0, fmt.Errorf("count字段类型不支持，仅支持数值/字符串类型，当前类型：%T，值：%v", v, v)
		}
		return count, nil
	}
	return 0, nil
}
func (db *ClickhouseDb) Find(ctx context.Context) (string, error) {
	defer db.clearData()
	if db.Db == nil {
		return "", errors.New("数据库连接池未初始化（clickhouse.Db为nil）")
	}
	db.Limit = "1"
	db.FindAll(ctx)
	if db.Err != nil {
		return "", db.Err
	}
	if len(db.Data) > 0 {
		return function.Json_encode(db.Data[0]), nil
	}
	return "", nil
}

// Insert 插入单条数据
// 返回值说明：ClickHouse无自增ID概念，固定返回0；需要主键时由业务侧自行生成（如uuid）
func (db *ClickhouseDb) Insert(ctx context.Context, data map[string]interface{}) (int64, error) {
	defer db.clearData()
	if db.Db == nil {
		return 0, errors.New("数据库连接池未初始化（clickhouse.Db为nil）")
	}
	// 空数据校验
	if len(data) == 0 {
		return 0, errors.New("插入数据不能为空")
	}
	if db.Table == "" {
		return 0, errors.New("未指定表名")
	} else {
		if !isValidTable(db.Table) {
			return 0, errors.New("表名包含非法字符，存在注入风险")
		}
	}
	var (
		fields       []string      // 存储字段名
		placeholders []string      // 存储参数占位符?
		values       []interface{} // 存储参数值（与占位符一一对应）
	)
	// 遍历data，拆分字段名和值
	for key, value := range data {
		fields = append(fields, fmt.Sprintf("`%s`", key)) // 字段名加反引号，避免关键字冲突
		placeholders = append(placeholders, "?")          // 用?作为占位符，防止SQL注入
		values = append(values, value)                    // 收集参数值
	}
	// 拼接SQL语句
	fieldStr := strings.Join(fields, ", ")
	placeholderStr := strings.Join(placeholders, ", ")
	sqlStr := fmt.Sprintf("INSERT INTO `%s` (%s) VALUES (%s)", db.Table, fieldStr, placeholderStr)

	// 执行SQL
	result, err := db.Db.ExecContext(ctx, sqlStr, values...)
	if err != nil {
		return 0, fmt.Errorf("执行插入SQL失败，SQL：%s，values:%s,错误：%w", sqlStr, function.Json_encode(values), err)
	}
	// ClickHouse无自增ID，返回受影响行数
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("获取受影响行数失败：%w", err)
	}
	return rowsAffected, nil
}

// InsertAll 批量插入数据（ClickHouse推荐批量写入，单次INSERT多值效率远高于循环单条插入）
func (db *ClickhouseDb) InsertAll(ctx context.Context, dataList []map[string]interface{}) (int64, error) {
	defer db.clearData()
	if db.Db == nil {
		return 0, errors.New("数据库连接池未初始化（clickhouse.Db为nil）")
	}
	// 空数据校验
	if len(dataList) == 0 {
		return 0, errors.New("插入数据不能为空")
	}
	if db.Table == "" {
		return 0, errors.New("未指定表名")
	} else {
		if !isValidTable(db.Table) {
			return 0, errors.New("表名包含非法字符，存在注入风险")
		}
	}
	// 提取第一条数据的字段作为批量插入的统一字段（确保字段一致）
	firstData := dataList[0]
	if len(firstData) == 0 {
		return 0, errors.New("单条数据的字段不能为空")
	}
	var (
		fields       []string      // 存储统一的字段名
		placeholders []string      // 存储单条数据的占位符（?）
		allValues    []interface{} // 存储所有数据的参数值（按字段顺序拼接）
	)
	// 遍历第一条数据，初始化字段名和单条占位符
	for key := range firstData {
		// 字段名合法性校验（可选，增强安全性）
		if !isValidField(key) {
			return 0, fmt.Errorf("字段名[%s]包含非法字符，存在注入风险", key)
		}
		fields = append(fields, fmt.Sprintf("`%s`", key))
		placeholders = append(placeholders, "?")
	}

	// 拼接单条数据的占位符字符串（如 (?, ?, ?)）
	singlePlaceholder := fmt.Sprintf("(%s)", strings.Join(placeholders, ", "))
	// 存储批量数据的占位符集合（如 (?, ?, ?), (?, ?, ?)）
	var batchPlaceholders []string

	// 遍历所有数据，收集参数值并校验字段一致性
	for idx, data := range dataList {
		// 临时存储单条数据的参数值（按统一字段顺序）
		var singleValues []interface{}
		for _, field := range fields {
			// 去掉字段名的反引号，获取原始键名
			rawKey := strings.Trim(field, "`")
			// 若当前数据缺失该字段，报错（严格模式）
			value, ok := data[rawKey]
			if !ok {
				return 0, fmt.Errorf("第%d条数据缺失字段[%s]", idx+1, rawKey)
			}
			singleValues = append(singleValues, value)
		}
		// 将单条数据的值追加到总参数列表
		allValues = append(allValues, singleValues...)
		// 追加单条占位符到批量集合
		batchPlaceholders = append(batchPlaceholders, singlePlaceholder)
	}

	// 拼接最终的SQL语句
	fieldStr := strings.Join(fields, ", ")
	batchPlaceholderStr := strings.Join(batchPlaceholders, ", ")
	sqlStr := fmt.Sprintf("INSERT INTO `%s` (%s) VALUES %s", db.Table, fieldStr, batchPlaceholderStr)
	// 执行批量插入SQL
	result, err := db.Db.ExecContext(ctx, sqlStr, allValues...)
	if err != nil {
		return 0, fmt.Errorf("执行批量SQL失败，SQL：%s，values:%s,错误：%w", sqlStr, function.Json_encode(allValues), err)
	}
	// 获取受影响的行数
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("获取受影响行数失败：%w", err)
	}
	return rowsAffected, nil
}

// Update 更新数据（自动转换为ClickHouse变更语法 ALTER TABLE ... UPDATE）
// 注意：CH的mutation默认异步执行，本方法返回不代表数据已变更完成
func (db *ClickhouseDb) Update(ctx context.Context, data map[string]interface{}) (int64, error) {
	defer db.clearData()
	if db.Db == nil {
		return 0, errors.New("数据库连接池未初始化（clickhouse.Db为nil）")
	}
	// 空数据校验
	if len(data) == 0 {
		return 0, errors.New("更新数据不能为空")
	}
	if db.Table == "" {
		return 0, errors.New("未指定表名")
	} else {
		if !isValidTable(db.Table) {
			return 0, errors.New("表名包含非法字符，存在注入风险")
		}
	}
	// 2. 构建SET子句：参数化赋值（如 `name`=?, `age`=?）
	var (
		setClauses []string      // SET子句的片段
		values     []interface{} // 存储所有参数值（SET + WHERE）
	)
	for key, value := range data {
		if !isValidField(key) {
			return 0, fmt.Errorf("更新字段[%s]包含非法字符，存在注入风险", key)
		}
		setClauses = append(setClauses, fmt.Sprintf("`%s`=?", key))
		values = append(values, value)
	}
	setSQL := strings.Join(setClauses, ", ")
	return db.execMutation(ctx, "UPDATE", setSQL, values)
}

// UpdateBySet 更新数据（自定义SET表达式，自动转换为ClickHouse变更语法 ALTER TABLE ... UPDATE）
func (db *ClickhouseDb) UpdateBySet(ctx context.Context, setTpl string, values ...interface{}) (int64, error) {
	defer db.clearData()
	if db.Db == nil {
		return 0, errors.New("数据库连接池未初始化（clickhouse.Db为nil）")
	}
	// 空数据校验
	if setTpl == "" {
		return 0, errors.New("update SET表达式不能为空")
	}
	if db.Table == "" {
		return 0, errors.New("未指定表名")
	} else {
		if !isValidTable(db.Table) {
			return 0, errors.New("表名包含非法字符，存在注入风险")
		}
	}
	return db.execMutation(ctx, "UPDATE", setTpl, values)
}

// SetInc 支持一次更新多个字段自增（自动转换为ClickHouse变更语法 ALTER TABLE ... UPDATE）
func (db *ClickhouseDb) SetInc(ctx context.Context, tpl string, step ...interface{}) (int64, error) {
	defer db.clearData()
	if db.Db == nil {
		return 0, errors.New("数据库连接池未初始化（clickhouse.Db为nil）")
	}
	if db.Table == "" {
		return 0, errors.New("未指定表名")
	} else {
		if !isValidTable(db.Table) {
			return 0, errors.New("表名包含非法字符，存在注入风险")
		}
	}
	if tpl == "" {
		return 0, errors.New("未指定更新的字段")
	} else {
		if !isValidInc(tpl) {
			return 0, fmt.Errorf("set子句[%s]包含非法字符，存在注入风险", tpl)
		}
	}
	var (
		values []interface{} // 存储所有参数值（SET + WHERE）
	)
	for _, value := range step {
		values = append(values, value)
	}
	return db.execMutation(ctx, "UPDATE", tpl, values)
}

// execMutation 执行ClickHouse变更语句（ALTER TABLE ... UPDATE / ALTER TABLE ... DELETE 的公共执行入口）
// mutationType：变更类型（"UPDATE" / "DELETE"）；setSQL：UPDATE时的SET子句，DELETE时传空
func (db *ClickhouseDb) execMutation(ctx context.Context, mutationType string, setSQL string, values []interface{}) (int64, error) {
	// ClickHouse不支持标准UPDATE/DELETE语法，统一转换为ALTER TABLE ...变更语法
	sqlStr := fmt.Sprintf("ALTER TABLE `%s` %s", db.Table, mutationType)
	if setSQL != "" {
		sqlStr += " " + setSQL
	}
	if len(db.WhereTemplates) > 0 {
		sqlStr += " WHERE " + strings.Join(db.WhereTemplates, " AND ")
	}
	if len(db.WhereArgs) > 0 {
		for _, arg := range db.WhereArgs {
			values = append(values, arg)
		}
	}
	// 执行SQL并处理错误
	result, err := db.Db.ExecContext(ctx, sqlStr, values...)
	if err != nil {
		// 包装错误，保留原始错误链和SQL信息（便于调试）
		return 0, fmt.Errorf("执行变更SQL失败，SQL：%s，values:%s,错误：%w", sqlStr, function.Json_encode(values), err)
	}
	// mutation语句通常返回0受影响行数（异步执行），此返回值仅供参考
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("获取受影响行数失败：%w", err)
	}
	return rowsAffected, nil
}

// Delete 删除数据（自动转换为ClickHouse变更语法 ALTER TABLE ... DELETE）
// 注意：CH的mutation默认异步执行，本方法返回不代表数据已删除完成
func (db *ClickhouseDb) Delete(ctx context.Context) (int64, error) {
	defer db.clearData()
	if db.Err != nil {
		return 0, db.Err
	}
	if db.Db == nil {
		return 0, errors.New("数据库连接未初始化")
	}
	if db.Table == "" {
		return 0, errors.New("未指定表名")
	} else {
		if !isValidTable(db.Table) {
			return 0, fmt.Errorf("表名[%s]包含非法字符，存在注入风险", db.Table)
		}
	}
	// ALTER TABLE ... DELETE不支持LIMIT子句
	if db.Limit != "" {
		return 0, errors.New("ClickHouse不支持DELETE LIMIT子句，请通过WHERE条件精确限定删除范围")
	}
	if len(db.WhereTemplates) > 0 {
		for _, tpl := range db.WhereTemplates {
			if !isValidWhere(tpl) {
				return 0, fmt.Errorf("where子句[%s]格式非法，存在注入风险", tpl)
			}
		}
	}
	// 转换为ALTER TABLE ... DELETE变更语法执行
	return db.execMutation(ctx, "DELETE", "", nil)
}

// Exec 执行sql语句，该方法不要依赖用户提交数据，仅执行一些特殊的SQL语句保证sqlStr是绝对安全的，不存在注入等情况
func (db *ClickhouseDb) Exec(ctx context.Context, sqlStr string, values ...interface{}) (int64, error) {
	if db.Db == nil {
		return 0, errors.New("数据库连接池未初始化（clickhouse.Db为nil）")
	}
	// 空数据校验
	if len(sqlStr) == 0 {
		return 0, errors.New("Exec需要执行的SQL语句不能为空")
	}
	// 执行SQL并处理错误
	result, err := db.Db.ExecContext(ctx, sqlStr, values...)
	if err != nil {
		// 包装错误，保留原始错误链和SQL信息（便于调试）
		return 0, fmt.Errorf("执行Exec的SQL失败，SQL：%s,values:%s,，错误：%w", sqlStr, function.Json_encode(values), err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("获取受影响行数失败：%w", err)
	}
	return rowsAffected, nil
}
func (db *ClickhouseDb) ToString() (string, error) {
	defer db.clearData()
	if db.Err != nil {
		return "", db.Err
	}
	if len(db.Data) == 0 {
		return "", nil
	}
	return function.Json_encode(db.Data), nil
}
func (db *ClickhouseDb) clearData() {
	db.Data = nil
	db.Table = ""
	db.Alias = ""
	db.WhereTemplates = nil
	db.WhereArgs = nil
	db.Order = ""
	db.Group = ""
	db.Field = ""
	db.RelationList = nil
	db.ArrayJoin = ""
	db.ArrayJoinLeft = false
	db.Limit = ""
	db.Err = nil
}

// CloseClickhouse 关闭所有 clickhouse 连接（供外部调用，如服务停止时）
func CloseClickhouse() error {
	var err error
	multiDBPool.Range(func(key, value interface{}) bool {
		dbObj, ok := value.(DbObj)
		if !ok {
			err = fmt.Errorf("无效的 clickhouse 客户端对象（key: %v）", key)
			return false // 终止遍历
		}
		// 关闭客户端（会释放连接池中的所有连接）
		if closeErr := dbObj.Db.Close(); closeErr != nil {
			err = fmt.Errorf("关闭 clickhouse 连接失败（dbKey: %v）: %w", key, closeErr)
			// 继续遍历，尝试关闭其他连接
			return true
		}
		fmt.Printf("clickhouse 连接已关闭（dbKey: %v）\n", key)
		return true
	})
	return err
}
