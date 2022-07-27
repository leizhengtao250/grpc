package test

import (
	"fmt"
	"testing"
	"time"
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

	//address := "127.0.0.1:8999"
	//network := "tcp"
	//cli, err := client.Dial(network, address, server.DefaultOption)
	//if err != nil {
	//	return
	//}
	//defer cli.Close()

	var t1 time.Duration = 5 * time.Second
	var t2 time.Time
	t2 = time.Now()
	time.Sleep(4 * time.Second)
	t3 := t2.Add(t1)
	fmt.Println(t3)
	t4 := time.Now()
	fmt.Println(t4)
	fmt.Println(t2.Add(t1).After(time.Now()))
}
