package test

import (
	"fmt"
	client "grpc/Client"
	"grpc/server"
	"testing"
)

type User struct {
	name string
	age  int
}

func (u *User) Eat() {
	fmt.Println(u.name + "eat")
}

func (u *User) Do() {
	fmt.Printf("%d,%s is doing", u.age, u.name)
}

func Test_client(t *testing.T) {

	address := "127.0.0.1:8999"
	network := "tcp"
	cli, err := client.Dial(network, address, server.DefaultOption)
	if err != nil {
		return
	}
	defer cli.Close()

}
