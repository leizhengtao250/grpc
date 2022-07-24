package server

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"grpc/codec"
	"io"
	"log"
	"net"
	"reflect"
	"sync"
	"sync/atomic"
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
	CodecType   codec.Type
	/**
	添加超时处理
	**/
	ConnectTimeout time.Duration
	HandleTimeout  time.Duration
}

var DefaultOption = &Option{
	MagicNumber:    MagicNumber,
	CodecType:      codec.GobType,
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

/**
 客户端的请求过来
服务端动态的读取客户端的请求
1.通过反射读取请求中的对象和类型
**/

type methodType struct {
	method    reflect.Method
	ArgType   reflect.Type
	ReplyType reflect.Type
	numCalls  uint64
}

func (m *methodType) NumCalls() uint64 {
	return atomic.LoadUint64(&m.numCalls)
}

func (m *methodType) NewArgv() reflect.Value {
	var argv reflect.Value
	if m.ArgType.Kind() == reflect.Ptr {
		argv = reflect.New(m.ArgType.Elem())
	} else {
		argv = reflect.New(m.ArgType).Elem()
	}
	return argv
}
func (m *methodType) NewReplyv() reflect.Value {
	// reply must be a pointer type
	replyv := reflect.New(m.ReplyType.Elem())
	switch m.ReplyType.Elem().Kind() {
	case reflect.Map:
		replyv.Elem().Set(reflect.MakeMap(m.ReplyType.Elem()))
	case reflect.Slice:
		replyv.Elem().Set(reflect.MakeSlice(m.ReplyType.Elem(), 0, 0))
	}
	return replyv
}

type service struct {
	name   string
	typ    reflect.Type
	rcvr   reflect.Value
	Method map[string]*methodType
}

func NewService(rcvr interface{}) *service {
	s := new(service)
	s.rcvr = reflect.ValueOf(rcvr)
	s.name = reflect.Indirect(s.rcvr).Type().Name()
	s.typ = reflect.TypeOf(rcvr)
	if !ast.IsExported(s.name) {
		log.Fatal("rpc server:%s is not a vaild service name", s.name)
	}
	s.registerMethods()
	return s
}

func (s *service) registerMethods() {
	s.Method = make(map[string]*methodType)
	for i := 0; i < s.typ.NumMethod(); i++ {
		method := s.typ.Method(i)
		mType := method.Type
		if mType.NumIn() != 3 || mType.NumOut() != 1 {
			continue
		}
		if mType.Out(0) != reflect.TypeOf((*error)(nil)).Elem() {
			continue
		}
		argType, replyType := mType.In(1), mType.In(2)
		if !isExportedOrBuiltinType(argType) || !isExportedOrBuiltinType(replyType) {

		}

	}

}

func isExportedOrBuiltinType(t reflect.Type) bool {
	return ast.IsExported(t.Name()) || t.PkgPath() == ""
}
func (s *service) Call(m *methodType, argv, replyv reflect.Value) error {
	atomic.AddUint64(&m.numCalls, 1)
	f := m.method.Func
	returnValues := f.Call([]reflect.Value{s.rcvr, argv, replyv})
	if errInter := returnValues[0].Interface(); errInter != nil {
		return errInter.(error)
	}
	return nil
}

func (server *codec.Server) handleRequest(cc codec.Codec, req *server.requeset, sending *sync.WaitGroup, timeout time.Duration) {

}
