/** @file sdr-framerelay-tcp.go
 *
 * @brief fremarelay between source and destination to optimize the stream
 * and even compress tcp flow from https://github.com/blinick/rtl-sdr/
 * @source https://github.com/Kotdnz/sdr-framerelay-tcp
 * @author Kostiantyn Nikonenko
 * @date January, 09, 2023
 * @time 12:57
 */

package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
)

var Version string = "v.1.7"
var isConnected bool

func main() {
	fmt.Println("sdr-fremarelay-tcp version: ", Version)
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
	fmt.Println("listening ", *listenPtr)
	defer listener.Close()

	var conSrc net.Conn
	var conDst net.Conn
	isConnected = false

	for {

		conSrc, err = listener.AcceptTCP()
		if err != nil {
			fmt.Println("[Error] Can't start src listener")
			log.Fatal(err)
		}

		// establish connection to client
		conDst, err = net.DialTCP("tcp", nil, addrDst)
		if err != nil {
			fmt.Println("Dial to dst error, rerun the loop", err)
			log.Panicln(err)
		}
		fmt.Println("connected to", *connectPtr)
		isConnected = true
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
	// sdr_tcp.c structure size is 5
	/* struct command{
		unsigned char cmd;
		unsigned int param;
	    }__attribute__((packed));
		struct command cmd={0, 0};
		sizeof(cmd) = 5 bytes */
	cmdSize := 5

	srcBuf := make([]byte, cmdSize*10)

	for {
		if !isConnected {
			break
		}
		// Read data from src
		if srcReadWrite.Reader.Size() >= cmdSize {
			_, err := srcReadWrite.Read(srcBuf)
			if err != nil {
				fmt.Println("Read cmd from src error", err)
				break
			}
			if err == nil {
				// Write data to a Dst
				_, err := dstReadWrite.Write([]byte(srcBuf))
				if err != nil {
					fmt.Println("Write cmd to dst error")
					break
				}
			}
			dstReadWrite.Writer.Flush()
		}
	}
	isConnected = false
}

func handle_data_stream(srcReadWrite bufio.ReadWriter, dstReadWrite bufio.ReadWriter) {
	// Handling command channel - from dst/connect to src/listening
	// buffer for data (sdr)
	// out target is handle the stream from
	// https://github.com/blinick/rtl-sdr/blob/wip_rtltcp_ringbuf/src/rtl_tcp.c
	// every second with buffer less that 8Mb
	dataSize := 128 * 1024
	dstBuf := make([]byte, dataSize*4) // 512
	for {
		if !isConnected {
			break
		}
		// our rtl_tcp sending blocks every 1 second
		// let him the time to prepare
		// time.Sleep(1 * time.Second)
		// Read data from dst
		if dstReadWrite.Reader.Size() >= dataSize {
			_, err := dstReadWrite.Read(dstBuf)
			if err != nil {
				fmt.Println("Read data from dst error")
				break
			}
			if err == nil {
				// Write data to src
				_, err := srcReadWrite.Write([]byte(dstBuf))
				if err != nil {
					fmt.Println("Write data to src error")
					break
				}
			}
			srcReadWrite.Writer.Flush()
		}
	}
	isConnected = false
}
