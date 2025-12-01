package mongoDb

import (
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// PipelineBuilder 聚合管道构建器
type PipelineBuilder struct {
	pipeline mongo.Pipeline // 最终生成的管道
}

// NewPipelineBuilder 创建管道构建器实例
func NewPipelineBuilder() *PipelineBuilder {
	return &PipelineBuilder{
		pipeline: mongo.Pipeline{},
	}
}

// Match 添加$match阶段（查询条件）
// where: 匹配条件，如bson.D{{"year", bson.M{"$gte": 2020}}}
func (b *PipelineBuilder) Match(where bson.D) *PipelineBuilder {
	if len(where) > 0 {
		b.pipeline = append(b.pipeline, bson.D{{"$match", where}})
	}
	return b
}

// Group 添加$group阶段（分组规则）
// group: 分组规则，如bson.D{{"_id", "$name"}, {"amount", bson.D{{"$sum", "$amount"}}}}
func (b *PipelineBuilder) Group(group bson.D) *PipelineBuilder {
	if len(group) > 0 {
		b.pipeline = append(b.pipeline, bson.D{{"$group", group}})
	}
	return b
}

// Sort 添加$sort阶段（排序规则）
// sort: 排序规则，如bson.D{{"_id", 1}, {"amount", -1}}
func (b *PipelineBuilder) Sort(sort bson.D) *PipelineBuilder {
	if len(sort) > 0 {
		b.pipeline = append(b.pipeline, bson.D{{"$sort", sort}})
	}
	return b
}

// Project 添加$project阶段（字段投影）
// project: 投影规则，如bson.M{"product": "$_id", "amount": "$amount"}
func (b *PipelineBuilder) Project(project interface{}) *PipelineBuilder {
	if project != nil {
		b.pipeline = append(b.pipeline, bson.D{{"$project", project}})
	}
	return b
}

// Skip 添加$skip阶段（跳过条数）
func (b *PipelineBuilder) Skip(skip int64) *PipelineBuilder {
	if skip > 0 {
		b.pipeline = append(b.pipeline, bson.D{{"$skip", skip}})
	}
	return b
}

// Limit 添加$limit阶段（限制条数）
func (b *PipelineBuilder) Limit(limit int64) *PipelineBuilder {
	if limit > 0 {
		b.pipeline = append(b.pipeline, bson.D{{"$limit", limit}})
	}
	return b
}

// AppendStage 追加自定义管道阶段（如$lookup/$unwind等）
// stage: 自定义阶段，如bson.D{{"$lookup", bson.D{{"from", "table"}, {"localField", "id"}, {"foreignField", "fid"}, {"as", "data"}}}}
func (b *PipelineBuilder) AppendStage(stage bson.D) *PipelineBuilder {
	if len(stage) > 0 {
		b.pipeline = append(b.pipeline, stage)
	}
	return b
}

// Build 生成最终的聚合管道
func (b *PipelineBuilder) Build() mongo.Pipeline {
	return b.pipeline
}

// BuildAggPipeline 快速生成聚合管道的函数（无需创建Builder实例，直接传参）
// 参数：where(匹配), group(分组), sort(排序), project(投影), skip/limit(分页)
func BuildAggPipeline(where bson.D, group bson.D, sort bson.D, project interface{}, skip, limit int64) mongo.Pipeline {
	return NewPipelineBuilder().
		Match(where).
		Group(group).
		Sort(sort).
		Project(project).
		Skip(skip).
		Limit(limit).
		Build()
}
