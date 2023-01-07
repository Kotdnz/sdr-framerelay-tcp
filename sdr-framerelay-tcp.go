package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
)

func main() {
	// read CLI
	listenPtr := flag.String("listen", "0.0.0.0:9001", "listen IP:Port by default is [0.0.0.0:9001]")
	connectPtr := flag.String("connect", "127.0.0.1:9002", "connect IP:Port by default is [127.0.0.1:9002]")
	compressPtr := flag.String("compress", "no", "what end of transport will be compressed. Default is [no], possible options listen, connect")
	//TODO - compress level
	flag.Parse()

	fmt.Println("Compressed is: ", *compressPtr)
	if string(*compressPtr) != "no" {
		if string(*compressPtr) == "connect" {
			fmt.Println("Compressing sending data")
		} else {
			fmt.Println("Decompressing receiving data")
		}
	}
	// convert address
	addrSrc, _ := net.ResolveTCPAddr("tcp", *listenPtr)
	addrDst, _ := net.ResolveTCPAddr("tcp", *connectPtr)

	// listener
	listener, err := net.ListenTCP("tcp", addrSrc)
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()
	fmt.Println("listening ", *listenPtr)

	for {
		conSrc, err := listener.AcceptTCP()
		if err != nil {
			log.Fatal(err)
		}

		// start handling the request, blocking mode
		go func(conSrc io.ReadWriter) {
			// Create the buffer for source
			srcReadWrite := bufio.NewReadWriter(bufio.NewReader(conSrc), bufio.NewWriter(conSrc))
			srcBuf := make([]byte, 8*1024*1024)

			// establish connection
			conDst, err := net.DialTCP("tcp", nil, addrDst)
			if err != nil {
				log.Fatal(err)
			}
			defer conDst.Close()
			fmt.Println("connected to", *connectPtr)

			// Create the buffer for dest
			dstReadWrite := bufio.NewReadWriter(bufio.NewReader(conDst), bufio.NewWriter(conDst))
			dstBuf := make([]byte, 2*1024)

			go func() {
				// Handling command channel - from dst/connect to src/listening
				for {
					// Read data from dst
					n2, err := dstReadWrite.Read(srcBuf)
					if err != nil {
						log.Fatal(err)
					}
					if n2 > 0 {
						// Write data to src
						_, err := srcReadWrite.Write([]byte(srcBuf))
						if err != nil {
							log.Fatal(err)
						}
					}
					dstReadWrite.Flush()
				}
			}()
			// Handling data channel - from src/listening to dst/connect
			for {
				// Read data from src
				n1, err := srcReadWrite.Read(dstBuf)
				if err != nil {
					log.Fatal(err)
				}
				if n1 > 0 {
					// Write data to a Dst
					_, err := dstReadWrite.Write([]byte(dstBuf))
					if err != nil {
						log.Fatal(err)
					}
				}
				srcReadWrite.Flush()
			}
		}(conSrc)
	}
}
