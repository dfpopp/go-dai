package base

import (
	"github.com/dfpopp/go-dai/db/elasticSearch"
	"github.com/dfpopp/go-dai/db/mongoDb"
	"github.com/dfpopp/go-dai/db/mysql"
	"github.com/dfpopp/go-dai/db/redisDb"
	"github.com/dfpopp/go-dai/logger"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type BaseModel struct {
	Log logger.Logger // 日志实例
}

// Init 初始化服务层（框架自动调用）
func (m *BaseModel) Init() {
	m.Log = logger.GetLogger()
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
