package main

import (
	"fmt"
	"sync"
)

/**
  Client
**/
func main() {
	var m sync.Map
	m.LoadOrStore(1, 1)
	fmt.Println(m)
}
