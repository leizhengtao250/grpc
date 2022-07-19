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

func (u *User) Eat() {
	fmt.Println(u.name + "eat")
}

func (u *User) Do() {
	fmt.Printf("%d,%s is doing", u.age, u.name)
}

func TestC(t *testing.T) {
	u := &User{
		name: "dalei",
		age:  13,
	}

	typ := reflect.TypeOf(u)
	fmt.Println(typ.NumMethod())
	for i := 0; i < typ.NumMethod(); i++ {
		method := typ.Method(i)
		fmt.Println(method.Type)
		fmt.Println(method.Type.NumIn())
		fmt.Println(method.Type.NumOut())
		fmt.Println("-------------------------")
	}

}
