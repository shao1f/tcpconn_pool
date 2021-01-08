package pool

import (
	"fmt"
	"sync"
	"time"
)

type Pool struct {
	New   func() (interface{}, error)
	Close func(x interface{})
	Ping  func(x interface{}) bool
	conns chan *item
	Idle  time.Duration
	Lock  sync.Mutex
}

type item struct {
	data interface{}
	t    time.Time
}

func NewPool(initCap, maxCap int, newFunc func() (interface{}, error)) (*Pool, error) {
	if maxCap <= 0 || initCap > maxCap {
		return nil, fmt.Errorf("err cap")
	}
	p := new(Pool)
	//p.Idle = idel
	if newFunc != nil {
		p.New = newFunc
	}
	p.conns = make(chan *item, maxCap)
	for i := 0; i < initCap; i++ {
		conn, err := p.create()
		if err != nil {
			return p, err
		}
		p.conns <- &item{data: conn, t: time.Now()}
	}
	return p, nil
}

func (p *Pool) Len() int {
	return len(p.conns)
}

func (p *Pool) Get() (interface{}, error) {
	if p.conns == nil {
		return p.create()
	}
	for {
		select {
		case v := <-p.conns:
			conn := v.data
			if p.Idle > 0 && time.Now().Sub(v.t) > p.Idle {
				if p.Close != nil {
					p.Close(conn)
				}
				continue
			}
			if p.Ping != nil && !p.Ping(conn) {
				continue
			}
			return conn, nil
		default:
			return p.create()
		}
	}
}

func (p *Pool) Put(x interface{}) {
	if p.conns == nil {
		if p.Close != nil {
			p.Close(x)
		}
		return
	}
	for {
		select {
		case p.conns <- &item{data: x, t: time.Now()}:
			return
		default:
			if p.Close != nil {
				p.Close(x)
			}
			return
		}
	}
}

func (p *Pool) Destroy() {
	p.Lock.Lock()
	defer p.Lock.Unlock()

	if p.conns == nil {
		return
	}
	close(p.conns)
	for c := range p.conns {
		if p.Close != nil {
			p.Close(c.data)
		}
	}
	p.conns = nil
}

func (p *Pool) RegisterCheck(interval time.Duration, check func(interface{}) bool) {
	if interval != 0 && check != nil {
		go func() {
			for {
				time.Sleep(interval)
				p.Lock.Lock()
				if p.conns == nil {
					p.Lock.Unlock()
					return
				}
				l := p.Len()
				p.Lock.Unlock()
				for i := 0; i < l; i++ {
					select {
					case v := <-p.conns:
						conn := v.data
						if p.Idle > 0 && time.Now().Sub(v.t) > p.Idle {
							if p.Close != nil {
								p.Close(conn)
							}
							continue
						}
						if !check(conn) {
							if p.Close != nil {
								p.Close(conn)
							}
							continue
						} else {
							select {
							case p.conns <- &item{data: conn, t: time.Now()}:
								continue
							default:
								if p.Close != nil {
									p.Close(conn)
								}
							}
						}
					default:
						break
					}
				}
			}
		}()
	}
}

func (p *Pool) create() (interface{}, error) {
	if p.New == nil {
		return nil, fmt.Errorf("pool new func is nil")
	}
	return p.New()
}
