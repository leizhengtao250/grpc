package client

import (
	"errors"
	"fmt"
	"grpc/codec"
	"log"
	"net"
	"sync"
)

/**
支持并非和异步的客户端
一个rpc需要满足如下几个条件
1.方法类型是可视的
2.方法是暴露的
3.方法有两个参数，第二个为指针
4.方法有返回的错误类型

*/

/**
@Seq        	用于发送的请求编号，每个请求拥有唯一编号
@ServiceMethod  调用的方法
@Args			调用的方法参数
@Reply			方法的返回值
@Error          错误的处理
@Done           为了支持异步调用，如果调用结束，会使用call.Done()通知对方
**/
type Call struct {
	Seq           uint64
	ServiceMethod string      //format "<service>.<method>"
	Args          interface{} //arguments to the function
	Reply         interface{} //reply from the function
	Error         error       //if error occurs,it will be set
	Done          chan *Call  //Strobes when call is complete
}

func (c *Call) done() {
	c.Done <- c
}

/**
client represents an rpc client
There may be multiple outstanding Calls associated
with a single Client and a Client may be used by multiple goroutines simultanously
**/
type Client struct {
	cc        codec.Codec
	opt       *codec.Option
	sending   sync.Mutex //protect following
	header    codec.Header
	mu        sync.Mutex //protect following
	seq       uint64
	pending   map[uint64]*Call
	closing   bool //user has called close
	shuntdown bool //server has told us to stop
}

// var _ io.Closer = (*Client)(nil)
var ErrShuttdown = errors.New("connection is shut down")

//close the connection

func (client *Client) Close() error {
	client.mu.Lock()
	defer client.mu.Unlock()
	if client.closing {
		return ErrShuttdown
	}
	client.closing = true
	return client.cc.Close()
}

//IsAvailable return true if the client does work
func (client *Client) IsAvailable() bool {
	client.mu.Lock()
	defer client.mu.Unlock()
	return !client.closing && !client.shuntdown
}

/**
一个call注册到client中
client给call赋予一个编号，每个编号都是唯一的
*/

//将函数注册到客户端
func (client *Client) registerCall(call *Call) (uint64, error) {
	client.mu.Lock() //每个编号都是唯一的，所以锁住赋值
	defer client.mu.Unlock()
	if client.closing || client.shuntdown {
		return 0, ErrShuttdown
	}
	call.Seq = client.seq
	client.pending[call.Seq] = call
	client.seq++
	return call.Seq, nil
}

//从client中移除call
func (client *Client) removeCall(seq uint64) *Call {
	client.mu.Lock()
	defer client.mu.Unlock()
	call := client.pending[seq]
	delete(client.pending, seq)
	return call
}

func (client *Client) terminateCalls(err error) {
	client.sending.Lock()
	defer client.sending.Unlock()
	client.mu.Lock()
	defer client.mu.Unlock()
	client.shuntdown = true
	for _, call := range client.pending {
		call.Error = err
		call.done()
	}
}

//客户端接收信息
/**
	接收的消息分为3种情况
	1。服务端正常发来消息，客户端负责解析
	2.call存在，但是服务端出错，即h.Error 不为空
	3.call不存在，
**/

func (client *Client) recieve() {
	var err error
	for err == nil {
		var h codec.Header
		if err = client.cc.ReadHeader(&h); err != nil {
			break
		}
		call := client.removeCall(h.Seq)
		switch {
		case call == nil: //call 不存在
			err = client.cc.ReadBody(nil)
		case h.Error != "": //call 存在 server发来消息出错
			call.Error = fmt.Errorf(h.Error)
			err = client.cc.ReadBody(nil)
			call.done() //本次调用结束
		default:
			err = client.cc.ReadBody(call.Reply)
			if err != nil {
				call.Error = errors.New("reading body error:" + err.Error())
			}
			call.done() //本次调用结束
		}

	}
	client.terminateCalls(err)
}

/**
	创建client实例的时候，
	1.和server协商好编解码的方式
	2.开启子协程receive()接收响应
**/
func NewClient(conn net.Conn, opt *codec.Option) (*Client, error) {
	f := codec.NewCodecFuncMap[opt.CodecType]
	if f == nil {
		err := fmt.Errorf("invaild codec type %s", opt.CodecType)
		log.Println("rpc client:codec error", err)
		return nil, err
	}
	return newClientCodec(f(conn), opt), nil
}

func newClientCodec(cc codec.Codec, opt *codec.Option) *Client {
	client := &Client{
		seq:     1,
		cc:      cc,
		opt:     opt,
		pending: make(map[uint64]*Call),
	}
	go client.recieve()
	return client
}

func parseOption(opts ...*codec.Option) (*codec.Option, error) {
	if len(opts) == 0 || opts[0] == nil {
		return codec.DefaultOption, nil
	}
	if len(opts) != 1 {
		return nil, errors.New("number of options is more than 1")
	}
	opt := opts[0]
	opt.MagicNumber = codec.DefaultOption.MagicNumber
	if opt.CodecType == "" {
		opt.CodecType = codec.DefaultOption.CodecType
	}
	return opt, nil
}

func Dial(network, address string, opts ...*codec.Option) (client *Client, err error) {
	opt, err := parseOption(opts...)
	if err != nil {
		return nil, err
	}
	conn, err := net.Dial(network, address)
	if err != nil {
		return nil, err
	}
	//close the connection if client is nil
	defer func() {
		if client == nil {
			_ = conn.Close()
		}
	}()
	return NewClient(conn, opt)
}

func (client *Client) send(call *Call) {
	client.sending.Lock()
	defer client.sending.Unlock()

	//register the call
	seq, err := client.registerCall(call)
	if err != nil {
		call.Error = err
		call.done()
		return
	}
	//prepare request head
	client.header.ServiceMethod = call.ServiceMethod
	client.header.Seq = seq
	client.header.Error = ""

	//encode and send the request
	if err := client.cc.Write(&client.header, call.Args); err != nil {
		call := client.removeCall(seq)
		//call may be nil.it usually means that write partially failed
		//client has received the response and handled
		if call != nil {
			call.Error = err
			call.done()
			return
		}
	}
}

//Go invokes the function asynchronoously
//it returns the call structure representin the invocation

func (client *Client) Go(ServiceMethod string, args, reply interface{}, done chan *Call) *Call {
	if done == nil {
		done = make(chan *Call, 10)
	} else if cap(done) == 0 {
		log.Panic("rpc client:done channel is unbuffered")
	}
	call := &Call{
		ServiceMethod: ServiceMethod,
		Args:          args,
		Reply:         reply,
		Done:          done,
	}
	client.send(call)
	return call

}

func (client *Client) Call(ServiceMethod string, args, reply interface{}) error {
	call := <-client.Go(ServiceMethod, args, reply, make(chan *Call, 1)).Done
	return call.Error
}
