package test

import (
	"fmt"
	"reflect"
	"testing"
)

type User struct {
	name string
	age  int
}

func (u *User) eat() {
	fmt.Println(u.name + "eat")
}

func (u *User) do() {
	fmt.Printf("%d,%s is doing", u.age, u.name)
}

func TestC(t *testing.T) {
	var u User
	typ := reflect.TypeOf(u)
	fmt.Println(typ.Method(0))
	fmt.Println(typ.Method(0).Type)
	fmt.Println(typ)
}
