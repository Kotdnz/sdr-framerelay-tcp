/** @file sdr-framerelay-tcp.go
 *
 * @brief fremarelay between source and destination to compress the data stream only
 * from https://github.com/blinick/rtl-sdr/
 * @github https://github.com/Kotdnz/sdr-framerelay-tcp
 * @author Kostiantyn Nikonenko
 * @date January, 12, 2023
 * @lib https://github.com/klauspost/compress/tree/master/zstd
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

	"github.com/klauspost/compress/zstd"
)

var Version string = "v.2.3"

func main() {
	fmt.Println("sdr-fremarelay-tcp version: ", Version)

	// read CLI
	flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	listen := flag.String("listen", "0.0.0.0:9001", "listen IP:Port.")
	backend := flag.String("connect", "127.0.0.1:9002", "connect to IP:Port.")
	compressPtr := flag.String("compress", "no", "what end of transport will be compressed/decompress. Possible options: 'decode' on last hop, 'encode' on first hop, and 'no'")
	compressLevel := flag.String("level", "Default", "The compressing level. Options: Fastest (lvl 1), Default (lvl 3), Better (lvl 7), Best (lvl 11)")

	flag.Parse()

	fmt.Println("Compress is:", *compressPtr, ", level is", *compressLevel)

	p := Proxy{Listen: *listen, Backend: *backend, compressDir: *compressPtr, compressLvl: *compressLevel}

	sigs := make(chan os.Signal)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		log.Println("\r- Ctrl+C pressed in Terminal")
		if err := p.Close(); err != nil {
			log.Fatal(err.Error())
		}
	}()

	if err := p.Run(); err != nil {
		log.Fatal(err.Error())
	}
}

// Copy data between two connections. Return EOF on connection close.
func Pipe(a, b net.Conn, dir string, lvl string) error {
	done := make(chan error, 1)

	// parsing the level
	encLevel := zstd.SpeedDefault
	switch lvl {
	case "Fastest":
		encLevel = zstd.SpeedFastest
	case "Default":
		encLevel = zstd.SpeedDefault
	case "Better":
		encLevel = zstd.SpeedBetterCompression
	case "Best":
		encLevel = zstd.SpeedBestCompression
	}

	cp := func(srcConn, dstConn net.Conn) {
		_, err := io.Copy(dstConn, srcConn)
		//log.Printf("Pure copied %d bytes from %s to %s", n, srcConn.RemoteAddr(), dstConn.RemoteAddr())
		done <- err
	}

	// Encoding
	enc := func(srcConn, dstConn net.Conn) {
		enc, err := zstd.NewWriter(io.WriteCloser(dstConn),
			zstd.WithEncoderLevel(encLevel),
			zstd.WithEncoderConcurrency(1),
			zstd.WithZeroFrames(true))
		if err != nil {
			log.Println("encoding error", err)
			done <- err
			return
		}
		defer enc.Close()
		_, err = io.Copy(enc, srcConn)
		//log.Printf("Encode copied %d bytes from %s to %s", n, srcConn.RemoteAddr(), dstConn.RemoteAddr())
		if err != nil {
			log.Println("encoding copy error", err)
			//enc.Close()
			done <- err
			return
		}
		//err = enc.Close()
		done <- err
	}

	// Decoding
	dec := func(srcConn, dstConn net.Conn) {
		dec, err := zstd.NewReader(io.Reader(srcConn))
		if err != nil {
			log.Println("Decoding error", err)
			done <- err
			return
		}
		defer dec.Close()
		_, err = io.Copy(dstConn, dec)
		//log.Printf("Decode copied %d bytes from %s to %s", n, srcConn.RemoteAddr(), dstConn.RemoteAddr())
		done <- err
	}

	// a=upConn, b=downConn
	// encode - applied to downConn (b)
	// decode - applied to upConn (a) listen

	switch dir {
	case "no":
		go cp(a, b)
		go cp(b, a)
	case "encode":
		go cp(a, b)
		go enc(b, a)
	case "decode":
		go cp(a, b)
		go dec(b, a)
	}

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
