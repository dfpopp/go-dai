package elasticSearch

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/dfpopp/go-dai/config"
	"github.com/dfpopp/go-dai/function"
	"github.com/dfpopp/go-dai/logger"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"io"
	"net"
	"net/http"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"
)

// 该文件为ES基本操作类，支持链式操作
// 全局多数据库连接池
var multiESPool sync.Map

const (
	BoolMust    BoolClauseType = "must"
	BoolShould  BoolClauseType = "should"
	BoolMustNot BoolClauseType = "must_not"
	BoolFilter  BoolClauseType = "filter"
)

// InitEs 初始化MySQL连接池
func InitEs() {
	cfgMap := config.GetEsConfig()
	for dbKey, cfg := range cfgMap {
		client, transport, err := connect(cfg)
		if err != nil {
			logger.Error(fmt.Sprintf("ES连接初始化失败（%s）: %v", dbKey, err))
		} else {
			multiESPool.Store(dbKey, DbObj{Client: client, Transport: transport, Pre: cfg.Pre, GzipStatus: cfg.GzipStatus})
		}
	}
}

// connect 建立MongoDB连接
func connect(cfg config.EsConfig) (*elasticsearch.Client, *http.Transport, error) {
	// 默认配置
	if cfg.Host == "" {
		cfg.Host = "localhost"
	}
	if cfg.Port == "" {
		cfg.Port = "9200"
	}
	// 2. 规范地址拼接（处理Host带协议的情况）
	var address string
	if strings.HasPrefix(cfg.Host, "http://") || strings.HasPrefix(cfg.Host, "https://") {
		address = fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)
	} else {
		// 默认补http（生产建议显式配置https）
		address = fmt.Sprintf("http://%s:%s", cfg.Host, cfg.Port)
	}
	// 3. 配置TLS（HTTPS支持）
	tlsConfig := &tls.Config{}
	if cfg.InsecureTLS {
		tlsConfig.InsecureSkipVerify = true // 测试环境跳过证书验证
	}
	cpuNum := runtime.NumCPU()
	if cfg.MaxIdleConnNum == 0 {
		cfg.MaxIdleConnNum = cpuNum * 10
	}
	if cfg.MaxIdleConnNumPerHost == 0 {
		cfg.MaxIdleConnNumPerHost = cpuNum * 10
	}
	if cfg.MaxConnNumPerHost == 0 {
		cfg.MaxConnNumPerHost = cpuNum * 15
	}
	if cfg.IdleConnTimeout == 0 {
		cfg.IdleConnTimeout = 300 //5分钟
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 5
	}
	if cfg.KeepAlive == 0 {
		cfg.KeepAlive = 30
	}
	if cfg.ResponseHeaderTimeout == 0 {
		cfg.ResponseHeaderTimeout = 15
	}
	if cfg.TLSHandshakeTimeout == 0 {
		cfg.TLSHandshakeTimeout = 5
	}
	// 4. 自定义HTTP Transport（核心：连接池+超时配置）
	transport := &http.Transport{
		// 连接池配置
		MaxIdleConns:        cfg.MaxIdleConnNum,                               // 全局最大空闲连接
		MaxIdleConnsPerHost: cfg.MaxIdleConnNumPerHost,                        // 每个主机最大空闲连接
		IdleConnTimeout:     time.Duration(cfg.IdleConnTimeout) * time.Second, // 空闲连接超时释放
		MaxConnsPerHost:     cfg.MaxConnNumPerHost,                            // 每个主机最大并发连接（限制并发）
		// 超时配置
		DialContext: (&net.Dialer{
			Timeout:   time.Duration(cfg.Timeout) * time.Second,   // 连接建立超时（TCP握手）
			KeepAlive: time.Duration(cfg.KeepAlive) * time.Second, // 长连接保活
		}).DialContext,
		ResponseHeaderTimeout: time.Duration(cfg.ResponseHeaderTimeout) * time.Second, // 响应头超时
		TLSClientConfig:       tlsConfig,                                              // TLS配置
		TLSHandshakeTimeout:   time.Duration(cfg.TLSHandshakeTimeout) * time.Second,   // TLS握手超时
	}
	header := http.Header{}
	if cfg.GzipStatus {
		header.Add("Accept-Encoding", "gzip")
	}
	// 5. 构建ES客户端配置
	esCfg := elasticsearch.Config{
		Addresses: []string{address},
		Username:  cfg.User,
		Password:  cfg.Pwd,
		// 自定义HTTP客户端（包含连接池+超时）
		Transport: transport,
		// 请求头配置
		Header: header,
		// 重试配置（可选，根据业务调整）
		RetryOnStatus: []int{502, 503, 504, 429},                                                      // 重试的状态码
		RetryBackoff:  func(i int) time.Duration { return time.Duration(i) * 100 * time.Millisecond }, // 退避策略
		MaxRetries:    3,                                                                              // 最大重试次数
	}

	// 6. 创建ES客户端
	client, err := elasticsearch.NewClient(esCfg)
	if err != nil {
		return nil, nil, err
	}

	// 7. 健康检查（验证连接是否有效）
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res, err := client.Info(
		client.Info.WithContext(ctx),
	)
	if err != nil {
		return nil, nil, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logger.Error(fmt.Sprintf("ES(dbKey:%s) 关闭body失败：%v", address, err))
		}
	}(res.Body)
	body, err := DeZip(cfg.GzipStatus, res)
	if res.IsError() {
		return nil, nil, fmt.Errorf("ES健康检查失败：%s", string(body))
	}
	return client, transport, nil
}
func GetEsDB(dbKey string) (*ESDb, error) {
	val, ok := multiESPool.Load(dbKey)
	if !ok {
		return nil, fmt.Errorf("数据库[%s]连接池未初始化", dbKey)
	}
	// 类型断言：将interface{}转为*sql.DB
	dbObj, ok := val.(DbObj)
	if !ok {
		return nil, fmt.Errorf("数据库[%s]连接池类型错误", dbKey)
	}
	return &ESDb{
		Client:        dbObj.Client,
		DbPre:         dbObj.Pre,
		GzipStatus:    dbObj.GzipStatus,
		Index:         []string{},
		Id:            "",
		WhereQuery:    nil,
		Aggs:          nil,
		Sort:          []string{},
		ExcludeSource: []string{},
		Source:        []string{},
		ScriptFields:  nil,
		From:          int64(0),
		Size:          int64(0),
		Highlight:     nil,
		Pk:            "",
		BatchTimeout:  0,
		BulkActions:   nil,
		Data:          nil,
		Err:           nil,
	}, nil
}
func (db *ESDb) SetIndex(tables string) *ESDb {
	if db.Err != nil {
		return db
	}
	tableList := strings.Split(tables, ",")
	for k, v := range tableList {
		if len(db.DbPre+v) > 255 {
			db.Err = fmt.Errorf("索引名[%s]拼接前缀后超长（最大255字符）", db.DbPre+v)
			return db
		}
		if isValidIndexName(v) == false {
			db.Err = fmt.Errorf("索引名[%s]非法，仅支持小写字母、数字、下划线、连字符，且以字母/数字开头", v)
			return db
		}
		tableList[k] = db.DbPre + v
	}
	db.Index = tableList
	return db
}

// SetId 设置要批量操作的文档ID
func (db *ESDb) SetId(id string) *ESDb {
	if db.Err != nil {
		return db
	}
	db.Id = id
	return db
}

// SetPk 设置批量操作的主键字段（如"id"）
func (db *ESDb) SetPk(pk string) *ESDb {
	if db.Err != nil {
		return db
	}
	if !validIdentifierRegex.MatchString(pk) {
		db.Err = fmt.Errorf("主键字段[%s]非法", pk)
		return db
	}
	db.Pk = pk
	return db
}

// SetBatchTimeout 设置批量操作的超时时间；控制 ES 服务端处理Bulk、batch（批量操作） 请求的最大时间（仅集群内部执行阶段）
func (db *ESDb) SetBatchTimeout(timeout int) *ESDb {
	if db.Err != nil {
		return db
	}
	db.BatchTimeout = timeout
	return db
}

// SetExcludeSource 设置返回的排除字段（_source.exclude）
func (db *ESDb) SetExcludeSource(fields ...string) *ESDb {
	if db.Err != nil {
		return db
	}
	for _, field := range fields {
		if !validIdentifierRegex.MatchString(field) {
			db.Err = fmt.Errorf("返回字段[%s]非法", field)
			return db
		}
		db.ExcludeSource = append(db.ExcludeSource, field)
	}
	return db
}

// SetSource 设置返回字段（_source）
func (db *ESDb) SetSource(fields ...string) *ESDb {
	if db.Err != nil {
		return db
	}
	for _, field := range fields {
		if !validIdentifierRegex.MatchString(field) {
			db.Err = fmt.Errorf("返回字段[%s]非法", field)
			return db
		}
		db.Source = append(db.Source, field)
	}
	return db
}

// SetWhere 优化版：支持叠加bool子条件，或设置单一查询类型
// 用法1（叠加bool子条件）：SetWhere(BoolMust, map[string]interface{}{"term": {"status": "active"}})
// 用法2（设置单一查询）：SetWhere("range", map[string]interface{}{"age": {"gte": 18}})
func (db *ESDb) SetWhere(clause interface{}, query interface{}) *ESDb {
	// 1. 链式错误传递：已有错误直接返回
	if db.Err != nil {
		return db
	}

	// 2. 基础参数校验：query 不能为空
	if query == nil {
		db.Err = errors.New("查询条件(query)不能为空")
		return db
	}
	// 校验 query 为有效 map（ES 查询条件必须是 map 结构）
	_, isMap := query.(map[string]interface{})
	if !isMap {
		db.Err = errors.New("查询条件(query)必须为map[string]interface{}类型")
		return db
	}

	// 3. 初始化WhereQuery（确保基础结构存在）
	if db.WhereQuery == nil {
		db.WhereQuery = make(map[string]interface{})
	}

	// 4. 区分参数类型：bool子句（must/should等） vs 单一查询类型（term/range等）
	switch c := clause.(type) {
	case BoolClauseType:
		// 场景1：叠加bool子条件（核心逻辑，增加安全校验）
		boolClause := string(c)
		// 校验 bool 子句合法性
		validBoolClauses := map[string]bool{
			string(BoolMust):    true,
			string(BoolShould):  true,
			string(BoolMustNot): true,
			string(BoolFilter):  true,
		}
		if !validBoolClauses[boolClause] {
			db.Err = fmt.Errorf("非法的bool子句类型：%s，仅支持must/should/must_not/filter", boolClause)
			return db
		}

		// 确保bool节点存在，且为map类型
		boolNode, ok := db.WhereQuery["bool"]
		if !ok {
			// 初始化bool节点
			boolNode = make(map[string]interface{})
			db.WhereQuery["bool"] = boolNode
		}
		boolQuery, ok := boolNode.(map[string]interface{})
		if !ok {
			db.Err = fmt.Errorf("bool查询节点类型错误，预期map[string]interface{}，实际%T", boolNode)
			return db
		}

		// 确保当前bool子句的切片存在，且为[]interface{}类型
		clauseVal, ok := boolQuery[boolClause]
		if !ok || clauseVal == nil {
			// 初始化子句切片
			clauseVal = make([]interface{}, 0)
			boolQuery[boolClause] = clauseVal
		}
		clauseSlice, ok := clauseVal.([]interface{})
		if !ok {
			db.Err = fmt.Errorf("bool子句[%s]类型错误，预期[]interface{}，实际%T", boolClause, clauseVal)
			return db
		}

		// 追加子条件到切片（安全叠加）
		newSlice := append(clauseSlice, query)
		boolQuery[boolClause] = newSlice
		// 若当前子句是should，且未设置minimum_should_match，则强制设置为1
		if boolClause == string(BoolShould) {
			if _, exists := boolQuery["minimum_should_match"]; !exists {
				boolQuery["minimum_should_match"] = 1
			}
		}

	case string:
		// 场景2：设置单一查询类型（如term/range/match_all等）
		if c == "" {
			db.Err = errors.New("查询类型不能为空")
			return db
		}
		// 若已存在bool查询，禁止覆盖（避免冲突）
		if _, ok := db.WhereQuery["bool"]; ok {
			db.Err = fmt.Errorf("已存在bool查询，无法同时设置单一查询类型[%s]", c)
			return db
		}
		// 校验单一查询类型合法性（可选，根据业务需要扩展）
		validQueryTypes := map[string]bool{
			"term":         true,
			"range":        true,
			"match":        true,
			"match_all":    true,
			"wildcard":     true,
			"term_set":     true,
			"match_phrase": true, // 短语匹配（比如"集水槽"精确短语匹配）
			"multi_match":  true, // 多字段匹配
			"exists":       true, // 字段存在匹配
			"prefix":       true, // 前缀匹配
			"regexp":       true, // 正则匹配
		}
		if validQueryTypes[c] || c == "" { // 空值已提前校验，此处兼容自定义查询类型
			db.WhereQuery[c] = query
		} else {
			db.Err = fmt.Errorf("暂不支持的查询类型：%s，支持类型：term/range/match/match_all/wildcard/term_set/match_phrase/multi_match/exists/prefix/regexp", c)
			return db
		}

	default:
		// 非法参数类型
		db.Err = errors.New("clause参数类型错误，仅支持BoolClauseType（must/should等）或string（term/range等）")
		return db
	}

	return db
}

// SetSort 设置排序（如 "id:asc", "create_time:desc"）
func (db *ESDb) SetSort(sort ...string) *ESDb {
	if db.Err != nil {
		return db
	}
	for _, s := range sort {
		parts := strings.Split(s, ":")
		if len(parts) != 2 || (parts[1] != "asc" && parts[1] != "desc") {
			db.Err = fmt.Errorf("排序格式错误[%s]，正确格式：字段名:asc/desc", s)
			return db
		}
		if !validIdentifierRegex.MatchString(parts[0]) {
			db.Err = fmt.Errorf("排序字段[%s]非法", parts[0])
			return db
		}
		db.Sort = append(db.Sort, s)
	}
	return db
}

// SetLimit 设置分页（对标MySQL的Limit，from=skip, size=num）
func (db *ESDb) SetLimit(from, size int64) *ESDb {
	if db.Err != nil {
		return db
	}
	if from < 0 {
		from = 0
	}
	if size < 0 {
		size = 0
	}
	if size > 10000 { // ES默认最大size限制
		size = 10000
	}
	db.From = from
	db.Size = size
	return db
}

// SetScriptFieldTruncate 为指定长文本字段配置脚本截取规则，返回指定长度的短字段（新字段field+"_short"）
// 核心特性：
//  1. 叠加配置：支持同时为多个字段（如content、xmmc）配置截取规则
//  2. 覆盖规则：重复配置同一字段时，后配置的规则会覆盖原有规则
//  3. 安全兼容：仅保留ES8 Painless兼容的极简脚本，避免语法/运行时报错
//  4. 性能优化：通过params._source读取原始数据，不触发fielddata（text字段禁用fielddata）
//
// 注意事项：
//   - 不支持在ES脚本中去除HTML标签（ES8 script_fields限制），建议入库时清理或业务层处理
//   - 最终返回的短字段别名格式为「原字段名_short」（如content → content_short）
//
// 参数：
//
//	field: 要截取的原字段名（仅支持字母/数字/下划线/中文，如"content"、"xmmc"）
//	length: 截取长度（必须>0，如100表示截取前100个字符）
//
// 返回：*ESDb 链式调用对象（错误会缓存至Err字段，后续调用直接返回）
func (db *ESDb) SetScriptFieldTruncate(field string, length int) *ESDb {
	// 提前终止：若已有错误，直接返回（链式调用失败快速失败）
	if db.Err != nil {
		return db
	}

	// 1. 字段名合法性校验（适配ES字段命名规范，支持中文字段）
	// 正则说明：^ 匹配开头，$ 匹配结尾，[]内为允许的字符，+ 表示至少一个字符
	fieldRegex := regexp.MustCompile(`^[a-zA-Z0-9_\\u4e00-\\u9fa5]+$`)
	if !fieldRegex.MatchString(field) {
		db.Err = fmt.Errorf("脚本字段[%s]名称非法，仅支持字母/数字/下划线/中文", field)
		return db
	}

	// 2. 截取长度校验（避免无效截取，如0或负数）
	if length <= 0 {
		db.Err = fmt.Errorf("字段[%s]截取长度必须>0，当前值：%d", field, length)
		return db
	}

	// 3. 初始化脚本字段容器（首次调用时创建，避免nil赋值）
	if db.ScriptFields == nil {
		db.ScriptFields = make(map[string]interface{})
	}

	// 4. 构建脚本字段核心配置
	// 4.1 生成短字段别名（原字段+_short，避免与原字段冲突）
	fieldAlias := fmt.Sprintf("%s_short", field)

	// 4.2 构建ES8 Painless兼容的极简脚本（核心逻辑）
	// 脚本说明：
	// - params._source: 读取文档原始_source数据（不受_source过滤影响，且不触发fielddata）
	// - containsKey: 安全判断字段是否存在，避免空指针
	// - toString(): 统一转为字符串，兼容数字/布尔等非字符串类型字段
	// - 三元表达式：长度超过指定值则截取前N位，否则返回原值
	scriptSource := strings.TrimSpace(fmt.Sprintf(`
		def val = params._source.containsKey('%s') ? params._source['%s'].toString() : '';
		return val.length() > %d ? val.substring(0, %d) : val;
	`, field, field, length, length))

	// 4.3 配置脚本字段（覆盖式添加，重复调用同一字段会替换原有规则）
	db.ScriptFields[fieldAlias] = map[string]interface{}{
		"script": map[string]interface{}{
			"lang":   "painless",   // 指定脚本语言为Painless（ES默认推荐）
			"source": scriptSource, // 脚本内容（极简逻辑，避免ES语法报错）
			"params": map[string]interface{}{
				// 传递_source参数：让脚本能读取原始文档数据（即使_source过滤了该字段）
				"_source": map[string]interface{}{},
			},
		},
	}

	// 链式调用返回自身
	return db
}

// SetHighlight 设置ES查询的高亮配置（支持单字段/多字段叠加）
// 功能说明：
//  1. 支持为单个字段配置高亮规则，多次调用可叠加多个字段的高亮配置
//  2. 若字段已配置过高亮，新配置会覆盖原有配置
//  3. 遵循ES高亮原生逻辑：FragmentSize=-1时返回完整字段，不截断
//
// 参数：
//
//	field: 需高亮的字段名（如 "content"、"title"），需符合ES字段命名规范
//	opt:   高亮配置选项（推荐通过 DefaultHighlightOptions 创建默认配置，或直接实例化 HighlightOption 自定义）
//
// 返回：*ESDb 链式调用对象
func (db *ESDb) SetHighlight(field string, opt HighlightOption) *ESDb {
	if db.Err != nil {
		return db
	}
	if !validIdentifierRegex.MatchString(field) {
		db.Err = fmt.Errorf("高亮字段[%s]非法", field)
		return db
	}
	fieldConfig := map[string]interface{}{
		"pre_tags":  []string{opt.PreTag},
		"post_tags": []string{opt.PostTag},
	}
	if opt.FragmentSize != 0 {
		fieldConfig["fragment_size"] = opt.FragmentSize
	}
	if opt.NumberOfFragments != 0 {
		fieldConfig["number_of_fragments"] = opt.NumberOfFragments
	}
	if len(db.Highlight) == 0 {
		db.Highlight = map[string]interface{}{
			"fields": map[string]interface{}{
				field: fieldConfig,
			},
		}
	} else {
		fields := db.Highlight["fields"].(map[string]interface{})
		fields[field] = fieldConfig
		db.Highlight = map[string]interface{}{
			"fields": fields,
		}
	}
	return db
}

// SetAggs 设置聚合配置
// 示例：SetAggs("age_stats", "stats", "age")
func (db *ESDb) SetAggs(aggName, aggType, field string) *ESDb {
	if db.Err != nil {
		return db
	}
	if !validIdentifierRegex.MatchString(aggName) || !validIdentifierRegex.MatchString(field) {
		db.Err = fmt.Errorf("聚合参数非法：name=%s, field=%s", aggName, field)
		return db
	}
	if db.Aggs == nil {
		db.Aggs = map[string]interface{}{}
	}
	db.Aggs[aggName] = map[string]interface{}{
		aggType: map[string]interface{}{
			"field": field,
		},
	}
	return db
}

// ===================== 核心操作方法 =====================

// FindAll 执行查询（对标MySQL的FindAll）
func (db *ESDb) FindAll(ctx context.Context) *ESDb {
	if db.Err != nil {
		return db
	}
	if db.Client == nil {
		db.Err = errors.New("ES客户端未初始化")
		return db
	}
	if len(db.Index) == 0 {
		db.Err = errors.New("未指定索引")
		return db
	}
	// 1. 构建查询DSL
	queryDSL := make(map[string]interface{})
	// 基础查询条件
	if len(db.WhereQuery) > 0 {
		queryDSL["query"] = db.WhereQuery
	} else {
		// 默认匹配所有
		queryDSL["query"] = map[string]interface{}{
			"match_all": map[string]interface{}{},
		}
	}
	if db.ScriptFields != nil && len(db.ScriptFields) > 0 {
		queryDSL["script_fields"] = db.ScriptFields
	}
	// 排序
	if len(db.Sort) > 0 {
		sortDSL := make([]map[string]interface{}, 0)
		for _, s := range db.Sort {
			parts := strings.Split(s, ":")
			sortDSL = append(sortDSL, map[string]interface{}{
				parts[0]: map[string]interface{}{
					"order": parts[1],
				},
			})
		}
		queryDSL["sort"] = sortDSL
	}
	// 返回字段
	if len(db.Source) > 0 || len(db.ExcludeSource) > 0 {
		sourceDSL := make(map[string]interface{})
		if len(db.Source) > 0 {
			sourceDSL["includes"] = db.Source
		}
		if len(db.ExcludeSource) > 0 {
			sourceDSL["exclude"] = db.ExcludeSource
		}
		queryDSL["_source"] = sourceDSL
	}
	// 分页
	queryDSL["from"] = db.From
	queryDSL["size"] = db.Size
	// 高亮
	if len(db.Highlight) > 0 {
		queryDSL["highlight"] = db.Highlight
	}
	// 聚合
	if len(db.Aggs) > 0 {
		queryDSL["aggs"] = db.Aggs
	}

	// 2. 序列化DSL
	queryBytes, err := json.Marshal(queryDSL)
	if err != nil {
		db.Err = fmt.Errorf("序列化查询DSL失败：%w", err)
		return db
	}
	// 3. 执行查询
	req := esapi.SearchRequest{
		Index:  db.Index,
		Body:   strings.NewReader(string(queryBytes)),
		Pretty: true,
	}
	res, err := req.Do(ctx, db.Client)
	if err != nil {
		db.Err = fmt.Errorf("执行查询失败：%w", err)
		return db
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logger.Error("ES查询关闭body失败 [索引：%s]，错误：%v", strings.Join(db.Index, ","), err)
		}
	}(res.Body)
	body, err := DeZip(db.GzipStatus, res)
	if err != nil {
		db.Err = fmt.Errorf("读取响应体失败：%v", err)
		return db
	}
	// 4. 解析响应结果
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		db.Err = fmt.Errorf("解析查询结果失败：%w", err)
		return db
	}
	// 5. 处理响应错误
	if res.IsError() {
		db.Err = fmt.Errorf("ES查询错误：%s", result["error"].(map[string]interface{})["reason"])
		return db
	}
	// 6. 提取文档数据
	hitsVal, ok := result["hits"]
	if !ok {
		db.Err = errors.New("ES响应无hits字段")
		return db
	}
	hitsMap, ok := hitsVal.(map[string]interface{})
	if !ok {
		db.Err = errors.New("ES响应hits字段类型错误")
		return db
	}
	hitsList, ok := hitsMap["hits"].([]interface{})
	if !ok {
		db.Err = errors.New("ES响应hits.hits字段类型错误")
		return db
	}
	// 新增：提取总匹配数（聚合场景常用）
	if totalVal, ok := hitsMap["total"]; ok {
		totalMap, ok := totalVal.(map[string]interface{})
		if ok {
			if totalCount, ok := totalMap["value"].(float64); ok {
				// 可新增 TotalCount 字段到 ESDb 结构体，存储总匹配数
				db.TotalCount = int64(totalCount)
			}
		}
	}
	data := make([]map[string]interface{}, 0, len(hitsList))
	for _, hit := range hitsList {
		hitMap, ok := hit.(map[string]interface{})
		if !ok {
			db.Err = fmt.Errorf("文档数据类型错误：%T", hit)
			return db
		}
		doc := make(map[string]interface{})
		// 文档元数据
		if id, ok := hitMap["_id"].(string); ok {
			doc["_id"] = id
		}
		if score, ok := hitMap["_score"].(float64); ok {
			doc["_score"] = score
		}
		// 文档内容
		if source, ok := hitMap["_source"].(map[string]interface{}); ok {
			for k, v := range source {
				doc[k] = v
			}
		}
		// 脚本字段
		if fieldsValue, ok := hitMap["fields"].(map[string]interface{}); ok {
			for k, v := range fieldsValue {
				doc[k] = v
			}
		}
		// 高亮内容
		if highlight, ok := hitMap["highlight"].(map[string]interface{}); ok {
			doc["_highlight"] = highlight
		}
		data = append(data, doc)
	}
	db.Data = data
	// 7. 聚合结果（如果有）
	aggsVal, hasAggs := result["aggregations"]
	if hasAggs {
		if aggs, ok := aggsVal.(map[string]interface{}); ok {
			db.AggsData = aggs
		} else {
			db.Err = errors.New("ES响应aggregations字段类型错误")
			return db
		}
	}
	// 无 aggregations 字段时不报错，仅置空 AggsData
	db.AggsData = nil
	return db
}

// FindCount 统计文档数量（对标MySQL的FindCount）
func (db *ESDb) FindCount(ctx context.Context) (int64, error) {
	defer db.clearData(false)
	if db.Err != nil {
		return 0, db.Err
	}
	// 重置分页，仅统计总数
	oldFrom, oldSize := db.From, db.Size
	db.From = 0
	db.Size = 0
	defer func() {
		db.From = oldFrom
		db.Size = oldSize
	}()

	// 构建计数DSL
	countDSL := map[string]interface{}{
		"query": db.WhereQuery,
	}
	if len(db.WhereQuery) == 0 {
		countDSL["query"] = map[string]interface{}{"match_all": map[string]interface{}{}}
	}
	countBytes, err := json.Marshal(countDSL)
	if err != nil {
		return 0, fmt.Errorf("序列化计数DSL失败：%w", err)
	}

	// 执行计数请求
	req := esapi.CountRequest{
		Index: db.Index,
		Body:  strings.NewReader(string(countBytes)),
	}
	res, err := req.Do(ctx, db.Client)
	if err != nil {
		return 0, fmt.Errorf("执行计数失败：%w", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logger.Error("ES执行findCount时关闭body时错误 Err：" + err.Error())
		}
	}(res.Body)
	body, err := DeZip(db.GzipStatus, res)
	if err != nil {
		return 0, fmt.Errorf("读取响应体失败：%v", err)
	}
	// 解析计数结果
	var countResp map[string]interface{}
	if err := json.Unmarshal(body, &countResp); err != nil {
		return 0, fmt.Errorf("解析计数结果失败：%w", err)
	}
	countVal, ok := countResp["count"]
	if !ok {
		return 0, fmt.Errorf("ES计数响应无count字段：%s", string(body))
	}
	countFloat, ok := countVal.(float64)
	if !ok {
		// 兼容 int 类型
		countInt, ok := countVal.(int64)
		if !ok {
			return 0, fmt.Errorf("count字段类型错误（预期float64/int64）：%T", countVal)
		}
		return countInt, nil
	}
	return int64(countFloat), nil
}

// Find 单文档查询（对标MySQL的Find）
func (db *ESDb) Find(ctx context.Context) (string, error) {
	defer db.clearData(false)
	if db.Err != nil {
		return "", db.Err
	}
	// 否则取第一条结果
	db.SetLimit(0, 1)
	db.FindAll(ctx)
	if len(db.Data) == 0 {
		return "", nil
	}
	return function.Json_encode(db.Data[0]), nil
}

// GetById 按ID查询单文档
func (db *ESDb) GetById(ctx context.Context, id string) (string, error) {
	defer db.clearData(false)
	if db.Err != nil {
		return "", db.Err
	}
	if len(db.Index) == 0 {
		return "", errors.New("未指定索引")
	}
	if id == "" {
		return "", errors.New("未指定文档ID")
	}
	req := esapi.GetRequest{
		Index:      db.Index[0],
		DocumentID: id,
	}
	if len(db.Source) > 0 {
		req.SourceIncludes = db.Source
	}
	if len(db.ExcludeSource) > 0 {
		req.SourceExcludes = db.ExcludeSource
	}
	res, err := req.Do(ctx, db.Client)
	if err != nil {
		return "", fmt.Errorf("查询文档失败：%w", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logger.Error("ES通过id获取指定文档时关闭body失败 Err：" + err.Error())
		}
	}(res.Body)
	body, err := DeZip(db.GzipStatus, res)
	if err != nil {
		return "", fmt.Errorf("读取响应体失败：%v", err)
	}
	if res.IsError() {
		if res.StatusCode == 404 {
			return "", fmt.Errorf("文档不存在：%s", id)
		}
		return "", fmt.Errorf("查询文档失败：%s，响应：%s", id, string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("解析文档失败：%w", err)
	}
	sourceVal, ok := result["_source"]
	if !ok || sourceVal == nil {
		return "", fmt.Errorf("文档[%s]无_source字段（可能被禁用）", id)
	}
	return function.Json_encode(sourceVal), nil
}

// Insert 新增单文档
func (db *ESDb) Insert(ctx context.Context, id string, data map[string]interface{}) (string, error) {
	defer db.clearData(false)
	if db.Err != nil {
		return "", db.Err
	}
	if len(db.Index) == 0 {
		return "", errors.New("未指定索引")
	}
	if len(data) == 0 {
		return "", errors.New("插入数据不能为空")
	}

	// 序列化文档
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("序列化文档失败：%w", err)
	}

	// 执行新增
	var req esapi.IndexRequest
	if id != "" {
		req = esapi.IndexRequest{
			Index:      db.Index[0],
			DocumentID: id,
			Body:       strings.NewReader(string(dataBytes)),
		}
	} else {
		req = esapi.IndexRequest{
			Index: db.Index[0],
			Body:  strings.NewReader(string(dataBytes)),
		}
	}

	res, err := req.Do(ctx, db.Client)
	if err != nil {
		return "", fmt.Errorf("新增文档失败：%w", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logger.Error("ES插入单个文档时关闭body失败 Err：" + err.Error())
		}
	}(res.Body)

	if res.IsError() {
		var errResp map[string]interface{}
		json.NewDecoder(res.Body).Decode(&errResp)
		return "", fmt.Errorf("ES新增错误：%s", errResp["error"].(map[string]interface{})["reason"])
	}

	// 返回文档ID
	var result map[string]interface{}
	err = json.NewDecoder(res.Body).Decode(&result)
	if err != nil {
		return "", err
	}
	return result["_id"].(string), nil
}

// InsertAll 批量插入/更新文档（链式调用）
// 返回：新增数、更新数、错误
// InsertAll 批量插入/更新文档（链式调用）
// 返回：新增数、更新数、错误
func (db *ESDb) InsertAll(ctx context.Context, dataList []map[string]interface{}) (insertCount int64, updateCount int64, err error) {
	defer db.clearData(false)
	// 链式错误传递
	if db.Err != nil {
		return 0, 0, db.Err
	}

	// 1. 基础校验
	if db.Client == nil {
		return 0, 0, errors.New("ES客户端未初始化")
	}
	if len(db.Index) == 0 {
		return 0, 0, errors.New("未指定索引名（请调用SetIndex）")
	}
	if len(dataList) == 0 {
		return 0, 0, nil
	}
	if len(dataList) >= 1000 {
		return 0, 0, errors.New("需要插入的文档数不得超过1000")
	}
	// 2. 构建Bulk请求体（优化：使用bytes.Buffer拼接）
	var bulkBuffer bytes.Buffer // 替换[]string为bytes.Buffer
	for idx, doc := range dataList {
		// 构建元数据
		meta := map[string]interface{}{
			"index": map[string]interface{}{
				"_index": db.Index[0], // 注意：原代码中db.Index是切片，此处取第一个（保持原逻辑）
			},
		}

		// 处理主键（SetPk设置的字段）
		if db.Pk != "" {
			pkVal, ok := doc[db.Pk]
			if !ok {
				return 0, 0, fmt.Errorf("第%d条文档缺失主键字段[%s]", idx+1, db.Pk)
			}
			// 转换主键为字符串（ES文档ID必须是字符串）
			pkStr, err := convertToString(pkVal)
			if err != nil {
				return 0, 0, fmt.Errorf("第%d条文档主键转换失败：%v", idx+1, err)
			}
			meta["index"].(map[string]interface{})["_id"] = pkStr
		}

		// 序列化元数据（直接写入缓冲区，避免字符串中转）
		if err := json.NewEncoder(&bulkBuffer).Encode(meta); err != nil {
			return 0, 0, fmt.Errorf("第%d条文档元数据序列化失败：%v", idx+1, err)
		}

		// 序列化文档数据（直接写入缓冲区）
		if err := json.NewEncoder(&bulkBuffer).Encode(doc); err != nil {
			return 0, 0, fmt.Errorf("第%d条文档数据序列化失败：%v", idx+1, err)
		}
	}

	// 3. 执行Bulk请求（缓冲区直接转为Reader，无额外拷贝）
	req := esapi.BulkRequest{
		Body:    bytes.NewReader(bulkBuffer.Bytes()),          // 直接使用缓冲区字节
		Timeout: time.Duration(db.BatchTimeout) * time.Second, // 超时配置
	}
	res, err := req.Do(ctx, db.Client)
	if err != nil {
		return 0, 0, fmt.Errorf("执行Bulk请求失败：%v", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logger.Error("ES批量插入时关闭body失败 Err：" + err.Error())
		}
	}(res.Body)

	// 后续逻辑（解压、解析响应）保持不变...
	// 4. 压缩响应判断（根据配置自动解压）
	body, err := DeZip(db.GzipStatus, res)
	if err != nil {
		return 0, 0, fmt.Errorf("读取响应体失败：%v", err)
	}

	// 5. 解析响应
	var bulkResp BulkResponse
	if err := json.Unmarshal(body, &bulkResp); err != nil {
		return 0, 0, fmt.Errorf("解析Bulk响应失败：%v，响应体：%s", err, string(body))
	}

	// 6. 处理结果（统计新增/更新数）
	var failCount int64
	for _, item := range bulkResp.Items {
		// 处理失败项
		if item.Index.Error.Type != "" {
			failCount++
			logger.Error("ES文档[%s]操作失败：%s-%s", item.Index.ID, item.Index.Error.Type, item.Index.Error.Reason)
			continue
		}
		// 统计新增/更新
		switch item.Index.Result {
		case "created":
			insertCount++
		case "updated":
			updateCount++
		}
	}

	// 7. 整体结果判断
	if bulkResp.Errors || failCount > 0 {
		err = fmt.Errorf("bulk操作部分失败，总数：%d，新增：%d，更新：%d，失败：%d",
			len(dataList), insertCount, updateCount, failCount)
	}
	return insertCount, updateCount, err
}

// UpdateById 按文档ID更新单文档
func (db *ESDb) UpdateById(ctx context.Context, id string, data map[string]interface{}) (bool, error) {
	defer db.clearData(false)
	if db.Err != nil {
		return false, db.Err
	}
	if len(db.Index) == 0 {
		return false, errors.New("未指定索引")
	}
	if id == "" {
		return false, errors.New("未指定文档ID")
	}
	if len(data) == 0 {
		return false, errors.New("更新数据不能为空")
	}

	// 构建更新DSL（partial update）
	updateDSL := map[string]interface{}{
		"doc": data,
	}
	updateBytes, err := json.Marshal(updateDSL)
	if err != nil {
		return false, fmt.Errorf("序列化更新DSL失败：%w", err)
	}

	// 执行更新
	req := esapi.UpdateRequest{
		Index:      db.Index[0],
		DocumentID: id,
		Body:       strings.NewReader(string(updateBytes)),
	}
	res, err := req.Do(ctx, db.Client)
	if err != nil {
		return false, fmt.Errorf("更新文档失败：%w", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logger.Error("ES修改指定id的文档时关闭body失败 Err：" + err.Error())
		}
	}(res.Body)
	body, err := DeZip(db.GzipStatus, res)
	if err != nil {
		return false, fmt.Errorf("读取响应体失败：%v", err)
	}
	if res.IsError() {
		var errResp map[string]interface{}
		err := json.Unmarshal(body, &errResp)
		if err != nil {
			return false, fmt.Errorf("ES更新错误：%s", err)
		}
		return false, fmt.Errorf("ES更新错误：%s", errResp["error"].(map[string]interface{})["reason"])
	}
	return true, nil
}

// UpdateByFull 批量覆盖文档（链式调用）
// 逻辑：基于文档ID替换整个文档（缺失字段会被删除），等价于 Bulk Index 操作
// 参数：
//
//	ctx - 上下文（客户端超时）
//	dataList - 待覆盖的文档列表（必须包含SetPk指定的主键字段）
//
// 返回：成功覆盖数、失败ID及原因、错误
func (db *ESDb) UpdateByFull(ctx context.Context, dataList []map[string]interface{}) (successCount int64, failMap map[string]string, err error) {
	defer db.clearData(false)
	// 链式错误传递
	if db.Err != nil {
		return 0, nil, db.Err
	}

	// 1. 基础校验
	if db.Client == nil {
		return 0, nil, errors.New("ES客户端未初始化")
	}
	if len(db.Index) == 0 {
		return 0, nil, errors.New("未指定索引名（请调用SetIndex）")
	}
	if db.Pk == "" {
		return 0, nil, errors.New("未指定主键字段（请调用SetPk）")
	}
	if len(dataList) == 0 {
		return 0, map[string]string{}, nil
	}
	if len(dataList) >= 1000 {
		return 0, nil, errors.New("需要更新的文档数不得超过1000")
	}
	// 2. 确定批量超时
	batchTimeout := 0
	if db.BatchTimeout > 0 {
		batchTimeout = db.BatchTimeout
	} else {
		batchTimeout = 30
	}

	// 3. 构建Bulk Index请求体（优化：使用bytes.Buffer）
	var bulkBuffer bytes.Buffer
	for idx, doc := range dataList {
		// 提取主键ID
		pkVal, ok := doc[db.Pk]
		if !ok {
			return 0, nil, fmt.Errorf("第%d条文档缺失主键字段[%s]", idx+1, db.Pk)
		}
		pkStr, err := convertToString(pkVal)
		if err != nil {
			return 0, nil, fmt.Errorf("第%d条文档主键转换失败：%v", idx+1, err)
		}

		// 构建Index元数据（全量覆盖）
		meta := map[string]interface{}{
			"index": map[string]interface{}{
				"_index": db.Index[0],
				"_id":    pkStr,
			},
		}

		// 序列化元数据（直接写入缓冲区）
		if err := json.NewEncoder(&bulkBuffer).Encode(meta); err != nil {
			return 0, nil, fmt.Errorf("第%d条文档元数据序列化失败：%v", idx+1, err)
		}

		// 序列化文档数据（直接写入缓冲区）
		if err := json.NewEncoder(&bulkBuffer).Encode(doc); err != nil {
			return 0, nil, fmt.Errorf("第%d条文档数据序列化失败：%v", idx+1, err)
		}
	}

	// 4. 执行Bulk请求
	req := esapi.BulkRequest{
		Body:    bytes.NewReader(bulkBuffer.Bytes()), // 直接使用缓冲区字节
		Timeout: time.Duration(batchTimeout) * time.Second,
	}
	res, err := req.Do(ctx, db.Client)
	if err != nil {
		if ctx.Err() != nil {
			return 0, nil, fmt.Errorf("客户端ctx超时：%v，批量超时配置：%d", ctx.Err(), batchTimeout)
		}
		return 0, nil, fmt.Errorf("执行全量覆盖失败：%v，批量超时配置：%d", err, batchTimeout)
	}
	defer res.Body.Close()

	// 后续逻辑（解压、解析响应）保持不变...
	// 5. 压缩响应处理
	body, err := DeZip(db.GzipStatus, res)
	if err != nil {
		return 0, nil, fmt.Errorf("读取响应体失败：%v", err)
	}

	// 6. 解析响应
	var bulkResp BulkResponse
	if err := json.Unmarshal(body, &bulkResp); err != nil {
		return 0, nil, fmt.Errorf("解析全量覆盖响应失败：%v，响应体：%s", err, string(body))
	}

	// 7. 统计结果
	successCount = 0
	failMap = make(map[string]string, len(dataList))
	for _, item := range bulkResp.Items {
		docID := item.Index.ID
		// 处理失败项
		if item.Index.Error.Type != "" {
			failMap[docID] = fmt.Sprintf("%s：%s", item.Index.Error.Type, item.Index.Error.Reason)
			logger.Error("ES文档[%s]全量覆盖失败：%v", docID, failMap[docID])
			continue
		}
		// 成功覆盖（result为"updated"）
		if item.Index.Result == "updated" {
			successCount++
		} else if item.Index.Result == "created" {
			// 文档不存在时会新增，视为“覆盖失败”（按需调整）
			failMap[docID] = "文档不存在，已新增（非预期覆盖）"
			logger.Error("ES文档[%s]全量覆盖失败：文档不存在，已新增", docID)
		}
	}

	// 8. 整体错误判断
	if bulkResp.Errors || len(failMap) > 0 {
		err = fmt.Errorf("全量覆盖部分失败：总数[%d]，成功[%d]，失败[%d]，批量超时配置：%d",
			len(dataList), successCount, len(failMap), batchTimeout)
	}
	return successCount, failMap, err
}

// UpdateByPartial 批量增量更新文档（链式调用）
// 逻辑：仅修改指定字段，保留其他字段，基于Bulk Update操作
// 参数：
//
//	ctx - 上下文（客户端超时）
//	dataList - 待更新的字段列表（格式：[{"id":1, "doc":{"status":"inactive", "age":26}}, ...]）
//
// 返回：成功更新数、失败ID及原因、错误
func (db *ESDb) UpdateByPartial(ctx context.Context, dataList []map[string]interface{}) (successCount int64, failMap map[string]string, err error) {
	defer db.clearData(false)
	// 链式错误传递
	if db.Err != nil {
		return 0, nil, db.Err
	}

	// 1. 基础校验
	if db.Client == nil {
		return 0, nil, errors.New("ES客户端未初始化")
	}
	if len(db.Index) == 0 {
		return 0, nil, errors.New("未指定索引名（请调用SetIndex）")
	}
	if db.Pk == "" {
		return 0, nil, errors.New("未指定主键字段（请调用SetPk）")
	}
	if len(dataList) == 0 {
		return 0, map[string]string{}, nil
	}
	if len(dataList) >= 1000 {
		return 0, nil, errors.New("需要更新的文档数不得超过1000")
	}
	// 2. 批量超时配置
	batchTimeout := 30
	if db.BatchTimeout > 0 {
		batchTimeout = db.BatchTimeout
	}

	// 3. 构建Bulk Update请求体（核心优化：bytes.Buffer替代[]string）
	// 预分配缓冲区容量（可选，按每条文档约300字节估算）
	estimatedSize := len(dataList) * 300
	var bulkBuffer bytes.Buffer
	bulkBuffer.Grow(estimatedSize) // 预扩容，减少内存分配

	for idx, doc := range dataList {
		// 提取主键ID
		pkVal, ok := doc[db.Pk]
		if !ok {
			return 0, nil, fmt.Errorf("第%d条文档缺失主键字段[%s]", idx+1, db.Pk)
		}
		pkStr, err := convertToString(pkVal)
		if err != nil {
			return 0, nil, fmt.Errorf("第%d条文档主键转换失败：%v", idx+1, err)
		}

		// 构建Update元数据
		meta := map[string]interface{}{
			"update": map[string]interface{}{
				"_index": db.Index[0],
				"_id":    pkStr,
			},
		}

		// 构建部分更新内容（移除主键字段，仅保留待更新字段）
		updateData := make(map[string]interface{})
		for k, v := range doc {
			if k != db.Pk { // 排除主键字段
				updateData[k] = v
			}
		}
		if len(updateData) == 0 {
			return 0, nil, fmt.Errorf("第%d条文档无待更新字段（主键：%s）", idx+1, pkStr)
		}
		updateBody := map[string]interface{}{
			"doc": updateData, // 部分更新核心语法
		}

		// 序列化元数据（直接写入缓冲区，无字符串中转）
		if err := json.NewEncoder(&bulkBuffer).Encode(meta); err != nil {
			return 0, nil, fmt.Errorf("第%d条文档元数据序列化失败：%v", idx+1, err)
		}

		// 序列化更新内容（直接写入缓冲区）
		if err := json.NewEncoder(&bulkBuffer).Encode(updateBody); err != nil {
			return 0, nil, fmt.Errorf("第%d条文档更新内容序列化失败：%v", idx+1, err)
		}
	}

	// 无有效更新内容时直接返回
	if bulkBuffer.Len() == 0 {
		return 0, map[string]string{}, nil
	}

	// 4. 执行Bulk请求
	req := esapi.BulkRequest{
		Body:    bytes.NewReader(bulkBuffer.Bytes()), // 零拷贝传递请求体
		Timeout: time.Duration(batchTimeout) * time.Second,
	}
	res, err := req.Do(ctx, db.Client)
	if err != nil {
		if ctx.Err() != nil {
			return 0, nil, fmt.Errorf("客户端ctx超时：%v，批量超时配置：%d", ctx.Err(), batchTimeout)
		}
		return 0, nil, fmt.Errorf("执行批量部分更新失败：%v，批量超时配置：%d", err, batchTimeout)
	}
	defer res.Body.Close()

	// 5. 读取并解析响应（逻辑与UpdateByFull一致）
	body, err := DeZip(db.GzipStatus, res)
	if err != nil {
		return 0, nil, fmt.Errorf("读取响应体失败：%v", err)
	}

	// 解析Bulk响应
	var bulkResp BulkUpdateResponse
	if err := json.Unmarshal(body, &bulkResp); err != nil {
		return 0, nil, fmt.Errorf("解析批量部分更新响应失败：%v，响应体：%s", err, string(body))
	}

	// 6. 统计结果
	successCount = 0
	failMap = make(map[string]string, len(dataList))
	for _, item := range bulkResp.Items {
		docID := item.Update.ID
		// 处理失败项
		if item.Update.Error.Type != "" {
			failMap[docID] = fmt.Sprintf("%s：%s", item.Update.Error.Type, item.Update.Error.Reason)
			logger.Error("ES文档[%s]部分更新失败：%v", docID, failMap[docID])
			continue
		}
		// 成功更新（result为"updated"）
		if item.Update.Result == "updated" {
			successCount++
		} else if item.Update.Result == "noop" {
			// 无更新（字段值未变化），视为成功或失败按需调整
			failMap[docID] = "无字段更新（值未变化）"
			logger.Error("ES文档[%s]部分更新无操作：字段值未变化", docID)
		} else if item.Update.Result == "not_found" {
			failMap[docID] = "文档不存在"
			logger.Error("ES文档[%s]部分更新失败：文档不存在", docID)
		}
	}

	// 7. 整体错误判断
	if bulkResp.Errors || len(failMap) > 0 {
		err = fmt.Errorf("批量部分更新部分失败：总数[%d]，成功[%d]，失败[%d]，批量超时配置：%d",
			len(dataList), successCount, len(failMap), batchTimeout)
	}

	return successCount, failMap, err
}

// Update 按条件批量更新（链式调用，基于SetWhere设置的条件）
// 逻辑：无需文档ID，按条件更新指定字段，适合批量修改符合条件的所有文档
// 参数：
//
//	ctx - 上下文（客户端超时）
//	updateDoc - 待更新的字段（如：map[string]interface{}{"status": "inactive"}）
//
// 返回：成功更新数、失败数、错误
func (db *ESDb) Update(ctx context.Context, updateDoc map[string]interface{}) (updatedCount int64, failCount int64, err error) {
	defer db.clearData(false)
	// 链式错误传递
	if db.Err != nil {
		return 0, 0, db.Err
	}

	// 1. 基础校验
	if db.Client == nil {
		return 0, 0, errors.New("ES客户端未初始化")
	}
	if len(db.Index) == 0 {
		return 0, 0, errors.New("未指定索引名（请调用SetIndex）")
	}
	if len(db.WhereQuery) == 0 {
		return 0, 0, errors.New("未设置更新条件（请调用SetWhere）")
	}
	if len(updateDoc) == 0 {
		return 0, 0, errors.New("未指定待更新的字段")
	}

	// 2. 确定批量超时
	batchTimeout := 0
	if db.BatchTimeout > 0 {
		batchTimeout = db.BatchTimeout
	} else {
		batchTimeout = 30
	}

	// 3. 构建Update By Query请求体
	updateBody := map[string]interface{}{
		"query": db.WhereQuery, // 复用SetWhere的条件
		"script": map[string]interface{}{
			"source": "ctx._source.putAll(params.doc)", // 批量设置字段
			"params": map[string]interface{}{
				"doc": updateDoc,
			},
			"lang": "painless",
		},
		// 可选：设置批次大小，避免单次更新过多文档
		"size": 1000,
	}
	bodyJSON, err := json.Marshal(updateBody)
	if err != nil {
		return 0, 0, fmt.Errorf("构建条件更新请求体失败：%v", err)
	}
	Refresh := true
	IgnoreUnavailable := true
	// 4. 执行Update By Query请求
	req := esapi.UpdateByQueryRequest{
		Index:   db.Index,
		Body:    strings.NewReader(string(bodyJSON)),
		Timeout: time.Duration(batchTimeout) * time.Second,
		// 可选：更新后立即刷新
		Refresh: &Refresh,
		// 可选：忽略不存在的索引
		IgnoreUnavailable: &IgnoreUnavailable,
	}
	res, err := req.Do(ctx, db.Client)
	if err != nil {
		if ctx.Err() != nil {
			return 0, 0, fmt.Errorf("客户端ctx超时：%v，批量超时配置：%d", ctx.Err(), batchTimeout)
		}
		return 0, 0, fmt.Errorf("执行条件更新失败：%v，批量超时配置：%d", err, batchTimeout)
	}
	defer res.Body.Close()

	// 5. 压缩响应处理
	body, err := DeZip(db.GzipStatus, res)
	if err != nil {
		return 0, 0, fmt.Errorf("读取响应体失败：%v", err)
	}

	// 6. 解析响应
	var resp UpdateByQueryResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return 0, 0, fmt.Errorf("解析条件更新响应失败：%v，响应体：%s", err, string(body))
	}

	// 7. 统计结果
	updatedCount = resp.Updated + resp.Noops // 无更新的也视为成功
	failCount = int64(len(resp.Failures))

	// 8. 处理失败详情
	if failCount > 0 {
		failReason := make([]string, 0, failCount)
		for _, f := range resp.Failures {
			failReason = append(failReason, fmt.Sprintf("索引[%s]：%s-%s", f.Index, f.Type, f.Reason))
			logger.Error("ES条件更新失败：%v", failReason[len(failReason)-1])
		}
		err = fmt.Errorf("条件更新部分失败：成功更新[%d]（含无变化[%d]），失败[%d]，失败原因：%v",
			updatedCount, resp.Noops, failCount, strings.Join(failReason, "; "))
	}

	return updatedCount, failCount, err
}

// DeleteById 删除单文档
func (db *ESDb) DeleteById(ctx context.Context, id string) (bool, error) {
	defer db.clearData(false)
	if db.Err != nil {
		return false, db.Err
	}
	if len(db.Index) == 0 {
		return false, errors.New("未指定索引")
	}
	if id == "" {
		return false, errors.New("未指定文档ID")
	}

	req := esapi.DeleteRequest{
		Index:      db.Index[0],
		DocumentID: id,
	}
	res, err := req.Do(ctx, db.Client)
	if err != nil {
		return false, fmt.Errorf("删除文档失败：%w", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logger.Error("ES删除指定id的文档时关闭body失败 Err：" + err.Error())
		}
	}(res.Body)
	body, err := DeZip(db.GzipStatus, res)
	if err != nil {
		return false, fmt.Errorf("读取响应体失败：%v", err)
	}
	if res.IsError() {
		var errResp map[string]interface{}
		err = json.Unmarshal(body, &errResp)
		if err != nil {
			return false, fmt.Errorf("ES删除错误：%s", err)
		}
		return false, fmt.Errorf("ES删除错误：%s", errResp["error"].(map[string]interface{})["reason"])
	}
	return true, nil
}

// DeleteByIDs 按文档ID批量删除（链式调用）
// 参数：
//
//	ctx - 上下文（含客户端超时）
//	ids - 待删除的文档ID列表
//
// 返回：成功删除数、失败ID及原因、错误
func (db *ESDb) DeleteByIDs(ctx context.Context, ids []string) (successCount int64, failMap map[string]string, err error) {
	defer db.clearData(false)
	// 链式错误传递
	if db.Err != nil {
		return 0, nil, db.Err
	}

	// 1. 基础校验
	if db.Client == nil {
		return 0, nil, errors.New("ES客户端未初始化")
	}
	if len(db.Index) == 0 {
		return 0, nil, errors.New("未指定索引名（请调用SetIndex）")
	}
	if len(ids) == 0 {
		return 0, map[string]string{}, nil
	}
	if len(ids) >= 1000 {
		return 0, nil, errors.New("需要删除的文档数不得超过1000")
	}
	// 2. 确定Bulk服务端超时（修复原逻辑错误）
	var batchTimeout int
	if db.BatchTimeout > 0 {
		batchTimeout = db.BatchTimeout
	} else {
		batchTimeout = 30
	}

	// 3. 构建Bulk Delete请求体（优化：使用bytes.Buffer）
	var bulkBuffer bytes.Buffer
	for _, docID := range ids {
		if docID == "" {
			continue
		}
		// 构建delete操作元数据
		meta := map[string]interface{}{
			"delete": map[string]interface{}{
				"_index": db.Index[0],
				"_id":    docID,
			},
		}
		// 序列化元数据（直接写入缓冲区）
		if err := json.NewEncoder(&bulkBuffer).Encode(meta); err != nil {
			return 0, nil, fmt.Errorf("文档ID[%s]元数据序列化失败：%v", docID, err)
		}
	}

	// 无有效ID时直接返回
	if bulkBuffer.Len() == 0 {
		return 0, map[string]string{}, nil
	}

	// 4. 执行Bulk请求
	req := esapi.BulkRequest{
		Body:    bytes.NewReader(bulkBuffer.Bytes()), // 直接使用缓冲区字节
		Timeout: time.Duration(batchTimeout) * time.Second,
	}
	res, err := req.Do(ctx, db.Client)
	if err != nil {
		if ctx.Err() != nil {
			return 0, nil, fmt.Errorf("客户端ctx超时：%v，Bulk超时配置：%d", ctx.Err(), batchTimeout)
		}
		return 0, nil, fmt.Errorf("执行Bulk删除失败：%v，Bulk超时配置：%d", err, batchTimeout)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logger.Error("ES按ID批量删除文档时关闭body失败 Err：" + err.Error())
		}
	}(res.Body)

	// 后续逻辑（解压、解析响应）保持不变...
	// 5. 压缩响应处理
	body, err := DeZip(db.GzipStatus, res)
	if err != nil {
		return 0, nil, fmt.Errorf("读取响应体失败：%v", err)
	}

	var bulkResp BulkDeleteResponse
	if err := json.Unmarshal(body, &bulkResp); err != nil {
		return 0, nil, fmt.Errorf("解析Bulk删除响应失败：%v，响应体：%s", err, string(body))
	}

	// 7. 统计结果
	successCount = 0
	failMap = make(map[string]string, len(ids))
	for _, item := range bulkResp.Items {
		docID := item.Delete.ID
		// 处理失败项
		if item.Delete.Error.Type != "" {
			failMap[docID] = fmt.Sprintf("%s：%s", item.Delete.Error.Type, item.Delete.Error.Reason)
			logger.Error("ES文档[%s]删除失败：%v", docID, failMap[docID])
			continue
		}
		// 成功删除（result为"deleted"，不存在则为"not_found"）
		if item.Delete.Result == "deleted" {
			successCount++
		} else if item.Delete.Result == "not_found" {
			failMap[docID] = "文档不存在"
			logger.Error("ES文档[%s]删除失败：文档不存在", docID)
		}
	}

	// 8. 整体错误判断
	if bulkResp.Errors || len(failMap) > 0 {
		err = fmt.Errorf("批量删除部分失败：总数[%d]，成功[%d]，失败[%d]，Bulk超时配置：%d",
			len(ids), successCount, len(failMap), batchTimeout)
	}

	return successCount, failMap, err
}

// Delete 按查询条件批量删除（链式调用，基于SetWhere设置的条件,Limit最大1000）
// 参数：
//
//	ctx - 上下文（含客户端超时）
//
// 返回：成功删除数、失败数、错误
func (db *ESDb) Delete(ctx context.Context) (deletedCount int64, failCount int64, err error) {
	defer db.clearData(false)
	// 链式错误传递
	if db.Err != nil {
		return 0, 0, db.Err
	}

	// 1. 基础校验
	if db.Client == nil {
		return 0, 0, errors.New("ES客户端未初始化")
	}
	if len(db.Index) == 0 {
		return 0, 0, errors.New("未指定索引名（请调用SetIndex）")
	}
	if len(db.WhereQuery) == 0 {
		return 0, 0, errors.New("未设置删除条件（请调用SetWhere）")
	}

	// 确定查询超时
	batchTimeout := 0
	if db.BatchTimeout > 0 {
		batchTimeout = db.BatchTimeout
	} else {
		batchTimeout = 30
	}
	limit := int64(0)
	if db.Size > 0 {
		limit = db.Size
	} else {
		limit = 1000
	}
	// 构建Delete By Query请求体
	deleteBody := map[string]interface{}{
		"query": db.WhereQuery, // 复用SetWhere设置的查询条件
		// 可选：设置批次大小，避免单次删除过多文档
		"max_docs": limit,
	}
	bodyJSON, err := json.Marshal(deleteBody)
	if err != nil {
		return 0, 0, fmt.Errorf("构建删除请求体失败：%v", err)
	}
	Refresh := true
	IgnoreUnavailable := true
	// 4. 执行Delete By Query请求
	req := esapi.DeleteByQueryRequest{
		Index:   db.Index,
		Body:    strings.NewReader(string(bodyJSON)),
		Timeout: time.Duration(batchTimeout) * time.Second,
		// 可选：设置刷新策略（删除后立即刷新）
		Refresh: &Refresh,
		// 可选：忽略不存在的索引
		IgnoreUnavailable: &IgnoreUnavailable,
	}
	res, err := req.Do(ctx, db.Client)
	if err != nil {
		if ctx.Err() != nil {
			return 0, 0, fmt.Errorf("客户端ctx超时：%v，查询超时配置：%d", ctx.Err(), batchTimeout)
		}
		return 0, 0, fmt.Errorf("执行条件删除失败：%v，查询超时配置：%d", err, batchTimeout)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logger.Error("ES按条件批量删除时关闭body失败 Err：" + err.Error())
		}
	}(res.Body)

	// 5. 压缩响应处理
	body, err := DeZip(db.GzipStatus, res)
	if err != nil {
		return 0, 0, fmt.Errorf("读取响应体失败：%v", err)
	}

	// 解析响应

	var resp DeleteByQueryResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return 0, 0, fmt.Errorf("解析条件删除响应失败：%v，响应体：%s", err, string(body))
	}

	// 7. 统计结果
	deletedCount = resp.Deleted
	failCount = int64(len(resp.Failures))

	// 8. 处理失败详情
	if failCount > 0 {
		failReason := make([]string, 0, failCount)
		for _, f := range resp.Failures {
			failReason = append(failReason, fmt.Sprintf("索引[%s]：%s-%s", f.Index, f.Type, f.Reason))
			logger.Error("ES条件删除失败：%v", failReason[len(failReason)-1])
		}
		err = fmt.Errorf("条件删除部分失败：成功删除[%d]，失败[%d]，失败原因：%v",
			deletedCount, failCount, strings.Join(failReason, "; "))
	}

	return deletedCount, failCount, err
}

// ToBegin 开启批量操作
func (db *ESDb) ToBegin() *ESDb {
	if db.Err != nil {
		return db
	}
	db.BulkActions = make([]string, 0)
	return db
}

// AddBulkInsert 批量新增（事务中）
func (db *ESDb) AddBulkInsert(data map[string]interface{}) *ESDb {
	if db.Err != nil {
		return db
	}
	if len(db.BulkActions) > 1000 {
		db.Err = errors.New("批量操作Bulk太大超过最大值1000")
		return db
	}
	meta := map[string]interface{}{
		"index": map[string]interface{}{
			"_index": db.Index[0],
		},
	}
	if id, ok := data["_id"].(string); ok && id != "" {
		meta["index"].(map[string]interface{})["_id"] = id
		delete(data, "_id")
	}
	metaBytes, err := json.Marshal(meta)
	if err != nil {
		db.Err = err
		return db
	}
	dataBytes, err := json.Marshal(data)
	if err != nil {
		db.Err = err
		return db
	}
	db.BulkActions = append(db.BulkActions, string(metaBytes), string(dataBytes))
	return db
}

// AddBulkUpdate 批量更新（事务中）
func (db *ESDb) AddBulkUpdate(data map[string]interface{}) *ESDb {
	if db.Err != nil {
		return db
	}
	if db.Id == "" {
		db.Err = errors.New("未指定文档ID")
		return db
	}
	if len(db.BulkActions) > 1000 {
		db.Err = errors.New("批量操作Bulk太大超过最大值1000")
		return db
	}
	meta := map[string]interface{}{
		"update": map[string]interface{}{
			"_index": db.Index[0],
			"_id":    db.Id,
		},
	}
	updateDSL := map[string]interface{}{"doc": data}
	metaBytes, err := json.Marshal(meta)
	if err != nil {
		db.Err = err
		return db
	}
	updateBytes, err := json.Marshal(updateDSL)
	if err != nil {
		db.Err = err
		return db
	}
	db.BulkActions = append(db.BulkActions, string(metaBytes), string(updateBytes))
	return db
}

// AddBulkDelete 批量删除（事务中）
func (db *ESDb) AddBulkDelete() *ESDb {
	if db.Err != nil {
		return db
	}
	if db.Id == "" {
		db.Err = errors.New("未指定文档ID")
		return db
	}
	if len(db.BulkActions) > 1000 {
		db.Err = errors.New("批量操作Bulk太大超过最大值1000")
		return db
	}
	meta := map[string]interface{}{
		"delete": map[string]interface{}{
			"_index": db.Index[0],
			"_id":    db.Id,
		},
	}
	metaBytes, err := json.Marshal(meta)
	if err != nil {
		db.Err = err
		return db
	}
	db.BulkActions = append(db.BulkActions, string(metaBytes))
	return db
}

// Commit 提交批量操作（对标MySQL的Commit）
func (db *ESDb) Commit(ctx context.Context) (int64, error) {
	defer db.clearData(true)
	if db.Err != nil {
		return 0, db.Err
	}
	if len(db.BulkActions) == 0 {
		return 0, errors.New("无批量操作待提交")
	}

	req := esapi.BulkRequest{
		Body: strings.NewReader(strings.Join(db.BulkActions, "\n") + "\n"),
	}
	res, err := req.Do(ctx, db.Client)
	if err != nil {
		return 0, fmt.Errorf("提交批量操作失败：%w", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logger.Error("ES执行批量操作提交时关闭body失败 Err：" + err.Error())
		}
	}(res.Body)
	body, err := DeZip(db.GzipStatus, res)
	if err != nil {
		return 0, fmt.Errorf("读取响应体失败：%v", err)
	}
	var result map[string]interface{}
	err = json.Unmarshal(body, &result)
	if err != nil {
		return 0, fmt.Errorf("批量操作提交失败：%v", string(body))
	}
	if result["errors"].(bool) {
		return 0, fmt.Errorf("批量操作部分失败：%v", result)
	}
	return int64(len(result["items"].([]interface{}))), nil
}

// Rollback 模拟的回滚批量操作，非原子性
func (db *ESDb) Rollback() {
	defer db.clearData(true)
	return
}

// CreateIndex 创建单个索引
// 参数：
//
//	ctx - 上下文（含客户端超时）
//	timeout - 可选，索引创建超时（默认30s）
//
// 返回：错误信息
func (db *ESDb) CreateIndex(ctx context.Context, mapping map[string]interface{}) error {
	defer db.clearData(false)
	// 链式错误传递
	if db.Err != nil {
		return db.Err
	}

	// 1. 基础校验
	if db.Client == nil {
		return errors.New("ES客户端未初始化")
	}
	if len(db.Index) == 0 {
		return errors.New("未指定索引名（请调用SetIndex）")
	}
	if mapping == nil || len(mapping) == 0 {
		return errors.New("索引mapping不能为空")
	}
	// 2. 超时配置
	batchTimeout := 0
	if db.BatchTimeout > 0 {
		batchTimeout = db.BatchTimeout
	} else {
		batchTimeout = 30
	}

	// 3. 序列化映射配置
	mappingBytes, err := json.Marshal(mapping)
	if err != nil {
		return fmt.Errorf("JSON序列化映射配置失败：%v", err)
	}

	// 4. 构建创建索引请求
	req := esapi.IndicesCreateRequest{
		Index:   db.Index[0],
		Body:    strings.NewReader(string(mappingBytes)),
		Timeout: time.Duration(batchTimeout) * time.Second,
	}

	// 5. 执行请求
	res, err := req.Do(ctx, db.Client)
	if err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("客户端ctx超时：%v，索引创建超时配置：%d", ctx.Err(), batchTimeout)
		}
		return fmt.Errorf("创建索引[%s]请求失败：%v", db.Index[0], err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logger.Error("ES创建索引时关闭body失败 Err：" + err.Error())
		}
	}(res.Body)

	// 6. 处理压缩响应
	body, err := DeZip(db.GzipStatus, res)
	if err != nil {
		return fmt.Errorf("读取响应体失败：%v", err)
	}

	// 7. 解析响应（错误处理）
	if res.IsError() {
		var e map[string]interface{}
		if err := json.Unmarshal(body, &e); err != nil {
			return fmt.Errorf("解析创建索引响应失败：%v", err)
		}
		// 提取错误信息
		errorReason := "未知错误"
		if errObj, ok := e["error"].(map[string]interface{}); ok {
			if reason, ok := errObj["reason"].(string); ok {
				errorReason = reason
			}
		}
		return fmt.Errorf("创建索引[%s]失败：%s", db.Index[0], errorReason)
	}
	return nil
}

// DeleteIndex 删除索引（支持单个/多个）
// 参数：
//
//	ctx - 上下文（含客户端超时）
//
// 返回：错误信息
func (db *ESDb) DeleteIndex(ctx context.Context) error {
	defer db.clearData(false)
	// 链式错误传递
	if db.Err != nil {
		return db.Err
	}

	// 1. 基础校验
	if db.Client == nil {
		return errors.New("ES客户端未初始化")
	}

	// 确定待删除的索引列表
	if len(db.Index) == 0 {
		return errors.New("未指定待删除的索引名（请调用SetIndex）")
	}

	// 2. 超时配置
	batchTimeout := 0
	if db.BatchTimeout > 0 {
		batchTimeout = db.BatchTimeout
	} else {
		batchTimeout = 30
	}

	// 3. 构建删除索引请求
	req := esapi.IndicesDeleteRequest{
		Index:   db.Index,
		Timeout: time.Duration(batchTimeout) * time.Second,
	}

	// 4. 执行请求
	res, err := req.Do(ctx, db.Client)
	if err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("客户端ctx超时：%v，索引删除超时配置：%d", ctx.Err(), batchTimeout)
		}
		return fmt.Errorf("删除索引[%s]请求失败：%v", strings.Join(db.Index, ","), err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logger.Error("ES删除指定索引时关闭body失败 Err：" + err.Error())
		}
	}(res.Body)

	// 5. 处理压缩响应
	body, err := DeZip(db.GzipStatus, res)
	if err != nil {
		return fmt.Errorf("读取响应体失败：%v", err)
	}
	// 6. 解析响应
	var responseMap map[string]interface{}
	if err := json.Unmarshal(body, &responseMap); err != nil {
		return fmt.Errorf("解析删除索引响应失败：%v，响应体：%s", err, string(body))
	}

	// 7. 处理响应结果
	if res.IsError() {
		errorReason := "未知错误"
		if errObj, ok := responseMap["error"].(map[string]interface{}); ok {
			if reason, ok := errObj["reason"].(string); ok {
				errorReason = reason
				// 忽略"索引不存在"的错误
				if strings.Contains(errorReason, "no such index") {
					logger.Error("索引[%s]不存在，无需删除", strings.Join(db.Index, ","))
					return nil
				}
			}
		}
		return fmt.Errorf("删除索引[%s]失败：%s", strings.Join(db.Index, ","), errorReason)
	}

	// 验证删除确认
	if acknowledged, ok := responseMap["acknowledged"].(bool); !ok || !acknowledged {
		return fmt.Errorf("删除索引[%s]未被确认，响应：%s", strings.Join(db.Index, ","), string(body))
	}
	return nil
}

// IndexExists 检查索引是否存在（链式调用）
func (db *ESDb) IndexExists(ctx context.Context) (bool, error) {
	defer db.clearData(false)
	if db.Err != nil {
		return false, db.Err
	}
	if db.Client == nil {
		return false, errors.New("ES客户端未初始化")
	}
	if len(db.Index) == 0 {
		return false, errors.New("未指定索引名（请调用SetIndex）")
	}

	req := esapi.IndicesExistsRequest{
		Index: db.Index,
	}
	res, err := req.Do(ctx, db.Client)
	if err != nil {
		return false, fmt.Errorf("检查索引[%s]是否存在失败：%v", db.Index, err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logger.Error("ES判断index是否存在时关闭body失败 Err：" + err.Error())
		}
	}(res.Body)

	switch res.StatusCode {
	case 200:
		return true, nil
	case 404:
		return false, nil
	default:
		return false, fmt.Errorf("检查索引[%s]状态异常，状态码：%d", db.Index, res.StatusCode)
	}
}

// ToString 返回查询结果JSON（对标MySQL的ToString）
func (db *ESDb) ToString() (string, error) {
	defer db.clearData(false)
	if db.Err != nil {
		return "", db.Err
	}
	if len(db.Data) == 0 {
		return "", nil
	}
	return function.Json_encode(db.Data), nil
}
func (db *ESDb) IkFenCi(ctx context.Context, analyzer string, analyzeText string) ([]string, error) {
	// 链式错误传递
	if db.Err != nil {
		return nil, db.Err
	}
	// 1. 基础校验
	if db.Client == nil {
		return nil, errors.New("ES客户端未初始化（db.Client为空）")
	}
	if analyzer == "" {
		return nil, errors.New("未指定分词器类型")
	}
	if analyzeText == "" {
		return nil, errors.New("未指定待分词文本")
	}

	// 2. 解析超时参数（服务端超时）
	//batchTimeout := 0
	//if db.BatchTimeout > 0 {
	//	batchTimeout = db.BatchTimeout
	//} else {
	//	batchTimeout = 10 // 默认10s
	//}
	analyzeReq := AnalyzeRequest{
		Analyzer: analyzer,
		Text:     []string{analyzeText}, // 支持多文本：[]string{"文本1", "文本2"}
	}
	// 序列化请求体
	reqBodyBytes, err := json.Marshal(analyzeReq)
	if err != nil {
		return nil, fmt.Errorf("序列化分词请求体失败：%v", err)
	}
	reqBody := strings.NewReader(string(reqBodyBytes))
	// 3. 构建官方Analyze请求（核心：使用esapi.IndicesAnalyzeRequest）
	req := esapi.IndicesAnalyzeRequest{
		Body: reqBody, // 核心：传递JSON请求体
		// 设置请求头（指定JSON格式）
		Header: http.Header{
			"Content-Type": []string{"application/json; charset=utf-8"},
		},
	}
	// 4. 执行官方API请求（直接复用db.Client）
	res, err := req.Do(ctx, db.Client)
	if err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("分词请求ctx超时：%v", ctx.Err())
		}
		return nil, fmt.Errorf("执行ES官方分词API失败：%v", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logger.Error("ES分词关闭body失败 Err：" + err.Error())
		}
	}(res.Body)
	// 5. 读取响应体（复用压缩处理逻辑）
	body, err := DeZip(db.GzipStatus, res)
	if err != nil {
		return nil, fmt.Errorf("读取分词响应体失败：%v", err)
	}
	// 6. 解析官方响应（使用强类型结构体）
	var analyzeResp AnalyzeResponse
	if err := json.Unmarshal(body, &analyzeResp); err != nil {
		return nil, fmt.Errorf("解析分词响应失败：%v，响应体：%s", err, string(body))
	}
	// 7. 处理ES返回的错误
	if analyzeResp.Error != nil || res.IsError() {
		errReason := "未知错误"
		if analyzeResp.Error != nil {
			if len(analyzeResp.Error.RootCause) > 0 {
				errReason = analyzeResp.Error.RootCause[0].Reason
			} else {
				errReason = analyzeResp.Error.Reason
			}
		} else {
			errReason = fmt.Sprintf("HTTP状态码异常：%d", res.StatusCode)
		}
		return nil, fmt.Errorf("ES分词API返回错误：%s，响应体：%s", errReason, string(body))
	}

	// 8. 处理分词结果（保留原有核心逻辑）
	wordList := make([]string, 0, len(analyzeResp.Tokens))
	if len(analyzeResp.Tokens) > 0 {
		if analyzer == "ik_smart" {
			// ik_smart 按偏移量拼接，防止重复/遗漏
			start := float64(0)
			index := 0
			for {
				if analyzeText == strings.Join(wordList, "") || index >= len(analyzeResp.Tokens) {
					break
				}
				for _, token := range analyzeResp.Tokens {
					if analyzeText == strings.Join(wordList, "") {
						break
					}
					if token.StartOffset >= start {
						start = token.EndOffset
						wordList = append(wordList, token.Token)
					}
				}
				index++ // 防止异常死循环
			}
		} else {
			// 其他分词器直接提取token
			for _, token := range analyzeResp.Tokens {
				wordList = append(wordList, token.Token)
			}
		}
	}
	return wordList, nil
}
func (db *ESDb) clearData(isClearTx bool) {
	db.Index = []string{}
	db.Id = ""
	db.WhereQuery = nil
	db.Aggs = nil
	db.Sort = []string{}
	db.ExcludeSource = []string{}
	db.Source = []string{}
	db.From = int64(0)
	db.Size = int64(0)
	db.Highlight = nil
	db.Pk = ""
	db.BatchTimeout = 0
	db.Data = nil
	db.AggsData = nil
	db.TotalCount = int64(0)
	db.Err = nil
	if isClearTx {
		db.BulkActions = nil
	}
}

// CloseES 关闭所有ES连接
func CloseES() error {
	var err error
	multiESPool.Range(func(key, value interface{}) bool {
		esObj, ok := value.(DbObj)
		if !ok {
			err = fmt.Errorf("无效的ES客户端对象（key: %v）", key)
			return false
		}
		if esObj.Transport != nil {
			esObj.Transport.CloseIdleConnections()
			esObj.Transport = nil
		}
		esObj.Client = nil
		fmt.Printf("ES 连接已关闭（dbKey: %v）\n", key)
		return true
	})
	fmt.Println("ES 连接已全部关闭")
	multiESPool = sync.Map{} // 清空连接池
	return err
}
