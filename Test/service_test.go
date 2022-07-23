package test

import (
	"fmt"
	"grpc/server"
	"testing"
)

type foo struct{}
type Args struct {
	Num1, Num2 int
}

func (f foo) Sum(args Args, reply *int) error {
	*reply = args.Num1 + args.Num2
	return nil
}

func (f foo) sum(args Args, reply *int) error {
	*reply = args.Num1 + args.Num2
	return nil
}
func _assert(conditon bool, msg string, v ...interface{}) {
	if !conditon {
		panic(fmt.Sprintf("assertion faild:"+msg, v...))
	}
}

func TestNewService(t *testing.T) {
	var foo foo
	s := server.NewService(&foo)
	_assert(len(s.Method) == 1, "wrong service Method, expect 1, but got %d", len(s.Method))
	myType := s.Method["Sum"]
	_assert(myType != nil, "wrong Method,Sum shouldn't nil")
}
