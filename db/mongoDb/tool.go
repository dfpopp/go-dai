package mongoDb

import (
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GetData 返回原始查询结果
func (m *Db) GetData() ([]map[string]interface{}, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return m.Data, nil
}

// IdToObjectID 字符串转ObjectID
func IdToObjectID(id string) (primitive.ObjectID, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return primitive.NilObjectID, fmt.Errorf("转换ObjectID失败: %v", err)
	}
	return oid, nil
}

// ObjectIDToString ObjectID转字符串
func ObjectIDToString(oid primitive.ObjectID) string {
	return oid.Hex()
}
func MapToBsonD(sdata map[string]interface{}) bson.D {
	tdata := bson.D{}
	keys := make([]string, 0)
	for key, _ := range sdata {
		keys = append(keys, key)
	}
	for _, key := range keys {
		tdata = append(tdata, bson.E{Key: key, Value: sdata[key]})
	}
	return tdata
}
