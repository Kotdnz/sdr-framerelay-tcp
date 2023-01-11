/** @file sdr-framerelay-tcp.go
 *
 * @brief fremarelay between source and destination to optimize the stream
 * and even compress tcp flow from https://github.com/blinick/rtl-sdr/
 * @source https://github.com/Kotdnz/sdr-framerelay-tcp
 * @author Kostiantyn Nikonenko
 * @date January, 11, 2023
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

var Version string = "v.2.2"

func main() {
	fmt.Println("sdr-fremarelay-tcp version: ", Version)

	// read CLI
	flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	listen := flag.String("listen", "0.0.0.0:9001", "listen IP:Port.")
	backend := flag.String("connect", "127.0.0.1:9002", "connect to IP:Port.")
	compressPtr := flag.String("compress", "no", "what end of transport will be compressed/decompress. Possible options: 'decode' on last hop, 'encode' on first hop, and 'no'")
	compressLevel := flag.String("speed", "Default", "The compressing level. Options: Fastest (lvl 1), Default (lvl 3), Better (lvl 7), Best (lvl 11)")

	flag.Parse()

	fmt.Println("Compress is:", *compressPtr, ", level is", *compressLevel)

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
	enc := func(r, w net.Conn) {
		//enc, err := zstd.NewWriter(io.WriteCloser(w), zstd.WithEncoderLevel(encLevel))
		// no idea on howto make this lib works https://github.com/klauspost/compress/tree/master/zstd
		enc, err := zstd.NewWriter(w, zstd.WithEncoderLevel(encLevel))
		if err != nil {
			log.Println("encoding error", err)
			done <- err
			return
		}
		_, err = io.Copy(enc, r)

		if err != nil {
			log.Println("encoding copy error", err)
			enc.Close()
			done <- err
			return
		}
		err = enc.Close()
		done <- err
	}

	// Decoding
	dec := func(r, w net.Conn) {
		// dec, err := zstd.NewReader(io.Reader(r))
		dec, err := zstd.NewReader(r)
		if err != nil {
			log.Println("Decoding error", err)
			done <- err
			return
		}
		defer dec.Close()
		_, err = io.Copy(w, dec)
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
