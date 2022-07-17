package main

import (
	"encoding/json"
	"grpc/codec"
	"log"
	"net"
	"time"
)

func startServer(addr chan string){
	//pick a free port
	l,err := net.Listen("tcp",":0")
	if err != nil {
		log.Fatal("network error:",err)
	}
	log.Println("start rpc server on",l.Addr())
	addr <- l.Addr().String()

}



func main(){
	addr := make(chan string)
	go startServer(addr)
	conn,_:=net.Dial("tcp",<-addr)
	defer func(){
		_ =conn.Close()
	}()
	time.Sleep(1*time.Second)
	_ = json.NewEncoder(conn).Encode(codec.DefaultOption)
	c :=codec.NewCodecFunc(conn)
	for i:=0;i<5;i++{
		h:=&codec.Header{
			
		}
	}




}
