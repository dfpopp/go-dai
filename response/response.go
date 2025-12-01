package response

// Response 统一响应格式
type Response struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

// Success 成功响应
func Success(data interface{}) *Response {
	return &Response{
		Code: 200,
		Msg:  "success",
		Data: data,
	}
}

// Error 失败响应
func Error(code int, msg string) *Response {
	return &Response{
		Code: code,
		Msg:  msg,
		Data: nil,
	}
}
