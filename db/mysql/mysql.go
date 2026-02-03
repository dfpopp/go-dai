package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/dfpopp/go-dai/config"
	"github.com/dfpopp/go-dai/function"
	"github.com/dfpopp/go-dai/logger"
	"math"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// 该文件为mysql基本操作类，支持链式操作，在执行findAll()后必须调用ToString()才能返回想要的结果和错误信息
// 全局多数据库连接池
var multiDBPool sync.Map

type MysqlDb struct {
	Db             *sql.DB // 复用全局数据库连接池
	Tx             *sql.Tx
	DbPre          string //表前缀
	Table          string
	Alias          string
	WhereTemplates []string      // WHERE条件模板列表（如["id = ?", "status = ?"]）
	WhereArgs      []interface{} // WHERE条件参数列表（与模板一一对应）
	Order          string
	Group          string
	Field          string
	RelationList   []string
	Limit          string
	Data           []map[string]interface{}
	Err            error
}
type DbObj struct {
	Db  *sql.DB // 复用全局数据库连接池
	Pre string
}

// InitMySQL 初始化MySQL连接池
func InitMySQL() {
	cfgMap := config.GetMysqlConfig()
	for dbKey, cfg := range cfgMap {
		db, err := sql.Open("mysql", cfg.User+":"+cfg.Pwd+"@tcp("+cfg.Host+":"+cfg.Port+")/"+cfg.Dbname+"?charset="+cfg.Charset)
		if err != nil {
			logger.Error("MySQL连接失败: " + err.Error())
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
			db.SetConnMaxIdleTime(time.Duration(cfg.ConnMaxIdleTime) * time.Second) // 空闲连接超时时间（300秒无使用则关闭）
			db.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetime) * time.Second) // 连接最长存活时间;mysql default conn timeout=8h, should < mysql_timeout
			// 测试连接
			if err := db.Ping(); err != nil {
				logger.Error("MySQL Ping失败: " + err.Error())
			}
			multiDBPool.Store(dbKey, DbObj{Db: db, Pre: cfg.Pre})
		}
	}
}
func GetMysqlDB(dbKey string) (*MysqlDb, error) {
	val, ok := multiDBPool.Load(dbKey)
	if !ok {
		return nil, fmt.Errorf("数据库[%s]连接池未初始化", dbKey)
	}
	// 类型断言：将interface{}转为*sql.DB
	dbObj, ok := val.(DbObj)
	if !ok {
		return nil, fmt.Errorf("数据库[%s]连接池类型错误", dbKey)
	}
	return &MysqlDb{
		Db:             dbObj.Db,
		Tx:             nil,
		DbPre:          dbObj.Pre,
		Table:          "",
		Alias:          "",
		WhereTemplates: nil,
		WhereArgs:      nil,
		Order:          "",
		Group:          "",
		Field:          "",
		RelationList:   nil,
		Limit:          "",
		Data:           nil,
		Err:            nil,
	}, nil
}
func (db *MysqlDb) ToBegin() error {
	if db.Err != nil {
		return db.Err
	}
	if db.Tx != nil {
		return nil
	}
	if db.Db == nil {
		return errors.New("数据库连接未初始化")
	}
	tx, err := db.Db.Begin()
	if err != nil {
		return err
	}
	db.Tx = tx
	return nil
}
func (db *MysqlDb) Rollback() error {
	defer db.clearData(true)
	if db.Tx == nil {
		return errors.New("事务未开启")
	}
	err := db.Tx.Rollback()
	if err != nil {
		if !errors.Is(err, sql.ErrTxDone) {
			return errors.New("回滚事务失败:" + err.Error())
		}
	}
	return nil
}
func (db *MysqlDb) Commit() error {
	defer db.clearData(true)
	if db.Tx == nil {
		return errors.New("事务未开启")
	}
	err := db.Tx.Commit()
	if err != nil {
		return err
	}
	return nil
}
func (db *MysqlDb) SetTable(table string) *MysqlDb {
	db.Table = db.DbPre + table
	return db
}
func (db *MysqlDb) SetAlias(alias string) *MysqlDb {
	db.Alias = alias
	return db
}
func (db *MysqlDb) SetField(field string) *MysqlDb {
	db.Field = field
	return db
}
func (db *MysqlDb) SetWhere(tpl string, args ...interface{}) *MysqlDb {
	// 空值校验：模板为空则直接返回
	tpl = strings.TrimSpace(tpl)
	if tpl == "" {
		return db
	}

	// 非法关键字拦截（可选，增强安全，防止恶意注入）
	dangerousKeywords := []string{"DROP", "ALTER", "TRUNCATE", "DELETE", "INSERT", "UPDATE", "EXEC"}
	for _, kw := range dangerousKeywords {
		if strings.Contains(strings.ToUpper(tpl), kw) {
			db.Err = fmt.Errorf("条件模板包含非法关键字：%s", kw)
			return db
		}
	}
	// 将模板和参数加入列表
	db.WhereTemplates = append(db.WhereTemplates, tpl)
	db.WhereArgs = append(db.WhereArgs, args...)
	return db
}
func (db *MysqlDb) SetWhereOr(data map[string]interface{}) *MysqlDb {
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
func (db *MysqlDb) SetWhereIn(field string, args ...interface{}) *MysqlDb {
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
func (db *MysqlDb) SetOrder(order string) *MysqlDb {
	db.Order = order
	return db
}
func (db *MysqlDb) SetGroup(group string) *MysqlDb {
	db.Group = group
	return db
}
func (db *MysqlDb) SetJoin(tableName string, condition string, joinType string) *MysqlDb {
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
func (db *MysqlDb) SetLimit(skip int64, num int64) *MysqlDb {
	if skip < 0 {
		skip = 0
	}
	if num < 0 {
		num = 0
	}
	if num > 1000 {
		num = 1000
	}
	if skip == 0 {
		db.Limit = strconv.FormatInt(num, 10)
	} else {
		db.Limit = strconv.FormatInt(skip, 10) + "," + strconv.FormatInt(num, 10)
	}
	return db
}
func (db *MysqlDb) FindAll(ctx context.Context) *MysqlDb {
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
	var rows *sql.Rows
	var err error
	if db.Tx != nil {
		rows, err = db.Tx.QueryContext(ctx, sqlStr, db.WhereArgs...)
	} else {
		rows, err = db.Db.QueryContext(ctx, sqlStr, db.WhereArgs...)
	}
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
	// 14. 检查遍历过程中的错误
	if err := rows.Err(); err != nil {
		db.Err = fmt.Errorf("遍历结果集失败: %w", err)
		return db
	}
	db.Data = result
	return db
}
func (db *MysqlDb) FindCount(ctx context.Context) (int64, error) {
	defer db.clearData(false)
	if db.Db == nil {
		return 0, errors.New("数据库连接池未初始化（mysql.Db为nil）")
	}
	db.Field = "COUNT(*) AS count"
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
		// 数据库常见的计数返回类型：int64（MySQL默认）、string（部分驱动序列化）
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
			// 防止uint64溢出int64（MySQL COUNT(*)结果不会超过int64范围，此处做兜底）
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
func (db *MysqlDb) Find(ctx context.Context) (string, error) {
	defer db.clearData(false)
	if db.Db == nil {
		return "", errors.New("数据库连接池未初始化（mysql.Db为nil）")
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
func (db *MysqlDb) Insert(ctx context.Context, data map[string]interface{}) (int64, error) {
	defer db.clearData(false)
	if db.Db == nil {
		return 0, errors.New("数据库连接池未初始化（mysql.Db为nil）")
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
	var result sql.Result
	var err error
	if db.Tx != nil {
		result, err = db.Tx.ExecContext(ctx, sqlStr, values...)
	} else {
		result, err = db.Db.ExecContext(ctx, sqlStr, values...)
	}
	if err != nil {
		return 0, fmt.Errorf("执行插入SQL失败，SQL：%s，values:%s,错误：%w", sqlStr, function.Json_encode(values), err)
	}
	// 获取自增ID
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("获取自增ID失败：%w", err)
	}
	return id, nil
}
func (db *MysqlDb) InsertAll(ctx context.Context, dataList []map[string]interface{}) (int64, error) {
	defer db.clearData(false)
	if db.Db == nil {
		return 0, errors.New("数据库连接池未初始化（mysql.Db为nil）")
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
			// 若当前数据缺失该字段，插入NULL（也可选择报错，根据业务调整）
			value, ok := data[rawKey]
			if !ok {
				//singleValues = append(singleValues, nil)
				// 可选：严格模式，字段缺失直接报错
				return 0, fmt.Errorf("第%d条数据缺失字段[%s]", idx+1, rawKey)
			} else {
				singleValues = append(singleValues, value)
			}
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
	// 核心修正：提前声明result和err，解决作用域问题
	var result sql.Result
	var err error
	if db.Tx != nil {
		result, err = db.Tx.ExecContext(ctx, sqlStr, allValues...)
	} else {
		result, err = db.Db.ExecContext(ctx, sqlStr, allValues...)
	}
	if err != nil {
		return 0, fmt.Errorf("执行批量SQL失败，SQL：%s，values:%s,错误：%w", sqlStr, function.Json_encode(allValues), err)
	}
	// 获取受影响的行数（批量插入时，LastInsertId仅返回第一条数据的自增ID，需注意）
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("获取受影响行数失败：%w", err)
	}
	return rowsAffected, nil
}
func (db *MysqlDb) Update(ctx context.Context, data map[string]interface{}) (int64, error) {
	defer db.clearData(false)
	if db.Db == nil {
		return 0, errors.New("数据库连接池未初始化（mysql.Db为nil）")
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
	// 4. 拼接最终SQL
	sqlStr := fmt.Sprintf("UPDATE `%s` SET %s", db.Table, setSQL)
	if len(db.WhereTemplates) > 0 {
		sqlStr += " WHERE " + strings.Join(db.WhereTemplates, " AND ")
	}
	if len(db.WhereArgs) > 0 {
		for _, arg := range db.WhereArgs {
			values = append(values, arg)
		}
	}
	// 5. 执行SQL并处理错误
	var result sql.Result
	var err error
	if db.Tx != nil {
		result, err = db.Tx.ExecContext(ctx, sqlStr, values...)
	} else {
		result, err = db.Db.ExecContext(ctx, sqlStr, values...)
	}
	if err != nil {
		// 包装错误，保留原始错误链和SQL信息（便于调试）
		return 0, fmt.Errorf("执行更新SQL失败，SQL：%s，values:%s,错误：%w", sqlStr, function.Json_encode(values), err)
	}
	// 获取受影响的行数（批量插入时，LastInsertId仅返回第一条数据的自增ID，需注意）
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("获取受影响行数失败：%w", err)
	}
	return rowsAffected, nil
}
func (db *MysqlDb) UpdateBySet(ctx context.Context, setTpl string, values ...interface{}) (int64, error) {
	defer db.clearData(false)
	if db.Db == nil {
		return 0, errors.New("数据库连接池未初始化（mysql.Db为nil）")
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
	// 4. 拼接最终SQL
	sqlStr := fmt.Sprintf("UPDATE `%s` SET %s", db.Table, setTpl)
	if len(db.WhereTemplates) > 0 {
		sqlStr += " WHERE " + strings.Join(db.WhereTemplates, " AND ")
	}
	if len(db.WhereArgs) > 0 {
		for _, arg := range db.WhereArgs {
			values = append(values, arg)
		}
	}
	// 5. 执行SQL并处理错误
	var result sql.Result
	var err error
	if db.Tx != nil {
		result, err = db.Tx.ExecContext(ctx, sqlStr, values...)
	} else {
		result, err = db.Db.ExecContext(ctx, sqlStr, values...)
	}
	if err != nil {
		// 包装错误，保留原始错误链和SQL信息（便于调试）
		return 0, fmt.Errorf("执行更新SQL失败，SQL：%s，values:%s,错误：%w", sqlStr, function.Json_encode(values), err)
	}
	// 获取受影响的行数（批量插入时，LastInsertId仅返回第一条数据的自增ID，需注意）
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("获取受影响行数失败：%w", err)
	}
	return rowsAffected, nil
}

// SetInc 支持一次更新多个字段自增
func (db *MysqlDb) SetInc(ctx context.Context, tpl string, step ...int) (int64, error) {
	defer db.clearData(false)
	if db.Db == nil {
		return 0, errors.New("数据库连接池未初始化（mysql.Db为nil）")
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
	// 2. 构建SET子句：参数化赋值（如 `name`=?, `age`=?）
	var (
		values []interface{} // 存储所有参数值（SET + WHERE）
	)
	for _, value := range step {
		values = append(values, value)
	}
	setSQL := tpl
	// 4. 拼接最终SQL
	sqlStr := fmt.Sprintf("UPDATE `%s` SET %s", db.Table, setSQL)
	if len(db.WhereTemplates) > 0 {
		sqlStr += " WHERE " + strings.Join(db.WhereTemplates, " AND ")
	}
	if len(db.WhereArgs) > 0 {
		for _, arg := range db.WhereArgs {
			values = append(values, arg)
		}
	}
	// 5. 执行SQL并处理错误
	var result sql.Result
	var err error
	if db.Tx != nil {
		result, err = db.Tx.ExecContext(ctx, sqlStr, values...)
	} else {
		result, err = db.Db.ExecContext(ctx, sqlStr, values...)
	}
	if err != nil {
		// 包装错误，保留原始错误链和SQL信息（便于调试）
		return 0, fmt.Errorf("执行更新SQL失败，SQL：%s，values:%s,错误：%w", sqlStr, function.Json_encode(values), err)
	}
	// 获取受影响的行数（批量插入时，LastInsertId仅返回第一条数据的自增ID，需注意）
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("获取受影响行数失败：%w", err)
	}
	return rowsAffected, nil
}
func (db *MysqlDb) Delete(ctx context.Context) (int64, error) {
	defer db.clearData(false)
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
	sqlStr := "DELETE FROM " + db.Table
	if len(db.WhereTemplates) > 0 {
		for _, tpl := range db.WhereTemplates {
			if !isValidWhere(tpl) {
				return 0, fmt.Errorf("where子句[%s]格式非法，存在注入风险", tpl)
			}
		}
		sqlStr += " WHERE " + strings.Join(db.WhereTemplates, " AND ")
	}
	if db.Limit != "" {
		// 校验LIMIT格式（仅允许数字和逗号）
		sqlStr += " LIMIT " + db.Limit
	}
	var result sql.Result
	var err error
	if db.Tx != nil {
		result, err = db.Tx.ExecContext(ctx, sqlStr, db.WhereArgs...)
	} else {
		result, err = db.Db.ExecContext(ctx, sqlStr, db.WhereArgs...)
	}
	if err != nil {
		// 包装错误，保留原始错误链和SQL信息（便于调试）
		return 0, fmt.Errorf("执行更新SQL失败，SQL：%s，values:%s,错误：%w", sqlStr, function.Json_encode(db.WhereArgs), err)
	}
	// 获取受影响的行数（批量插入时，LastInsertId仅返回第一条数据的自增ID，需注意）
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("获取受影响行数失败：%w", err)
	}
	return rowsAffected, nil
}

// Exec 执行sql语句，该方法不要依赖用户提交数据，仅执行一些特殊的SQL语句保证sqlStr是绝对安全的，不存在注入等情况
func (db *MysqlDb) Exec(ctx context.Context, sqlStr string, values ...interface{}) (int64, error) {
	if db.Db == nil {
		return 0, errors.New("数据库连接池未初始化（mysql.Db为nil）")
	}
	// 空数据校验
	if len(sqlStr) == 0 {
		return 0, errors.New("Exec需要执行的SQL语句不能为空")
	}
	// 5. 执行SQL并处理错误
	var result sql.Result
	var err error
	if db.Tx != nil {
		result, err = db.Tx.ExecContext(ctx, sqlStr, values...)
	} else {
		result, err = db.Db.ExecContext(ctx, sqlStr, values...)
	}
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

// BatchUpdateCaseWhen 构建基于CASE WHEN的批量更新SQL和参数（安全、通用版）
// 功能：生成单SQL的批量更新语句，避免多次执行UPDATE，提升效率
// 参数：
//
//	table - 表名
//	pk - 主键字段名（如id、user_id）
//	fields - 需要更新的字段列表（排除主键）
//	datas - 批量更新数据，key=主键值，value=字段-值映射（支持任意类型：string/int/nil等）
//
// 返回：
//
//	sql - 生成的批量更新SQL语句（带?占位符）
//	args - SQL对应的参数列表（与占位符一一对应）
//	err - 错误信息（表名/字段非法、数据为空等）
func BatchUpdateCaseWhen(table string, pk string, fields []string, dataList map[string]interface{}) (string, error) {
	// 1. 基础合法性校验
	// 表名合法性
	if !isValidTable(table) {
		return "", fmt.Errorf("表名[%s]包含非法字符，仅允许字母、数字、下划线，且以字母开头", table)
	}
	// 主键合法性
	if !isValidField(pk) {
		return "", fmt.Errorf("主键[%s]包含非法字符，仅允许字母、数字、下划线，且以字母开头", pk)
	}
	// 字段合法性
	for _, field := range fields {
		if !isValidField(field) {
			return "", fmt.Errorf("字段[%s]包含非法字符，仅允许字母、数字、下划线，且以字母开头", field)
		}
	}
	// 数据非空校验
	if len(dataList) == 0 {
		return "", errors.New("批量更新数据不能为空")
	}
	// 字段列表非空校验
	if len(fields) == 0 {
		return "", errors.New("更新字段列表不能为空")
	}
	// 2. 拆分主键值和更新数据，去重并收集主键列表
	pkValues := make([]string, 0, len(dataList)) // 主键值列表（用于IN条件）
	caseClauses := make(map[string][]string)     // 每个字段对应的CASE WHEN子句
	args := make([]interface{}, 0)               // 参数列表（存储所有更新值）

	// 初始化每个字段的CASE WHEN子句容器
	for _, field := range fields {
		if field == pk {
			continue
		}
		caseClauses[field] = make([]string, 0)
	}
	// 3. 构建每个字段的CASE WHEN子句和参数
	for pkVal, fieldVals := range dataList {
		// 将fieldVals断言为map[string]interface{}（存储字段-值映射）
		fieldMap, ok := fieldVals.(map[string]interface{})
		if !ok {
			return "", fmt.Errorf("数据格式错误，主键[%s]的字段值必须是map[string]interface{}", pkVal)
		}
		pkValues = append(pkValues, "?") // 主键值用占位符，防止注入
		args = append(args, pkVal)       // 收集主键参数

		// 遍历每个需要更新的字段，构建WHEN子句
		for _, field := range fields {
			if field == pk {
				continue
			}
			// 获取字段值，无值则使用原字段值（ELSE已处理）
			val, exists := fieldMap[field]
			if !exists {
				continue
			}
			// 构建WHEN子句：WHEN ? THEN ?（两个占位符，分别对应主键和字段值）
			whenClause := "WHEN ? THEN ?"
			caseClauses[field] = append(caseClauses[field], whenClause)
			// 收集参数：先主键值，再字段值
			args = append(args, pkVal, val)
		}
	}
	// 4. 构建SET子句（核心：拼接每个字段的CASE WHEN）
	setClauses := make([]string, 0, len(caseClauses))
	for field, clauses := range caseClauses {
		if len(clauses) == 0 {
			continue
		}
		// 拼接字段的CASE WHEN完整子句：`field` = CASE `pk` WHEN ? THEN ? ... ELSE `field` END
		caseSQL := fmt.Sprintf("`%s` = CASE `%s` %s ELSE `%s` END",
			field, pk, strings.Join(clauses, " "), field)
		setClauses = append(setClauses, caseSQL)
	}
	if len(setClauses) == 0 {
		return "", errors.New("无有效更新字段，生成的SET子句为空")
	}
	// 5. 拼接最终SQL（WHERE条件使用主键IN，而非硬编码id）
	whereSQL := fmt.Sprintf("`%s` IN (%s)", pk, strings.Join(pkValues, ","))
	fullSQL := fmt.Sprintf("UPDATE `%s` SET %s WHERE %s", table, strings.Join(setClauses, ", "), whereSQL)
	return fullSQL, nil
}
func (db *MysqlDb) ToString() (string, error) {
	defer db.clearData(false)
	if db.Err != nil {
		return "", db.Err
	}
	if len(db.Data) == 0 {
		return "", nil
	}
	return function.Json_encode(db.Data), nil
}
func (db *MysqlDb) clearData(isClearTx bool) {
	db.Data = nil
	db.Table = ""
	db.Alias = ""
	db.WhereTemplates = nil
	db.WhereArgs = nil
	db.Order = ""
	db.Group = ""
	db.Field = ""
	db.RelationList = nil
	db.Limit = ""
	db.Err = nil
	if isClearTx {
		db.Tx = nil
	}
}

// CloseMysql 关闭所有 mysql 连接（供外部调用，如服务停止时）
func CloseMysql() error {
	var err error
	multiDBPool.Range(func(key, value interface{}) bool {
		dbObj, ok := value.(DbObj)
		if !ok {
			err = fmt.Errorf("无效的 mysql 客户端对象（key: %v）", key)
			return false // 终止遍历
		}
		// 关闭客户端（会释放连接池中的所有连接）
		if closeErr := dbObj.Db.Close(); closeErr != nil {
			err = fmt.Errorf("关闭 mysql 连接失败（dbKey: %v）: %w", key, closeErr)
			// 继续遍历，尝试关闭其他连接
			return true
		}
		fmt.Printf("mysql 连接已关闭（dbKey: %v）\n", key)
		return true
	})
	return err
}
