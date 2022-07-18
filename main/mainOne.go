package main

import (
	"encoding/json"
	"fmt"
	"grpc/codec"
	"log"
	"net"
	"time"
)

func startServer(addr chan string) {
	//pick a free port
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatal("network error:", err)
	}
	log.Println("start rpc server on", l.Addr())
	addr <- l.Addr().String()
	codec.NewServer().Accept(l)

}

/**
  Client
**/
func main() {
	addr := make(chan string)
	go startServer(addr) //start a server
	conn, _ := net.Dial("tcp", <-addr)
	defer func() {
		_ = conn.Close()
	}()
	time.Sleep(1 * time.Second)
	_ = json.NewEncoder(conn).Encode(codec.DefaultOption)
	c := codec.NewGobCodec(conn)
	for i := 0; i < 5; i++ {
		h := &codec.Header{
			ServiceMethod: "Foo.sum",
			Seq:           uint64(i),
		}
		_ = c.Write(h, fmt.Sprintf("grpc req %d", h.Seq))
		_ = c.ReadHeader(h)
		var reply string
		_ = c.ReadBody(&reply)
		log.Println("reply:", reply)
	}
}
