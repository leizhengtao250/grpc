package codec

import "io"

/**
客户端在注册中心调用服务
就会向注册中心发起请求
@Seq --msgID -- 用于调用router
@ServiceMethod --服务名和方法名

**/
type Header struct {
	ServiceMethod string
	Seq           uint64
	Error         string
}

/**
@Read  为解码
@Write 为编码
**/
type Codec interface {
	io.Closer
	ReadHeader(*Header) error
	ReadBody(interface{}) error
	Write(*Header, interface{}) error
}

type NewCodecFunc func(io.ReadWriteCloser) Codec

type Type string

const (
	GobType  Type = "application/gob"
	JsonType Type = "application/json"
)

/**
根据不同的id 用不同的函数解析
**/
var NewCodecFuncMap map[Type]NewCodecFunc

func init() {
	NewCodecFuncMap = make(map[Type]NewCodecFunc)
	NewCodecFuncMap[GobType] = NewGobCodec
}
