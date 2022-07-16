package codec

import (
	"encoding/json"
	"io"
	"log"
	"net"
)

const MagicNumber = 0x3bef5c

/**
	客户端传送内容到服务端
	@Option内容用json传送
	@服务端对Option json解码
	@然后得到CodecType ,再用CodecType解码剩余内容
**/

/**
| Option{MagicNumber: xxx, CodecType: xxx} | Header{ServiceMethod ...} | Body interface{} |
| <------      固定 JSON 编码      ------>  | <-------   编码方式由 CodeType 决定   ------->|
**/

type Option struct {
	MagicNumber int
	CodecType   Type
}

var DefaultOption = &Option{
	MagicNumber: MagicNumber,
	CodecType:   GobType,
}

//Server represent an RPC Server
type Server struct{}

//NewServer return a new Server
func NewServer() *Server {
	return &Server{}
}

//Accept accept connections on the listener and serve requests
//for each incomming connection
func (s *Server) Accept(lis net.Listener) {
	for {
		conn, err := lis.Accept()
		if err != nil {
			log.Println("rpc server :accept error", err)
			return
		}
		go s.ServeConn(conn)
	}
}

func (s *Server) ServeConn(conn io.ReadWriteCloser) {
	defer func() {
		_ = conn.Close()
	}()
	var opt Option
	if err := json.NewDecoder(conn).Decode(&opt); err != nil {
		log.Println("rpc server :option err:", err)
		return
	}
	if opt.MagicNumber != MagicNumber {
		log.Printf("rpc server:invaild magic number %x", opt.MagicNumber)
		return
	}
	f := NewCodecFuncMap[opt.CodecType] //get the process function
	if f == nil {
		log.Printf("rpc server:invaild codec type %s", opt.CodecType)
		return
	}
	s.serveCodec(f(conn))
}

func (s *Server) serveCodec(c Codec) {

}
