package pool

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

var poolMap = make(map[string]*Pool)
var poolMapLock sync.RWMutex

func init() {
	go func() {
		tc := time.NewTicker(time.Second * 10)
		defer tc.Stop()
		//tc := time.NewTimer(time.Second * 10)
		for range tc.C {
			poolMapLock.RLock()
			l := len(poolMap)
			var pcl int
			for _, p := range poolMap {
				//fmt.Println("pl", p.Len())
				pcl += p.Len()
			}
			poolMapLock.RUnlock()
			fmt.Println("pool num:", l)
			fmt.Println("total conn num:", pcl)
			//tc.Reset(time.Second * 10)
		}
	}()
}

func GetConnPool(addr, network string, timeout time.Duration) (*Pool, error) {
	poolMapLock.RLock()
	p, ok := poolMap[addr]
	if ok {
		poolMapLock.RUnlock()
		return p, nil
	}
	poolMapLock.RUnlock()

	poolMapLock.Lock()
	defer poolMapLock.Unlock()
	if p, ok := poolMap[addr]; ok {
		return p, nil
	}
	pool, err := NewPool(2, 10, func() (interface{}, error) {
		conn, err := net.DialTimeout(network, addr, timeout)
		if err != nil {
			return nil, err
		}
		return conn, nil
	})
	if err != nil {
		return nil, err
	}

	pool.Idle = 3 * time.Minute
	pool.Ping = func(x interface{}) bool {
		if x == nil {
			return false
		}
		if _, ok := x.(net.Conn); !ok {
			return false
		}
		return true
	}
	pool.Close = func(x interface{}) {
		if conn, ok := x.(net.Conn); ok {
			conn.Close()
		}
	}
	pool.RegisterCheck(3*time.Minute, func(x interface{}) bool {
		if x == nil {
			return false
		}
		conn, ok := x.(net.Conn)
		if !ok {
			return false
		}
		if _, err := conn.Read(make([]byte, 1)); err == io.EOF {
			return false
		}
		return true
	})

	poolMap[addr] = pool
	return pool, nil
}
