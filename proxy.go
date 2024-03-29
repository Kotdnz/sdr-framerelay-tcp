// Modified source from https://github.com/BlueDragonX/go-proxy-example

package main

import (
	"log"
	"net"
	"sync"
)

// Proxy connections from Listen to Backend.
type Proxy struct {
	Listen      string
	Backend     string
	listener    net.Listener
	compressDir string
	compressLvl string
	compressAlg string
	concurrency int
}

func (p *Proxy) Run() error {
	var err error

	if p.listener, err = net.Listen("tcp", p.Listen); err != nil {
		return err
	}

	wg := &sync.WaitGroup{}
	for {
		if conn, err := p.listener.Accept(); err == nil {
			wg.Add(1)
			go func() {
				defer wg.Done()
				p.handle(conn)
			}()
		} else {
			break
		}
	}
	wg.Wait()
	return nil
}

func (p *Proxy) Close() error {
	return p.listener.Close()
}

func (p *Proxy) handle(upConn net.Conn) {
	defer upConn.Close()
	log.Printf("accepted: %s", upConn.RemoteAddr())
	downConn, err := net.Dial("tcp", p.Backend)
	if err != nil {
		log.Printf("unable to connect to %s: %s\n", p.Backend, err)
		return
	}
	defer downConn.Close()
	if err := Pipe(upConn, downConn, p.compressDir, p.compressLvl, p.compressAlg, p.concurrency); err != nil {
		log.Printf("pipe failed: %s\n", err)
	} else {
		log.Printf("disconnected: %s\n", upConn.RemoteAddr())
	}
}
