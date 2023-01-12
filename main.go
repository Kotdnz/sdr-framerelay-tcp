/** @file sdr-framerelay-tcp.go
 *
 * @brief fremarelay between source and destination to compress the data stream only
 * from https://github.com/blinick/rtl-sdr/
 * @github https://github.com/Kotdnz/sdr-framerelay-tcp
 * @author Kostiantyn Nikonenko
 * @date January, 12, 2023
 * @lib https://github.com/klauspost/compress/tree/master/zstd
 * @lib https://github.com/pierrec/lz4
 */

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
)

var Version string = "v.3.2 January, 12, 2023"

func main() {
	fmt.Println("sdr-fremarelay-tcp version: ", Version)

	// read CLI
	flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	listen := flag.String("listen", "0.0.0.0:9001", "listen IP:Port.")
	backend := flag.String("connect", "127.0.0.1:9002", "connect to IP:Port.")
	compressPtr := flag.String("compress", "no", "Possible options: 'decode' on last hop, 'encode' on first hop, and 'no'")
	compressLevel := flag.String("level", "Fastest", "The compressing level. Options: Fastest (lvl 1), Default (lvl 3), Better (lvl 7), Best (lvl 11-zstd / 9-lz4)")
	compressAlg := flag.String("algorithm", "zstd", "Compressing algorithm: 'zstd' or 'lz4'")
	concurency := flag.Int("conc", 2, "Concurrency: 1,2,3")

	flag.Parse()

	p := Proxy{Listen: *listen, Backend: *backend,
		compressDir: *compressPtr,
		compressLvl: *compressLevel,
		compressAlg: *compressAlg,
		concurrency: *concurency}

	sigs := make(chan os.Signal)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		log.Println("\r- Ctrl+C pressed in Terminal")
		if err := p.Close(); err != nil {
			log.Fatal(err.Error())
		}
		os.Exit(0)
	}()

	if err := p.Run(); err != nil {
		log.Fatal(err.Error())
	}
}
