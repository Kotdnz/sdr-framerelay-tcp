/** @file sdr-framerelay-tcp.go
 *
 * @brief fremarelay between source and destination to optimize the stream
 * and even compress tcp flow from https://github.com/blinick/rtl-sdr/
 * @source https://github.com/Kotdnz/sdr-framerelay-tcp
 * @author Kostiantyn Nikonenko
 * @date January, 10, 2023
 *
 */

package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
)

var Version string = "v.2.0"

func main() {
	fmt.Println("sdr-fremarelay-tcp version: ", Version)

	// read CLI
	flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	listen := flag.String("listen", "0.0.0.0:9001", "listen IP:Port by default is [0.0.0.0:9001]")
	backend := flag.String("connect", "127.0.0.1:9002", "connect IP:Port by default is [127.0.0.1:9002]")
	compressPtr := flag.String("compress", "no", "what end of transport will be compressed. Default is [no], possible options listen, connect")
	compressLevel := flag.Int("level", 9, "The compressing level, default is 9")

	flag.Parse()

	fmt.Println("Compressed is: ", *compressPtr, ", level is ", *compressLevel)
	if string(*compressPtr) != "no" {
		if string(*compressPtr) == "connect" {
			fmt.Println("Compressing sending data")
		} else {
			fmt.Println("Decompressing receiving data")
		}
	}
	// convert address
	//listen, _ := net.ResolveTCPAddr("tcp", *listenPtr)
	//backend, _ := net.ResolveTCPAddr("tcp", *connectPtr)

	p := Proxy{Listen: *listen, Backend: *backend}

	sigs := make(chan os.Signal)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		if err := p.Close(); err != nil {
			log.Fatal(err.Error())
		}
	}()

	if err := p.Run(); err != nil {
		log.Fatal(err.Error())
	}
}

// Copy data between two connections. Return EOF on connection close.
func Pipe(a, b net.Conn) error {
	done := make(chan error, 1)

	cp := func(r, w net.Conn) {
		_, err := io.Copy(r, w)
		//log.Printf("copied %d bytes from %s to %s", n, r.RemoteAddr(), w.RemoteAddr())
		done <- err
	}

	go cp(a, b)
	go cp(b, a)
	err1 := <-done
	err2 := <-done
	if err1 != nil {
		return err1
	}
	if err2 != nil {
		return err2
	}
	return nil
}
