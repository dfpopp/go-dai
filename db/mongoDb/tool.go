package mongoDb

import (
	"fmt"
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
func (m *Db) IdToObjectID(id string) (primitive.ObjectID, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		m.Err = fmt.Errorf("转换ObjectID失败: %v", err)
		return primitive.NilObjectID, m.Err
	}
	return oid, nil
}

// ObjectIDToString ObjectID转字符串
func ObjectIDToString(oid primitive.ObjectID) string {
	return oid.Hex()
}
