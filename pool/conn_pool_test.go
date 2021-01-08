package pool

import (
	"fmt"
	"net"
	"testing"
	"time"
)

var addr = "127.0.0.1:8080"
var network = "tcp"

func TestGetConnPool(t *testing.T) {
	go server(t)
	time.Sleep(time.Second * 2)

	p, err := GetConnPool(addr, network, 300*time.Millisecond)
	if err != nil {
		t.Error(err)
	}
	c, err := p.Get()
	if err != nil {
		t.Error(err)
	}
	conn, ok := c.(net.Conn)
	if !ok {
		t.Error("not a conn")
	}
	conn.Write([]byte("[gag]"))

	p.Put(conn)

	t.Log(p.Len())
	fmt.Println(123)
	time.Sleep(time.Minute * 2)
}

func server(t *testing.T) {
	ta, err := net.ResolveTCPAddr(network, addr)
	if err != nil {
		t.Fatal(err)
	}
	tl, err := net.ListenTCP(network, ta)
	if err != nil {
		t.Fatal(err)
	}
	for {
		tc, err := tl.AcceptTCP()
		if err != nil {
			t.Log(err)
			continue
		}
		t.Log("get conn,addr=", tc.RemoteAddr())
	}
}
