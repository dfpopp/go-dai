package elasticSearch

import (
	"github.com/elastic/go-elasticsearch/v8"
	"net/http"
)

// BoolClauseType 定义Bool子句类型（约束合法的bool子句）
type BoolClauseType string

type ESDb struct {
	Client        *elasticsearch.Client // 复用全局数据库连接池
	DbPre         string                //表前缀
	GzipStatus    bool                  //响应内容是否开启gzip压缩
	Index         []string
	Id            string
	WhereQuery    map[string]interface{} // 查询条件（DSL）
	Aggs          map[string]interface{} // 聚合配置
	Sort          []string
	ExcludeSource []string
	Source        []string
	ScriptFields  map[string]interface{}
	From          int64
	Size          int64
	Highlight     map[string]interface{}
	Pk            string // 批量操作的主键字段（如"id"）
	BatchTimeout  int    //批量操作超时设置
	BulkActions   []string
	Data          []map[string]interface{}
	AggsData      map[string]interface{} // 新增：专存聚合结果
	TotalCount    int64
	Err           error
}
type DbObj struct {
	Client     *elasticsearch.Client // 复用全局数据库连接池
	Transport  *http.Transport
	Pre        string
	GzipStatus bool //响应内容是否开启gzip压缩
}

// HighlightOption 定义高亮配置的可选参数
// 字段说明：
//
//	PreTag:            高亮前置标签（必填，如 "<em>"）
//	PostTag:           高亮后置标签（必填，如 "</em>"）
//	FragmentSize:      每个高亮片段的最大字符长度（可选，0=使用ES默认值100；-1=返回完整字段内容）
//	NumberOfFragments: 返回的高亮片段最大数量（可选，0=使用ES默认值5；1=仅返回最匹配的1个片段）
type HighlightOption struct {
	FragmentSize      int    // 片段长度，0=使用ES默认
	NumberOfFragments int    // 片段数量，0=使用ES默认
	PreTag            string // 前置标签
	PostTag           string // 后置标签
}

// BulkResponse ES Bulk响应结构体（精准解析）
type BulkResponse struct {
	Took   int  `json:"took"`
	Errors bool `json:"errors"`
	Items  []struct {
		Index struct {
			ID     string `json:"_id"`
			Result string `json:"result"`
			Error  struct {
				Type   string `json:"type"`
				Reason string `json:"reason"`
			} `json:"error,omitempty"`
		} `json:"index"`
	} `json:"items"`
}

// BulkUpdateResponse Update响应结构体
type BulkUpdateResponse struct {
	Took   int  `json:"took"`
	Errors bool `json:"errors"`
	Items  []struct {
		Update struct {
			ID     string `json:"_id"`
			Result string `json:"result"`
			Error  struct {
				Type   string `json:"type"`
				Reason string `json:"reason"`
			} `json:"error,omitempty"`
		} `json:"update"`
	} `json:"items"`
}

// BulkDeleteResponse Bulk删除响应结构体
type BulkDeleteResponse struct {
	Took   int  `json:"took"`
	Errors bool `json:"errors"`
	Items  []struct {
		Delete struct {
			ID     string `json:"_id"`
			Result string `json:"result"`
			Error  struct {
				Type   string `json:"type"`
				Reason string `json:"reason"`
			} `json:"error,omitempty"`
		} `json:"delete"`
	} `json:"items"`
}
type DeleteByQueryResponse struct {
	Took     int64 `json:"took"`
	Deleted  int64 `json:"deleted"`
	Failures []struct {
		Index  string `json:"index"`
		Reason string `json:"reason"`
		Type   string `json:"type"`
	} `json:"failures"`
}
type UpdateByQueryResponse struct {
	Took     int64 `json:"took"`
	Updated  int64 `json:"updated"`
	Noops    int64 `json:"noops"` // 无更新的文档数
	Failures []struct {
		Index  string `json:"index"`
		Reason string `json:"reason"`
		Type   string `json:"type"`
	} `json:"failures"`
}

// AnalyzeRequest ES _analyze API 请求体（低版本客户端需手动构建）
type AnalyzeRequest struct {
	Analyzer string   `json:"analyzer"`
	Text     []string `json:"text"`
}
type AnalyzeResponse struct {
	Tokens []struct {
		Token       string  `json:"token"`        // 分词结果
		StartOffset float64 `json:"start_offset"` // 起始偏移量
		EndOffset   float64 `json:"end_offset"`   // 结束偏移量
		Type        string  `json:"type"`         // 分词类型
		Position    int     `json:"position"`     // 位置
	} `json:"tokens"`
	Error *struct { // 仅失败时存在
		RootCause []struct {
			Type   string `json:"type"`
			Reason string `json:"reason"`
		} `json:"root_cause"`
		Type   string `json:"type"`
		Reason string `json:"reason"`
	} `json:"error,omitempty"`
	Status int `json:"status,omitempty"` // 响应状态码
}
