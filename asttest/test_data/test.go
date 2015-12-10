package main

import (
	"fmt"
	"strconv"
)

// SimpleInterface is ...
type SimpleInterface interface {
	Tojson() string
}

// Impl is implement ....
type Impl struct {
	Val int
}

// Tojson is simpleInterface method
func (a *Impl) Tojson() string {
	return "ght" + strconv.Itoa(a.Val)
}

func a(i SimpleInterface) {
	if i == nil {
		fmt.Printf("i==nil")
	} else {
		fmt.Printf("i!=nil")
		//fmt.Println(i.Tojson())
	}
}

func main() {
	var i *Impl
	a(i)
}
