# goDai 待框架支持清单（二期）

> 背景：一期 9 条（ES SetCollapse/SetTrackTotalHits/SetSearchTimeout、AggsData 取数、聚合内星号、函数分组、SetWhere 词边界、FindCount 空 Field、mongo Aggregate 清态）已于 2026-09-23 全部修复核实，详见 [goDai待框架支持清单.md](./goDai待框架支持清单.md)。
>
> 本文是业务系统迁移实战中**新发现的 2 个框架能力缺口**：
>
> | # | 数据库 | 缺口 | 影响 | 紧迫度 |
> |---|---|---|---|---|
> | 1 | ClickHouse | 行级展开（ARRAY JOIN）与"分组后计数"无链式表达 | 公司竞买关系图 waitBid 分支外层计数，当前手写原始 SQL | 高（唯一一处无法链式化的存量） |
> | 2 | MySQL | UPSERT（INSERT ... ON DUPLICATE KEY UPDATE）原语缺失 | iOS 内购绑定映射表等幂等写入场景 | 中（可拆解落地，但属通用原语缺失） |
>
> 每条含：业务场景、正在生产的真实 SQL、现有链式方法逐项尝试与失败原因、最小复现、建议改法（精确到 goDai 源码位置与实现要点）、改造后调用方切换样例。
>
> **✅ 实现状态（2026-09-23，goDai 侧）**
>
> - **缺口一**：按方案 A 实现 `SetArrayJoin(expr)` / `SetLeftArrayJoin(expr)`（`db/clickhouse/clickhouse.go` + `tool.go` 新增 `isValidArrayJoin` 校验；ARRAY JOIN 拼装点为 FetchSql / FindAll 两处，`ToString` 不拼 SQL 无需插入；`clearData` 已同步清空）。方案 B（FindGroupCount）暂未实现，普通分组计数仍走 `SetField("DISTINCT "+group)` + `FindCount`。
> - **缺口二**：按评审意见**未采用**独立的 `InsertOnDuplicate` 方法，改为 `SetUpsert(cols...)` + `Insert` / `InsertAll` 扩展（`db/mysql/mysql.go`）：`Insert` 生成 `col = ?` 参数化形式，`InsertAll` 生成 `col = VALUES(col)` 逐行取值，事务内同样生效。调用方样例：`db.SetTable("ios_uid_map").SetUpsert("uid", "product_id", "account").Insert(ctx, data)`。

---

## 缺口一：ClickHouse 行级展开（ARRAY JOIN）与"分组后计数"能力缺失

### 1.1 业务场景

标讯主表 `ztb`（ClickHouse）中，`zbdw` 是单一中标单位列，`zbhx` 是 **Array(String) 数组列**（候选单位集合，一个标讯可含多个）。业务"竞买关系"页签 waitBid（方法名 `GetCompanyRelationData`，公司维度聚合列表）需要：

- 筛选：`arrayExists(x -> x = '某公司', zbhx)`——目标公司出现在数组中；
- 分组：把数组**逐元素展开成行**（ClickHouse 的 arrayJoin 语义），按展开后的每个元素分组；
- 列表：每组取公司名、条数 `count(*)`、最新一条标讯 `argMax(...)`；
- **计数：分页需要总组数 =「数组元素展开去重后的个数」**。

其中行查询/列表已经用 goDai 链式实现（一期修复了函数分组后 `SetGroup("arrayJoin(zbhx) AS zbhx")` 可过校验），**唯一卡住的是计数**。

### 1.2 现状手写 SQL（正在生产的真实语句）

内层（分组行查询，已链式化，此处展示等价 SQL）：

```sql
SELECT zbhx AS companyName, count(*) AS markCount,
       argMax((title, nid, release_time), release_time) AS arg
FROM ld_ztb
WHERE (index_name IN ('2026','2025')) AND (arrayExists(x -> x = '某公司', zbhx))
GROUP BY arrayJoin(zbhx) AS zbhx
ORDER BY markCount DESC
LIMIT 20, 20
```

外层计数（**目前唯一手写原始 SQL 的地方**，参数化 `?` 透传，`QueryContext` 执行）：

```sql
SELECT COUNT(*) AS count FROM (
    SELECT zbhx AS companyName, count(*) AS markCount,
           argMax((title, nid, release_time), release_time) AS arg
    FROM ld_ztb
    WHERE (index_name IN (?,?)) AND (arrayExists(x -> x = ?, zbhx))
    GROUP BY arrayJoin(zbhx) AS zbhx
) LIMIT 1
```

计数语义 = 内层 GROUP BY 结果的**组数**（展开去重后的候选单位个数），不是原始行数。

### 1.3 现有链式方法逐项尝试与失败原因

对 `db/clickhouse/clickhouse.go` 全部 24 个导出方法逐一核对（SetTable/SetAlias/SetField/SetWhere/SetWhereOr/SetWhereIn/SetOrder/SetGroup/SetJoin/SetLimit/FetchSql/FindAll/FindCount/Find/Insert/InsertAll/Update/UpdateBySet/SetInc/Delete/Exec/ToString/clearData）：

| 尝试 | 结果 | 失败原因 |
|---|---|---|
| ① `SetField("DISTINCT arrayJoin(zbhx)")` + `FindCount` | 语义不成立 | `FindCount`（L488-539）把 Field 包装成 `COUNT(DISTINCT arrayJoin(zbhx)) AS count`。ClickHouse 的 arrayJoin 是行级展开函数，只在 SELECT 列表表达式位置生效；放进聚合函数参数内不会展开行——得到的是原始行去重或直接报错，均非"展开后组数"。ClickHouse 官方指定的行级展开语法是 **ARRAY JOIN 子句** |
| ② `SetGroup("arrayJoin(zbhx) AS zbhx")` + `FindCount`（Field 传 `zhbx`） | 结果错误 | 生成 `SELECT COUNT(zhbx) AS count ... GROUP BY arrayJoin(zbhx)`——这是"每组成员数"，返回**多行**（每组一行），`FindCount` 只读第一行，拿到的是第一组的成员数而非组数 |
| ③ `SetField("DISTINCT x")` + `FindCount`，分组键改用 SetWhere 携带子查询 | 校验拦截 | `SetWhere` 模板禁嵌套 `SELECT...FROM`（危险字校验），子查询塞不进 WHERE |
| ④ 用 `Exec` 执行计数 SQL | 无法取回结果 | CH 的 `Exec`（L828）走 `ExecContext` 面向变更语句，SELECT 行集读不回 |
| ⑤ 分页拉全部分组行 → Go 里 `len()` 计数 | 语义可行，**性能不可接受** | `FindAll` 默认 LIMIT 500、`SetLimit` num 上限 1000（一期已确认的硬约束）；一个公司的候选单位分组数轻松上千，需要多次往返只为数一个数。属"拆解后性能明显更差"的情形，因此按规范保留手写例外 |

### 1.4 最小复现

```go
// 场景：count 展开后分组的组数
ch, _ := baseModel.GetClickhouseDb("default")

// 尝试 ①：DISTINCT + arrayJoin 进 COUNT —— 语义不成立
ch.SetTable("ztb").
    SetField("DISTINCT arrayJoin(zbhx)").
    SetWhere("arrayExists(x -> x = ?, zbhx)", "某公司").
    FindCount(context.Background())
// 生成：SELECT COUNT(DISTINCT arrayJoin(zbhx)) AS count FROM ztb WHERE ...
// 期望：数组元素展开去重后的个数；实际：arrayJoin 不在聚合参数内展开，返回原始行去重数或报错

// 尝试 ②：GROUP BY 函数分组 + FindCount —— 多行只读第一行，结果错误
ch.SetTable("ztb").
    SetField("zhbx").
    SetWhere("arrayExists(x -> x = ?, zbhx)", "某公司").
    SetGroup("arrayJoin(zbhx) AS zbhx").
    FindCount(context.Background())
// 生成：SELECT COUNT(zhbx) AS count FROM ztb WHERE ... GROUP BY arrayJoin(zbhx) AS zbhx
// 期望：组数（1 行 1 值）；实际：每组一行（N 行），FindCount 只读第一行 → 值为第一组成员数

// 期望能写成的形态（ClickHouse 原生语法，等价于手写子查询计数）：
// SELECT COUNT(DISTINCT x) AS count FROM ld_ztb ARRAY JOIN zbhx AS x
//   WHERE index_name IN (?,?) AND arrayExists(x -> x = ?, zbhx)
// 或：SELECT COUNT(*) AS count FROM (SELECT ... GROUP BY arrayJoin(zbhx) AS zbhx) LIMIT 1
```

### 1.5 建议改法 A（推荐）：新增 `SetArrayJoin`

**API 设计**：

```go
// SetArrayJoin 设置 ARRAY JOIN 行级展开子句（ClickHouse 特色语法，位于 FROM/JOIN 之后、WHERE 之前）
// expr 示例："zhbx AS x"（普通展开）或 "zhbx"（原地覆盖）；
// 左展开用 SetLeftArrayJoin（生成 LEFT ARRAY JOIN）
func (db *ClickhouseDb) SetArrayJoin(expr string) *ClickhouseDb
func (db *ClickhouseDb) SetLeftArrayJoin(expr string) *ClickhouseDb
```

**SQL 拼装位置**：`FetchSql`（L284-361）与 `FindAll`（L362 起）两处的拼装序列当前为：

```
SELECT {field} FROM {table}[ AS alias][ {RelationList: JOIN ...}][ WHERE ...][ GROUP BY ...][ ORDER BY ...][ LIMIT ...]
```

新增字段 `ArrayJoin string`，插在 **RelationList 之后、WHERE 之前**（ClickHouse 语法位置）：

```
SELECT {field} FROM {table}[ AS alias][ {JOIN ...}][ ARRAY JOIN {expr}][ WHERE ...][ GROUP BY ...][ ORDER BY ...][ LIMIT ...]
```

**实现要点**：
1. 校验：新增 `isValidArrayJoin(expr)`，白名单允许 `[a-zA-Z0-9_.\s(\),]`（覆盖 `zhbx AS x`、`arrayJoin(zbhx)` 形态即可，不必开放任意表达式）；
2. `clearData()`（L858）同步清空 `ArrayJoin` 字段；
3. `FetchSql`/`FindAll`/`ToString` 三处拼装点都要插入（保持与 GROUP BY 同样的顺序）。

**改造后调用方代码（waitBid 计数 + 列表全部链式化）**：

```go
// 计数：ARRAY JOIN 展开后去重计数（FindCount 对 SetField 原样包装，DISTINCT 写法已被现有普通分组计数验证可用）
count, err := ch.SetTable("ztb").
    SetArrayJoin("zhbx AS x").
    SetField("DISTINCT x").
    SetWhere("index_name IN(?,?)", year1, year2).
    SetWhere("arrayExists(x -> x = ?, zbhx)", companyName).
    FindCount(context.Background())

// 列表：ARRAY JOIN 后按 x 分组（不再需要 GROUP BY arrayJoin 写法）
ch2.SetTable("ztb").
    SetArrayJoin("zhbx AS x").
    SetField("x AS companyName, count(nid) AS markCount, argMax((title,nid,release_time), release_time) AS arg").
    SetWhere("index_name IN(?,?)", year1, year2).
    SetWhere("arrayExists(x -> x = ?, zbhx)", companyName).
    SetGroup("x").
    SetOrder("markCount DESC").
    SetLimit(int64((page-1)*pageSize), int64(pageSize)).
    FindAll(context.Background())
```

落地后可删除手写执行器 `chQueryGroupCount`/`chQueryRowsRaw`（company/model.go L64-118），CH 原始驱动调用清零。

### 1.6 建议改法 B（备选）：新增 `FindGroupCount`

**API 设计**：

```go
// FindGroupCount 分组结果计数：外层包 COUNT(*) 统计"分组后的组数"
// 等价 SELECT COUNT(*) AS count FROM (SELECT {field|*} FROM {table} [WHERE ...] GROUP BY {group}) LIMIT 1
func (db *ClickhouseDb) FindGroupCount(ctx context.Context) (int64, error)
```

**实现要点**：
1. 复用 FetchSql 的拼装逻辑生成内层 SQL（field 缺省 `*`），包一层 `SELECT COUNT(*) AS count FROM (...) LIMIT 1`；
2. `?` 参数与 WhereArgs 原样透传（保持参数化）；
3. 结果解析复用 `FindCount` 的类型 switch（ClickHouse COUNT 返回 uint64，L507-535 已有完整处理）；
4. 安全性：子查询内容全部经过既有 field/group/where 校验，新暴露面只有外层常量包装。

**改造后调用方代码**：

```go
// waitBid 计数（内层 field/group 与行查询完全同源，不再需要 COUNT(DISTINCT) 变体）
count, err := ch.SetTable("ztb").
    SetField("zhbx AS companyName, count(nid) AS markCount, argMax(...) AS arg").
    SetWhere("index_name IN(?,?)", year1, year2).
    SetWhere("arrayExists(x -> x = ?, zbhx)", companyName).
    SetGroup("arrayJoin(zbhx) AS zbhx").
    FindGroupCount(context.Background())
```

**附带收益**：普通分组计数（现为 `SetField("DISTINCT "+group)` + `FindCount` 的特例用法）可统一改走 `FindGroupCount`，语义更直白。

### 1.7 两方案对比

| | 方案 A：SetArrayJoin | 方案 B：FindGroupCount |
|---|---|---|
| 表达力 | 高——ARRAY JOIN 是 ClickHouse 行级展开的官方语法，计数与列表查询都能标准化 | 仅解决"分组后计数" |
| 实现成本 | 中——新增字段 + 三处拼装点 + 校验函数 | 低——包一层子查询，复用 FetchSql |
| 普通分组计数 | 不影响（仍走 COUNT(DISTINCT)） | 可统一收编 |
| 最少改动 | 二选一即可解除手写例外 | 同左 |

**推荐**：实现 A（表达力优先）；若求最小改动则实现 B。两者不冲突，可并行。

---

## 缺口二：MySQL UPSERT（INSERT ... ON DUPLICATE KEY UPDATE）原语缺失

### 2.1 业务场景

iOS 内购"用户 ↔ 原始交易ID"绑定映射表 `ios_uid_map`：

```sql
CREATE TABLE ios_uid_map (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    uid VARCHAR(64) NOT NULL,
    original_transaction_id VARCHAR(128) NOT NULL,
    product_id VARCHAR(128) NOT NULL,
    account VARCHAR(128) NOT NULL,
    create_time INT NOT NULL,
    UNIQUE KEY uk_original (original_transaction_id)
);
```

`original_transaction_id` 全表唯一（同一苹果订阅只能绑定一个用户）。绑定写入是**幂等 upsert**：首次插入，重复绑定（用户换账号重新同步同一订阅）则覆盖 uid/product_id/account。同类场景还有各类"按业务唯一键幂等落库"的写入。

### 2.2 现状 SQL（正在生产的真实语句）

```sql
INSERT INTO ios_uid_map (uid, original_transaction_id, product_id, account, create_time)
VALUES ('10001', '2000012345678901', 'com.xxx.member.12m', 'userA', 1758624000)
ON DUPLICATE KEY UPDATE uid = VALUES(uid), product_id = VALUES(product_id), account = VALUES(account)
```

### 2.3 现有链式方法逐项尝试与失败原因

对 `db/mysql/mysql.go` 全部导出方法核对（ToBegin/Rollback/Commit/SetTable/SetAlias/SetField/SetWhere/SetWhereOr/SetWhereIn/SetOrder/SetGroup/SetJoin/SetLimit/FetchSql/FindAll/FindCount/Find/Insert/InsertAll/Update/UpdateBySet/SetInc/Delete/Exec/ToString/clearData）：

| 尝试 | 结果 | 失败原因 |
|---|---|---|
| ① `Insert`（L585-634） | 唯一键冲突报错 | 生成纯 `INSERT INTO ...`，无任何冲突处理子句 |
| ② `Update`（L722）/ `UpdateBySet`（L717） | 不是 upsert | 只生成 `UPDATE ... WHERE ...`，记录不存在时影响 0 行，无法兜底插入 |
| ③ `Exec` 执行原始 upsert SQL | 能跑，但**正是本项目要清零的"原生 SQL 写法"** | 框架有 Exec 兜底通道，但这属于绕开链式层，每处调用都要手拼 SQL 与参数顺序，失去链式层的校验与事务自动绑定一致性（Exec 事务内经 Tx 判断也可用，但 SQL 文本脱离结构化拼装） |
| ④ `FindCount` 判存在 → `Insert` 或 `Update` | **语义可拆**（见 2.4） | 非原子：并发窗口 + 需要调用方处理冲突降级 |

### 2.4 不依赖框架的拆解方案（可落地，作为框架缺位期的过渡）

```
1. FindCount(original_transaction_id = ?)            → 判存在
2. 不存在 → Insert(...)；存在 → Update(...)（WHERE original_transaction_id = ?）
3. Insert 撞唯一键（MySQL 错误 1062）→ 降级执行 Update（并发兜底）
```

- **并发分析**：两请求并发判断"不存在"→ 双 Insert → 第二个撞唯一键报 1062 → 降级 Update 补偿 → 最终一致，无脏数据。upsert 的覆盖语义（后写赢）保持。
- **性能**：多一次 FindCount 往返（毫秒级），该场景（用户同步 iOS 订阅，且有 5 秒防重锁）完全无感。
- **结论**：能落地、非阻塞；但"每处 upsert 都要拆两条 + 1062 兜底"属于通用原语缺失导致的结构性样板代码，建议框架补原语后收敛。

### 2.5 最小复现

```go
mdb, _ := baseModel.GetMysqlDb("site")

// 期望：一条链式完成 upsert（唯一键 uk_original 冲突时更新指定列）
// 现状：无任何链式方法能生成 ON DUPLICATE KEY UPDATE
mdb.SetTable("ios_uid_map").Insert(ctx, map[string]interface{}{
    "uid": "10001", "original_transaction_id": "2000012345678901",
    "product_id": "com.xxx.member.12m", "account": "userA", "create_time": 1758624000,
})
// 实际：重复执行报 Error 1062 Duplicate entry，无法表达幂等语义
```

### 2.6 建议改法：新增 `InsertOnDuplicate`

**API 设计**：

```go
// InsertOnDuplicate 插入并在唯一键冲突时按 updateData 更新（MySQL ON DUPLICATE KEY UPDATE）
// data：插入列→值；updateData：冲突时更新的列→值（显式给定，不用 VALUES(col) 简写——该写法 MySQL 8.4 起弃用）
// 事务内调用自动走 db.Tx（与 Insert 一致）；返回 LastInsertId
func (db *MysqlDb) InsertOnDuplicate(ctx context.Context, data map[string]interface{}, updateData map[string]interface{}) (int64, error)
```

**生成 SQL 形态**：

```sql
INSERT INTO `ios_uid_map` (`uid`, `original_transaction_id`, `product_id`, `account`, `create_time`)
VALUES (?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE `uid` = ?, `product_id` = ?, `account` = ?
```

**实现要点**（可直接复用 Insert 的实现骨架）：
1. 列拆分/反引号/`?` 占位逻辑与 `Insert`（L601-615）一致；`updateData` 同样拆列，值**追加**到 `values` 尾部（占位符全参数化，无拼接注入面）；
2. `updateData` 为空时退化为纯 Insert（或直接报错"updateData 不能为空"，与 Update 的空参校验 L591 风格一致——建议报错，避免误用）；
3. 执行段照抄 `Insert` L618-624：`db.Tx != nil` 走 `Tx.ExecContext`，否则 `Db.ExecContext`（ToBegin 后自动进事务）；
4. 返回值：`LastInsertId`——upsert 命中更新分支时 MySQL 返回的是受影响行的首列 id，调用方按"插入才关心 id"处理即可（文档注明该语义）；
5. `clearData(false)` 时机与 Insert 保持一致（L586）。

**落地后调用方切换样例**（即当前手写 SQL 的归位形态）：

```go
// 批次 19 重写 iosBind 时：BindIosUidMap 由 1062 兜底拆解切换为一条链式
_, err := mdb.SetTable("ios_uid_map").InsertOnDuplicate(ctx,
    map[string]interface{}{
        "uid": uid, "original_transaction_id": original,
        "product_id": productId, "account": account, "create_time": now,
    },
    map[string]interface{}{
        "uid": uid, "product_id": productId, "account": account,
    })
```

---

## 附：涉及 goDai 源码位置速查

| 位置 | 与本清单的关系 |
|---|---|
| `db/clickhouse/clickhouse.go` FetchSql L284-361 / FindAll L362 | SQL 拼装序列，ARRAY JOIN 插入点 = RelationList 之后、WHERE 之前 |
| `db/clickhouse/clickhouse.go` FindCount L488-539 | Field 原样包装为 `COUNT(field) AS count`；`DISTINCT x` 前缀写法已验证可用；uint64 结果解析已完备（新计数原语可直接复用） |
| `db/clickhouse/clickhouse.go` SetJoin L251（RelationList） | 与 ARRAY JOIN 同为"FROM 后附加子句"，实现可参照 |
| `db/clickhouse/clickhouse.go` clearData L858 | 新增字段需同步清空 |
| `db/clickhouse/clickhouse.go` Exec L828 | ExecContext 无行集读取，SELECT 不能走（缺口一④的根因） |
| `db/mysql/mysql.go` Insert L585-634 | InsertOnDuplicate 的实现骨架（列拆分/占位符/Tx 绑定/LastInsertId） |
| `db/mysql/mysql.go` ToBegin L168 / Update L722 | 事务绑定与更新语义参照 |
| `db/mysql/mysql.go` tool.go（isValidTable/isValidField 等） | 新增子句的校验函数实现参照 |

> 两条落地后：ClickHouse 原始驱动调用全清零（唯一手写例外解除）；MySQL 仅剩 extend/iosBind 过渡期的 *sql.DB 形态（批次 19 重写为链式后 `BaseModel.MysqlSqlDB()` 可删除）。
