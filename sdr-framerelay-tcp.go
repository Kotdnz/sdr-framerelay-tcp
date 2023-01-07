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

		// establish connection
		conDst, err := net.DialTCP("tcp", nil, addrDst)
		if err != nil {
			log.Fatal(err)
		}

		// buffer for source (UX)
		srcBuf := make([]byte, 2*1024)
		srcReadWrite := bufio.NewReadWriter(bufio.NewReader(conSrc), bufio.NewWriter(conSrc))

		// buffer for data (sdr)
		dstBuf := make([]byte, 128*1024)
		dstReadWrite := bufio.NewReadWriter(bufio.NewReader(conDst), bufio.NewWriter(conDst))

		fmt.Println("connected to", *connectPtr)
		// from dst -> src
		go handle_data_stream(conSrc, conDst, *srcReadWrite, *dstReadWrite, srcBuf, dstBuf)
		go handle_cmd_stream(conSrc, conDst, *srcReadWrite, *dstReadWrite, srcBuf, dstBuf)
	}
}

func handle_cmd_stream(conSrc io.ReadWriter, conDst io.ReadWriter, srcReadWrite bufio.ReadWriter, dstReadWrite bufio.ReadWriter, srcBuf []byte, dstBuf []byte) {
	// Handling cmd channel - from src/listening -> dst/connect

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

func handle_data_stream(conSrc io.ReadWriter, conDst io.ReadWriter, srcReadWrite bufio.ReadWriter, dstReadWrite bufio.ReadWriter, srcBuf []byte, dstBuf []byte) {
	// Handling command channel - from dst/connect to src/listening
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
