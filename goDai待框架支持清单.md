goDai 框架待支持清单（ldApiServer 迁移整改，2026-09-23）

背景：ldApiServer 已全面切换 goDai 链式方法，以下 7 项是链式方法
暂时覆盖不了的能力。每条按统一结构展开：goDai 现状（源码行为）→
具体例子 → 业务影响点 → 拆解评估结论 → 框架改法建议，均附源码
位置与可运行的复现代码，方便核实。改好哪条知会一声，我改对应代码。


一、ESDb 缺 collapse（字段去重折叠）

【goDai 现状】
FindAll 只会往请求体拼 query / sort / _source / from / size /
highlight / aggs / script_fields 七样东西（db/elasticSearch/
elasticSearch.go FindAll L646-706），collapse 连导出字段都没有
（不像 Aggs 是导出字段，可以直接赋值注入复杂聚合体）。

【具体例子】
旧代码真实用法（extend/DaiEs/elasticSearch.go L700-701）：

    if unique != "" { //去重，去重不支持统计
        query["collapse"] = map[string]string{"field": unique}
    }

unique 是查询函数的形参（elasticSearch.go L674）：

    func SearchNoCount(yearList []string, mustList []map[string]interface{},
        notMustList []map[string]interface{}, shouldList []map[string]interface{},
        aggList map[string]interface{}, filterList []map[string]interface{},
        unique string, sort string, sortType string, from int, size int) ([]map[string]interface{}, error)

标讯/公司类列表查询都经这个参数传去重字段。

【业务影响点】
标讯列表按 cid（公司 id）collapse——同一公司发布 100 条标讯，
列表里只占 1 条，翻页时不会出现"满屏同一家公司"。涉及 search
公司类列表的 unique 去重场景。

【拆解评估结论】
不可行。用 terms 聚合 + top_hits 替代，分页语义完全不同（聚合
分页要 composite cursor，且拿不到精确总条数）；全量拉回 Go 去重，
数据量不可行。

【框架改法建议】
加 SetCollapse(field string)，FindAll 拼
queryDSL["collapse"] = map[string]string{"field": field}。


二、ESDb 缺 track_total_hits（大计数精确值）

【goDai 现状】
FindAll 的 TotalCount 直接取 hits.total.value（elasticSearch.go
L751-759）。ES 7+ 默认 total 上限 10000：超过时返回
value=10000, relation="gte"——goDai 拿到的就是 10000，列表页
超过 1 万条时计数恒显示 10000。

【具体例子】
旧代码真实用法（extend/DaiEs/elasticSearch.go L199、
mohurdSearch.go L82）：

    if count >= 10000 {
        searchBody["track_total_hits"] = true  // 二次请求精确统计
    }

【业务影响点】
search/mohurd 列表的大计数场景：命中超过 1 万时前端看到的总数
是错的（恒为 10000）。

【拆解评估结论】
已有合规绕法——列表 FindAll + 计数单独 FindCount（goDai 的
FindCount 走 esapi CountRequest，本身就是精确值，无 1 万上限）。
只是多一次请求；框架加上 track_total_hits 后可一次请求拿到精确
total，属优化项、非阻塞项。

【框架改法建议】
ESDb 加导出字段 TrackTotalHits bool（或 setter），FindAll 拼
queryDSL["track_total_hits"] = true。


三、ESDb 缺 timeout（服务端查询超时）

【goDai 现状】
goDai 的 SearchRequest（elasticSearch.go L702-706）没设 timeout
字段，也没有任何入口。

【具体例子】
旧代码真实用法（extend/DaiEs/elasticSearch.go L703）：

    query["timeout"] = "5s"

服务端 5 秒后返回"已查到的部分结果"（不报错）。注意这和 HTTP
客户端超时不是一回事——客户端超时是直接报错，服务端 timeout
是优雅降级拿部分数据。

【业务影响点】
标讯年份多索引（跨多个 zhaobiao_YYYY 索引）聚合查询较慢，缺这
个能力只是少了"慢查询熔断"，不影响正确性。

【拆解评估结论】
无正确性问题，低优先级。

【框架改法建议】
SearchRequest 加 Timeout 字段（低优先级）。


四、字段校验把组合表达式里的星号判为非法（COUNT(*) / SUM(*) 表达不了）

【goDai 现状】
mysql/CH 共用同一套 validFieldRegex（db/mysql/tool.go、
db/clickhouse/tool.go L32-42），非法字符判定为
[^a-zA-Z0-9_\s().,'"]——星号不在白名单；isValidField（tool.go
L73-93）只对"整个 field 恰好是一个星号"特判放行。

【具体例子】

    ch.SetField("COUNT(*) AS cnt").SetTable("ztb").FindAll(ctx)
    // Err: 查询字段[COUNT(*) AS cnt]包含非法字符，存在注入风险

    ch.SetTable("ztb").SetWhere("xmdw = ?", name).FindCount(ctx)
    // 更隐蔽：不 SetField 时 FindCount 内部拼 "COUNT(*) AS count"
    // （mysql.go L526-530），再过 FindAll 的字段校验——必然报错。
    // 所有 FindCount 必须先 SetField 非空列

【业务影响点】
常规统计字段写法（COUNT(*)、SUM(*)）无法用链式表达。本条详见
第五条（FindCount 已单列）。

【拆解评估结论】
COUNT(*) 有等价替代（本次整改实际用法）：

    COUNT(*)                -> COUNT(nid)           nid 主键非空，每行都计
    COUNT(*)（phone 表）    -> COUNT(xmdw)          xmdw 是等值条件键列，恒非空
    分组计数(GROUP BY zbdw 外层 COUNT)
                            -> COUNT(DISTINCT zbdw) 组数 = 去重数

无法替代的：SUM(*)——语义是"对所有数值列求和"，结果依赖表结构
（列的顺序和数量），没有任何单一表达式能等价重建。

业务侧真实代码实例（ldApiServer GetCompanyCompetitorData，现为
保留手写的例外分支，等本条修好即可切链式）：

    // setting 空配置分支（companyagg.go）：count(*)/SUM(*) 无法过 SetField 校验
    result, err = m.chQueryRows("SELECT zbdw AS companyName,argMax((title,nid,release_time),release_time) AS arg,count(*) AS markCount,SUM(*) AS amount FROM ztb WHERE zbdw!='' AND arrayExists(x -> x=?, zbhx) AND index_name IN('"+strings.Join(yearList, "','")+"') GROUP BY zbdw LIMIT "+strconv.Itoa((page-1)*pageSize)+","+strconv.Itoa(pageSize), []interface{}{params["company_name"]})

而 setting 有配置的分支已切链式成功（供对照，说明其余部分链式
完全可用）：

    field := "zbdw AS companyName,argMax((title,nid,release_time),release_time) AS arg,count(infoid2 IN(" + settingId + ")) AS markCount,SUM(CASE WHEN infoid2 IN(" + settingId + ") THEN jg ELSE 0 END) AS amount"
    ch2.SetTable("ztb").SetField(field).SetWhere("zbdw != ?", "").
        SetWhere("arrayExists(x -> x = ?, zbhx)", params["company_name"]).
        SetWhereIn("index_name", yearArgs...).SetGroup("zbdw").
        SetLimit(int64((page-1)*pageSize), int64(pageSize)).FindAll(context.Background())

【框架改法建议】
字段校验放行"聚合函数内的星号"（如正则特判 COUNT(*) / SUM(*)
整体），不影响注入防护。


五、FindCount 不先 SetField 必然报错（内部拼的 COUNT(*) 过不了自己的校验）

【goDai 现状】
源码位置：db/mysql/mysql.go FindCount L526-531、
db/clickhouse/clickhouse.go FindCount L496-501，两处逻辑相同：

    if db.Field == "" {
        db.Field = "COUNT(*) AS count"    // 拼出来的表达式含星号
    } else {
        db.Field = "COUNT(" + db.Field + ") AS count"
    }
    db.Limit = "1"
    db.FindAll(ctx)                       // FindAll 再校验字段合法性

FindAll 的 isValidField 校验 COUNT(*) AS count 时，星号命中非法
字符类，必然报错。也就是说：调用方不 SetField 直接 FindCount 取
全表计数，这个最普通的用法在当前框架下走不通。

【具体例子】
最小复现（mysql 同理）：

    ch, _ := clickhouse.GetClickhouseDB("default")
    count, err := ch.SetTable("ztb").SetWhere("1=1").FindCount(context.Background())
    // 预期：全表 count
    // 实际 err：查询字段[COUNT(*) AS count]包含非法字符，存在注入风险

而 SetField 一个非空列就能跑：

    count, err := ch.SetTable("ztb").SetField("nid").SetWhere("1=1").FindCount(ctx)
    // 实际执行 SELECT COUNT(nid) AS count ... LIMIT 1

业务侧真实代码实例（ldApiServer 现在的合规写法，逐处规避）：

    // companydata.go chTelCount：SetField(键列) 后 FindCount
    count, err := ch.SetTable(table).SetField(keyCol).
        SetWhere(keyCol+" = ?", companyName).SetWhere(telCol+" != ?", "").
        FindCount(context.Background())

【业务影响点】
ldApiServer 存量代码已全部带 SetField 规避（逐处排查过），本条
属防误用加固——任何新代码或框架其他使用方，按直觉写 FindCount
就会踩中。

【拆解评估结论】
无需拆解，属框架自身闭环问题。

【框架改法建议】（二选一）
    1. FindCount 空 Field 时改拼 COUNT(1) 或 COUNT(主键) 等可过
       校验的形式；
    2. 或 FindCount 不重拼 Field 再转 FindAll，单独执行计数语句。


六、CH SetGroup 仅标识符（函数分组不行）

【goDai 现状】
validGroupRegex（db/clickhouse/tool.go L61，mysql 同款）是
^[a-zA-Z_][a-zA-Z0-9_]*(\.[a-zA-Z_][a-zA-Z0-9_]*)?(,...)*$——
只允许字段名 / 点 / 逗号，函数、别名、常量都不行。

【具体例子】

    ch.SetGroup("arrayJoin(zbhx)")
    // Err: GROUP BY子句[arrayJoin(zbhx)]包含非法字符，存在注入风险

同理 GROUP BY toStartOfHour(ts)、GROUP BY 1 也都过不了。

【业务影响点】
arrayJoin(zbhx) 是把数组列展开成多行：一条标讯有 3 个候选中标
单位，就展开成 3 行再分组计数。业务上是"候选单位（waitBid）维度
的中标统计"（GetCompanyRelationData/V2 的 method=waitBid 分支）。

【拆解评估结论】
不可行。展开后行数不可控，拉回 Go 去重计数等于每次全量传输，
性能反而更差——语义不可拆，保留手写执行器（companyagg.go 已
注释登记，共 2 处）。

业务侧真实代码实例（ldApiServer GetCompanyRelationData waitBid
分支，现为保留手写的例外，等本条修好即可切链式）：

    // 计数（companyagg.go）：GROUP BY arrayJoin 外层包 COUNT(*) 统计组数
    count, err = m.chQueryGroupCount("SELECT "+field+" FROM "+m.chPre()+table+
        " WHERE ("+strings.Join(where, ") AND (")+") GROUP BY "+group, args)
    // group = "arrayJoin(zbhx) AS zbhx"
    // field = "zbhx AS companyName,count(*) AS markCount,argMax((title,nid,release_time),release_time) AS arg"

    // 列表（companyagg.go）：
    dataList, err = m.chQueryRows("SELECT "+field+" FROM "+m.chPre()+table+
        " WHERE ("+strings.Join(where, ") AND (")+") GROUP BY "+group+
        " ORDER BY markCount DESC LIMIT "+strconv.Itoa((page-1)*pageSize)+","+strconv.Itoa(pageSize), args)

而 method=tender/bid/agent（GROUP BY xmdw/zbdw/dldw 普通字段）已
全部切链式成功（供对照）：

    count, err = ch.SetTable(table).SetField("DISTINCT "+group).
        SetWhere(whereStr, args...).FindCount(context.Background())
    ch2.SetTable(table).SetField(field).SetWhere(whereStr, args...).
        SetGroup(group).SetOrder("markCount DESC").
        SetLimit(int64((page-1)*pageSize), int64(pageSize)).FindAll(context.Background())

【框架改法建议】
SetGroup 放行函数分组（至少白名单 arrayJoin(...)），或提供
独立 setter。


七、SetWhere 危险字子串误伤（update_time 踩雷）

【goDai 现状】
mysql.go L229-235 / clickhouse.go L203-209，SetWhere 一进来就做
前置检查：

    dangerousKeywords := []string{"DROP", "ALTER", "TRUNCATE", "DELETE", "INSERT", "UPDATE", "EXEC"}
    for _, kw := range dangerousKeywords {
        if strings.Contains(strings.ToUpper(tpl), kw) {   // 子串匹配，不是单词匹配
            db.Err = fmt.Errorf("条件模板包含非法关键字：%s", kw)
            return db
        }
    }

【具体例子】

    ch.SetWhere("update_time >= ?", sTime)   // Err: 条件模板包含非法关键字：UPDATE
                                              // 因为 "UPDATE_TIME" 含 "UPDATE" 子串
    ch.SetWhere("deleted = ?", "N")           // Err: ...DELETE（"DELETED" 含 "DELETE"）
    ch.SetWhere("insert_time <= ?", eTime)    // Err: ...INSERT

注意区分两道检查：FindAll 里的 isValidWhere 正则其实只禁
"UPDATE\s+"（UPDATE 后跟空白）这类真注入形态，update_time 是
能过它的——字段名死在 SetWhere 的第一道 Contains 前置检查，
根本到不了第二道。

【业务影响点】
已迁移代码 grep 过一遍，存量 where 条件无踩雷。但后续批次（尤其
common 的 43 actions）旧 SQL 里若有 WHERE update_time >= ...，
迁到链式必被拦。

业务侧真实旧 SQL 实例（ldApiServer 待迁移批次里的原句，
extend/task/home.go L36——这条同时踩两个雷：UPDATE 子串 + 引号
字面量，迁链式时 update_time 和 status='Y' 都过不了 SetWhere）：

    sqlStr := "SELECT * FROM " + dbPre + "company_fine WHERE status='Y' AND update_time<=" +
        strconv.FormatInt(now.Unix()-3600*6, 10) + " ORDER BY sort DESC,id DESC LIMIT 27"

按链式应写成（本条修好后即可成立）：

    mdb.SetTable("company_fine").
        SetWhere("status = ?", "Y").
        SetWhere("update_time <= ?", now.Unix()-3600*6).
        SetOrder("sort DESC,id DESC").SetLimit(0, 27).FindAll(context.Background())

【拆解评估结论】
无绕法——模板在 SetWhere 入口就被拦，到不了执行层；受影响的那
条 SQL 只能整体保留手写。改框架后即可切回链式。

【框架改法建议】
Contains 改单词边界匹配，或先按标识符分词再比对：

    var dangerousKeywordsRe = regexp.MustCompile("(?i)\\b(DROP|ALTER|TRUNCATE|DELETE|INSERT|UPDATE|EXEC)\\b")

说明：\b 是 ASCII 词边界，下划线算词字符——UPDATE_TIME 中
UPDATE 后面跟下划线、不构成边界、放行；UPDATE table SET 中
UPDATE 后面是空格、构成边界、拦截。deleted 同理放行，DELETE FROM
同理拦截。update_time 这类字段名即可放行，真注入依然拦得住。


附一：同族限制顺带说明（这次整改已实际绕过的）

where 模板还禁引号字面量（xmdh!='' 写不了，改 xmdh != ? 传空串）、
禁嵌套 SELECT...FROM 和 CONCAT(/CHAR(/ASCII(/GROUP_CONCAT(
（参数化模糊查询走 LIKE ? 传 "%xx%"）。这些限制合理，已按参数化
方式迁移完毕，无需修改。


附二：等框架改好后 ldApiServer 侧对应的动作

    1  SetCollapse            search 公司类 unique 去重切 ES 链式执行器
    2  track_total_hits       search/mohurd 列表大计数二次请求可合并
    3  timeout                标讯年份多索引查询补服务端超时
    4  字段校验放行聚合内星号  CompetitorData setting 空分支切链式，删 1 处手写
    5  FindCount 修复         存量已全部带 SetField 规避，属防误用加固
    6  SetGroup 放行函数      RelationData V1/V2 waitBid 切链式，删 2 处手写
    7  SetWhere 改 \b 匹配    后续批次 update_time 等条件可直接链式
