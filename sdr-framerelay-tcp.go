package main

import (
	"bufio"
	"flag"
	"fmt"
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

		// establish connection to client
		conDst, err := net.DialTCP("tcp", nil, addrDst)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("connected to", *connectPtr)

		srcReadWrite := bufio.NewReadWriter(bufio.NewReader(conSrc), bufio.NewWriter(conSrc))
		dstReadWrite := bufio.NewReadWriter(bufio.NewReader(conDst), bufio.NewWriter(conDst))

		// running the routine to handle
		go handle_data_stream(*srcReadWrite, *dstReadWrite)
		go handle_cmd_stream(*srcReadWrite, *dstReadWrite)
	}
}

func handle_cmd_stream(srcReadWrite bufio.ReadWriter, dstReadWrite bufio.ReadWriter) {
	// Handling cmd channel - from src/listening -> dst/connect
	// buffer for source (UX)
	srcBuf := make([]byte, 2*1024)

	for {
		// Read data from src
		n, err := srcReadWrite.Read(srcBuf)
		if err != nil {
			log.Fatal(err)
		}
		if n > 0 {
			// Write data to a Dst
			_, err := dstReadWrite.Write([]byte(srcBuf))
			if err != nil {
				log.Fatal(err)
			}
		}
		dstReadWrite.Flush()
	}
}

func handle_data_stream(srcReadWrite bufio.ReadWriter, dstReadWrite bufio.ReadWriter) {
	// Handling command channel - from dst/connect to src/listening
	// buffer for data (sdr)
	dstBuf := make([]byte, 128*1024)
	for {
		// Read data from dst
		n, err := dstReadWrite.Read(dstBuf)
		if err != nil {
			log.Fatal(err)
		}
		if n > 0 {
			// Write data to src
			_, err := srcReadWrite.Write([]byte(dstBuf))
			if err != nil {
				log.Fatal(err)
			}
		}
		srcReadWrite.Flush()
	}
}
