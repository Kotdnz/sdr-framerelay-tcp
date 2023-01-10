/** @file sdr-framerelay-tcp.go
 *
 * @brief fremarelay between source and destination to optimize the stream
 * and even compress tcp flow from https://github.com/blinick/rtl-sdr/
 * @source https://github.com/Kotdnz/sdr-framerelay-tcp
 * @author Kostiantyn Nikonenko
 * @date January, 10, 2023
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

var Version string = "v.2.1"

func main() {
	fmt.Println("sdr-fremarelay-tcp version: ", Version)

	// read CLI
	flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	listen := flag.String("listen", "0.0.0.0:9001", "listen IP:Port by default is [0.0.0.0:9001]")
	backend := flag.String("connect", "127.0.0.1:9002", "connect IP:Port by default is [127.0.0.1:9002]")
	compressPtr := flag.String("compress", "no", "what end of transport will be compressed/decompress. Default is [no], possible options: 'decode' on last hop, 'encode' on first hop")
	compressLevel := flag.String("speed", "default", "The compressing level. Options: Fastest (lvl 1), Default (lvl 3), Better (lvl 7), Best (lvl 11)")

	flag.Parse()

	fmt.Println("Compressed is: ", *compressPtr, ", level is ", *compressLevel)

	p := Proxy{Listen: *listen, Backend: *backend, compressDir: *compressPtr, compressLvl: *compressLevel}

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

	cp := func(r, w net.Conn) {
		n, err := io.Copy(r, w)
		log.Printf("Pure copied %d bytes from %s to %s", n, r.RemoteAddr(), w.RemoteAddr())
		done <- err
	}

	// Encoding
	enc := func(in io.Reader, out io.Writer) error {
		enc, err := zstd.NewWriter(out, zstd.WithEncoderLevel(encLevel))
		if err != nil {
			return err
		}
		n, err := io.Copy(enc, in)
		log.Printf("[Encoded] copied %d bytes ", n)
		if err != nil {
			enc.Close()
			return err
		}
		err = enc.Close()
		done <- err
		return err
	}

	// Decoding
	dec := func(in io.Reader, out io.Writer) error {
		dec, err := zstd.NewReader(in)
		if err != nil {
			return err
		}
		defer dec.Close()

		n, err := io.Copy(out, dec)
		log.Printf("[Decoded] copied %d bytes ", n)
		done <- err
		return err
	}

	// a=upConn, b=downConn
	// encode - applied to downConn (b)
	// decode - applied to upConn (a)

	switch dir {
	case "no":
		go cp(a, b)
		go cp(b, a)
	case "encode":
		go enc(a, b)
		go cp(b, a)
	case "decode":
		go dec(a, b)
		go cp(b, a)
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
