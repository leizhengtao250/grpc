package codec

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"reflect"
	"sync"
	"time"
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
	/**
	添加超时处理
	**/
	ConnectTimeout time.Duration
	HandleTimeout  time.Duration
}

var DefaultOption = &Option{
	MagicNumber:    MagicNumber,
	CodecType:      GobType,
	ConnectTimeout: 10 * time.Second,
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

var invalidRequest = struct{}{}

func (s *Server) serveCodec(c Codec) {
	sending := new(sync.Mutex) //make sure to send a complete response
	wg := new(sync.WaitGroup)
	for {
		req, err := s.readRequest(c)
		if err != nil {
			if req == nil {
				break
			}
			req.h.Error = err.Error()
			s.sendResponse(c, req.h, invalidRequest, sending)
			continue
		}
		wg.Add(1)
		go s.handleRequest(c, req, sending, wg)
	}
	wg.Wait()
	_ = c.Close()
}

type request struct {
	h            *Header       //header of  request
	argv, replyv reflect.Value // argv and replyv of request
}

/**
@read request header from client
**/
func (s *Server) readRequestHeader(c Codec) (*Header, error) {
	var h Header
	if err := c.ReadHeader(&h); err != nil {
		if err != io.EOF && err != io.ErrUnexpectedEOF {
			log.Println("rpc server : read header error:", err)
		}
		return nil, err
	}
	return &h, nil
}

/**
@read request base header
**/
func (s *Server) readRequest(c Codec) (*request, error) {
	h, err := s.readRequestHeader(c)
	if err != nil {
		return nil, err
	}
	req := &request{
		h: h,
	}
	//TODO :

	req.argv = reflect.New(reflect.TypeOf(""))
	if err = c.ReadBody(req.argv.Interface()); err != nil {
		log.Println("rpc server:read argv err:", err)
	}
	return req, nil
}

/**
@send msg to client
*/

func (s *Server) sendResponse(c Codec, h *Header, body interface{}, sending *sync.Mutex) {
	sending.Lock()
	defer sending.Unlock()
	if err := c.Write(h, body); err != nil {
		log.Println("rpc server : write response error:", err)
	}
}

/**
process the request
*/
func (s *Server) handleRequest(c Codec, req *request, sending *sync.Mutex, wg *sync.WaitGroup) {
	defer wg.Done()
	log.Println(req.h, req.argv.Elem())
	req.replyv = reflect.ValueOf(fmt.Sprintf("grpc resp %d", req.h.Seq))
	s.sendResponse(c, req.h, req.replyv.Interface(), sending)
}
