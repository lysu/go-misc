package main

import (
	"fmt"
	"github.com/samuel/go-zookeeper/zk"
	"time"
)

func main() {

	conn, event, err := zk.Connect([]string{"10.10.10.64:2181"}, 1*time.Second)
	if err != nil {
		panic(err)
	}
	go func() {
		for {
			e := <-event
			fmt.Println("e1:", e)
		}
	}()
	e, _, err := conn.Exists("/t1")
	if err != nil {
		panic(err)
	}
	if !e {
		s, err := conn.Create("/t1", nil, 0, zk.WorldACL(zk.PermAll))
		if err != nil {
			panic(err)
		}
		fmt.Println(s)
	}

	cs, _, ev, err := conn.ChildrenW("/t1")
	if err != nil {
		panic(err)
	}
	fmt.Println(cs)

	go func() {
		for {
			e := <-ev
			fmt.Println("Event:", e)
		}
	}()

	e2, _, err := conn.Exists("/t1/1")
	if err != nil {
		panic(err)
	}
	if !e2 {
		s, err := conn.Create("/t1/1", nil, int32(zk.FlagEphemeral), zk.WorldACL(zk.PermAll))
		if err != nil {
			panic(err)
		}
		fmt.Println("create", s)
	}

	select {}

}
