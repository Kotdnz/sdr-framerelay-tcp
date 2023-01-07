package main

import (
        "bufio"
        "fmt"
        "net"
	"io"
	"log"
)

func main() {
	// convert address
	addrSrc, _ := net.ResolveTCPAddr("tcp", "0.0.0.0:9001")
	addrDst, _ := net.ResolveTCPAddr("tcp", "0.0.0.0:9002")
	
	// listener
	listener, err := net.ListenTCP("tcp", addrSrc)
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()
	fmt.Println("listening :9001")

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
                        fmt.Println("connected to :9002")

			// Create the buffer for dest
			dstReadWrite := bufio.NewReadWriter(bufio.NewReader(conDst), bufio.NewWriter(conDst))
			dstBuf := make([]byte, 8*1024*1024)

			for {
				// Read data from src
				n1, err := srcReadWrite.Read(srcBuf)
				if err != nil {
                              	   log.Fatal(err)
                    		}
				if n1 > 0 {
			  		// Write data to a Dst
			  		_, err := dstReadWrite.Write([]byte(srcBuf))
					if err != nil {
					  log.Fatal(err)
					}
					//srcBuf.Flush()
				}
				// Writing vice versa
				// Read data from dst
				n2, err := dstReadWrite.Read(dstBuf)
				if err != nil {
					log.Fatal(err)
				}
				if n2 > 0 {
					// Write data to src
					_, err := srcReadWrite.Write([]byte(dstBuf))
					if err != nil {
						log.Fatal(err)
					}
					//dstBuf.Flush()
				}
			}
			//conSrc.Close()
		} (conSrc)
	}
}


