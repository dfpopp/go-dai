package function

import (
	"bytes"
	"crypto/md5"
	"crypto/rc4"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/transform"
	"hash/crc32"
	"io/ioutil"
	"math"
	"math/rand"
	"net/url"
	"os"
	"os/user"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
)

func Ksort(sdata map[string]string) map[string]string {
	var keys []string
	tdata := make(map[string]string)
	for key, _ := range sdata {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		tdata[key] = sdata[key]
	}
	return tdata
}
func MapToBsonD(sdata map[string]interface{}, keys []string) bson.D {
	tdata := bson.D{}
	if len(keys) == 0 {
		for key, _ := range sdata {
			keys = append(keys, key)
		}
	} else {
		for key, _ := range sdata {
			if !InArray(key, keys) {
				keys = append(keys, key)
			}
		}
	}
	for _, key := range keys {
		tdata = append(tdata, bson.E{Key: key, Value: sdata[key]})
	}
	return tdata
}
func Md5(str string) string {
	h := md5.New()
	h.Write([]byte(str))
	return hex.EncodeToString(h.Sum(nil))
}
func Crc(str string) uint32 {
	return crc32.ChecksumIEEE([]byte(str))
}

func Random(length int, is_digital bool) string {
	var str string = ""
	seeds := make([]string, 10, 10)
	if is_digital == true {
		seeds = []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"}
	} else {
		seeds = []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m", "n", "o", "p", "q", "r", "s", "t", "u", "v", "w", "x", "y", "z"}
	}
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < length; i++ {
		str += seeds[rand.Intn(len(seeds))]
	}
	return str
}
func InArray(needle string, haystack []string) bool {
	if len(haystack) > 0 {
		for _, item := range haystack {
			if item == needle {
				return true
			}
		}
	}
	return false
}
func ArrayUnique(data any, key string) []interface{} {
	var datas []interface{}
	var newDatas []interface{}
	var keyDatas = make(map[string]interface{})
	switch reflect.TypeOf(data).Kind() {
	case reflect.Slice, reflect.Array:
		value := reflect.ValueOf(data)
		for i := 0; i < value.Len(); i++ {
			datas = append(datas, value.Index(i).Interface())
		}
		break
	}
	for _, value := range datas {
		switch reflect.TypeOf(value).String() {
		case "map[string]interface{}":
			mapValue := value.(map[string]interface{})
			keyDatas[mapValue[key].(string)] = value
			break
		case "map[string]string":
			mapValue := value.(map[string]string)
			keyDatas[mapValue[key]] = value
			break
		}
	}
	for _, value := range keyDatas {
		newDatas = append(newDatas, value)
	}
	return newDatas
}
func OneArrayUnique(data []string) []string {
	newData := make([]string, 0)
	valueMap := map[string]int{}
	for _, value := range data {
		if value != "" {
			valueMap[value] = 1
		}
	}
	for key, _ := range valueMap {
		newData = append(newData, key)
	}
	return newData
}
func ArrayChunk(data any, size int) [][]interface{} {
	var datas []interface{}
	var newDatas [][]interface{}
	switch reflect.TypeOf(data).Kind() {
	case reflect.Slice, reflect.Array:
		value := reflect.ValueOf(data)
		for i := 0; i < value.Len(); i++ {
			datas = append(datas, value.Index(i).Interface())
		}
		break
	}
	arrayLen := len(datas)
	for i := 0; i < int(math.Ceil(float64(arrayLen)/float64(size))); i++ {
		end := (i + 1) * size
		if end > arrayLen {
			end = arrayLen
		}
		newDatas = append(newDatas, datas[i*size:end])
	}
	return newDatas
}
func Base64_encode(str string) string {
	return base64.StdEncoding.EncodeToString([]byte(str))
}
func Rc4(keyStr string, str string) string {
	var key []byte = []byte(keyStr)        //初始化用于加密的KEY
	rc4Obj, _ := rc4.NewCipher(key)        //返回 Cipher
	rc4Str := []byte(str)                  //需要加密的字符串
	plaintext := make([]byte, len(rc4Str)) //
	rc4Obj.XORKeyStream(plaintext, rc4Str)
	//XORKeyStream方法将src的数据与秘钥生成的伪随机位流取XOR并写入dst。dst和src可指向同一内存地址；但如果指向不同则其底层内存不可重叠plaintext就是你加密的返回过来的结果了，注意：plaintext则为 base-16 编码的字符串，每个字节使用 2 个字符表示 必须格式化成字符串
	return string(plaintext)
}
func InputData(values url.Values) url.Values {
	var data = make(url.Values, 0)
	for key, value := range values {
		if len(value) > 0 {
			strArr := make([]string, 0)
			for _, v := range value {
				if v != "" {
					if InArray(key, []string{"uid", "id", "ucode", "url", "sign", "code", "iv", "encryptData"}) {
						strArr = append(strArr, v)
					} else {
						strArr = append(strArr, StrSafe(v))
					}
				}
			}
			if len(strArr) > 0 {
				data[key] = strArr
			}
		}
	}
	return data
}
func StrSafe(str string) string {
	str = styleRex.ReplaceAllString(str, "")
	str = scriptRex.ReplaceAllString(str, "")
	str = strings.TrimSpace(str)
	hanZiNum := 0
	for _, c := range str {
		if unicode.Is(unicode.Han, c) {
			hanZiNum++
		}
	}
	if hanZiNum == 0 {
		str = dangerRex.ReplaceAllString(str, "")
	}
	blackWord := strings.Split("select|update|delete|insert|truncate|declare|drop|execute|sleep", "|")
	strList := strings.Fields(str)
	newList := make([]string, 0)
	for _, v := range strList {
		nv := strings.ToLower(v)
		isFind := false
		for _, word := range blackWord {
			if nv == word {
				isFind = true
				break
			}
		}
		if isFind == false {
			newList = append(newList, v)
		}
	}
	str = strings.Join(newList, " ")
	if safeRex.MatchString(str) {
		return str
	} else {
		return ""
	}
}
func IntVal(str string) string {
	strArr := strings.Split(str, ",")
	for key, val := range strArr {
		num, err := strconv.Atoi(val)
		if err != nil {
			strArr[key] = "0"
		}
		if num != 0 {
			strArr[key] = val
		} else {
			strArr[key] = "0"
		}
	}
	return strings.Join(strArr, ",")
}
func FloatVal(str string) string {
	num, err := strconv.ParseFloat(str, 32)
	if err != nil {
		return "0"
	}
	if num != 0 {
		return str
	} else {
		return "0"
	}
}
func Json_encode(data interface{}) string {
	jsonData, er := json.Marshal(data)
	if er != nil {
		return ""
	}
	jsonData = bytes.Replace(jsonData, []byte("\\u0026"), []byte("&"), -1)
	jsonData = bytes.Replace(jsonData, []byte("\\u003c"), []byte("<"), -1)
	jsonData = bytes.Replace(jsonData, []byte("\\u003e"), []byte(">"), -1)
	jsonData = bytes.Replace(jsonData, []byte(`\<`), []byte(`<`), -1)
	jsonData = bytes.Replace(jsonData, []byte(`\>`), []byte(`>`), -1)
	return string(jsonData)
}
func StrToValidUtf8(str string) string {
	newStr := ""
	for _, s := range str {
		if len(string(s)) < 4 {
			newStr = newStr + string(s)
		}
	}
	return newStr
}
func GbkToUtf8(s []byte) (string, error) {
	reader := transform.NewReader(bytes.NewReader(s), simplifiedchinese.GBK.NewDecoder())
	d, e := ioutil.ReadAll(reader)
	if e != nil {
		return "", e
	}
	return string(d), nil
}
func Big5ToUtf8(s []byte) (string, error) {
	//_, d, _ := mahonia.NewDecoder("utf-8").Translate([]byte(mahonia.NewDecoder("big5").ConvertString(string(s))), true)
	reader := transform.NewReader(bytes.NewReader(s), traditionalchinese.Big5.NewDecoder())
	d, e := ioutil.ReadAll(reader)
	if e != nil {
		return "", e
	}
	return string(d), nil
}

func Utf8ToGbk(s []byte) ([]byte, error) {
	reader := transform.NewReader(bytes.NewReader(s), simplifiedchinese.GBK.NewEncoder())
	d, e := ioutil.ReadAll(reader)
	if e != nil {
		return nil, e
	}
	return d, nil
}
func DirToWwwUser(path string, userName string, basePath string) {
	pathUser, err := user.Lookup(userName)
	if err != nil {
		return
	}
	uid, _ := strconv.Atoi(pathUser.Uid)
	gid, _ := strconv.Atoi(pathUser.Gid)
	path = strings.ReplaceAll(path, "\\", "/")
	path = strings.Replace(path, basePath, "", 1)
	paths := strings.Split(path, "/")
	for _, path := range paths {
		if path != "" {
			os.Chown(basePath+"/"+path, uid, gid)
			basePath = basePath + "/" + path
		}
	}
}
func StrToTime(timeStr string) string {
	if timeStr == "" {
		return "0"
	}
	if timeStr == "<nil>" {
		return "0"
	}
	timeStr = strings.ReplaceAll(timeStr, "/", "-")
	timeStr = strings.ReplaceAll(timeStr, "T", " ")
	timeStr = strings.ReplaceAll(timeStr, "+08:00", "")
	timeStr = strings.TrimSpace(timeStr)
	timeStr = strings.ReplaceAll(timeStr, "  ", " ")
	if strings.Contains(timeStr, "年") {
		timeStr = strings.ReplaceAll(timeStr, "年", "-")
		timeStr = strings.ReplaceAll(timeStr, "月", "-")
		timeStr = strings.ReplaceAll(timeStr, "日", "")
	}
	dateArr := strings.Split(timeStr, " ")
	if len(dateArr) == 1 {
		timeStr = strings.TrimSpace(timeStr) + " 00:00:00"
	} else {
		timeArr := strings.Split(dateArr[1], ":")
		if len(timeArr) == 2 {
			timeStr = timeStr + ":00"
		} else if len(timeArr) == 1 {
			timeStr = timeStr + ":00:00"
		}
	}
	stamp, _ := time.ParseInLocation("2006-01-02 15:04:05", timeStr, time.Local) //使用parseInLocation将字符串格式化返回本地时区时间
	return strconv.FormatInt(stamp.Unix(), 10)
}

// TimeToStr 将时间戳转换成日期格式字符串format=2006-01-02 15:04:05
func TimeToStr(timeStamp int64, format string) string {
	if timeStamp == 0 {
		timeStamp = time.Now().Unix()
	}
	if format == "" {
		format = "2006-01-02 15:04:05"
	}
	return time.Unix(timeStamp, 0).Format(format)
}
func AddMonthPreserveEndOfMonth(t time.Time, months int) time.Time {
	// 跳转到目标月份的第一天
	firstOfMonth := time.Date(t.Year(), t.Month()+time.Month(months), 1, 23, 59, 59, 0, t.Location())
	// 获取目标月份的最后一天
	lastOfMonth := firstOfMonth.AddDate(0, 1, -1)
	// 如果原始日期天数大于目标月份天数，使用最后一天
	if t.Day() > lastOfMonth.Day() {
		return lastOfMonth
	}
	return firstOfMonth.AddDate(0, 0, t.Day()-1)
}

// GetDaysAgoSpecificTime 获取几天前几点几分几秒的时间戳
// days: 表示需要回溯的天数，正数表示过去，负数表示未来
// hour: 目标时间的小时数（0-23）
// minute: 目标时间的分钟数（0-59）
// second: 目标时间的秒数（0-59）
// 返回值: 计算得到的时间对象，时区与当前系统一致
func GetDaysAgoSpecificTime(day int, hour int, minute int, second int) int64 {
	// 获取当前时间
	now := time.Now()

	// 减去10天
	tenDaysAgo := now.AddDate(0, 0, 0-day)

	// 调整到当天的0点0分0秒
	return time.Date(
		tenDaysAgo.Year(),
		tenDaysAgo.Month(),
		tenDaysAgo.Day(),
		hour, minute, second, 0,
		tenDaysAgo.Location(), // 使用与原时间相同的时区
	).Unix()
}
