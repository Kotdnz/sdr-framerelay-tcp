package main

import (
	"io"
	"log"
	"net"
	"os"

	"github.com/klauspost/compress/zstd"
	"github.com/pierrec/lz4/v4"
)

// Copy data between two connections. Return EOF on connection close.
func Pipe(a, b net.Conn, dir string, lvl string, alg string, conc int) error {
	done := make(chan error, 1)
	zstd_encLevel := zstd.SpeedDefault
	lz4_encLevel := lz4.Level3

	log.Println("Compressing type is:", dir)
	if alg == "zstd" {
		// parsing the level
		switch lvl {
		case "Fastest":
			zstd_encLevel = zstd.SpeedFastest
			log.Println("Compress level is Fastest")
		case "Default":
			zstd_encLevel = zstd.SpeedDefault
			log.Println("Compress level is: Default")
		case "Better":
			zstd_encLevel = zstd.SpeedBetterCompression
			log.Println("Compress level is: Better")
		case "Best":
			zstd_encLevel = zstd.SpeedBestCompression
			log.Println("Compress level is: Best")
		default:
			log.Println("Compress level is: Default")
		}
	}
	if alg == "lz4" {
		// parsing the level
		switch lvl {
		case "Fastest":
			lz4_encLevel = lz4.Fast
			log.Println("Compress level is Fastest")
		case "Default":
			lz4_encLevel = lz4.Level3
			log.Println("Compress level is: Default")
		case "Better":
			lz4_encLevel = lz4.Level7
			log.Println("Compress level is: Better")
		case "Best":
			lz4_encLevel = lz4.Level9
			log.Println("Compress level is: Best")
		default:
			log.Println("Compress level is: Default")
		}
	}
	cp := func(srcConn, dstConn net.Conn) {
		_, err := io.Copy(dstConn, srcConn)
		//log.Printf("Pure copied %d bytes from %s to %s", n, srcConn.RemoteAddr(), dstConn.RemoteAddr())
		done <- err
	}

	// Encoding
	enc := func(srcConn, dstConn net.Conn) {
		if alg == "zstd" {
			enc, err := zstd.NewWriter(io.WriteCloser(dstConn),
				zstd.WithEncoderLevel(zstd_encLevel),
				zstd.WithEncoderConcurrency(conc))
			//		    zstd.WithZeroFrames(true))

			if err != nil {
				log.Println("encoding ZSTD error", err)
				done <- err
				return
			}
			defer enc.Close()
			_, err = io.Copy(enc, srcConn)
			//log.Printf("Encode copied %d bytes from %s to %s", n, srcConn.RemoteAddr(), dstConn.RemoteAddr())
			if err != nil {
				log.Println("encoding ZSTD copy error", err)
				enc.Close()
				done <- err
				return
			}
			err = enc.Close()
			done <- err
			return
		} else {
			if alg == "lz4" {
				enc4 := lz4.NewWriter(io.WriteCloser(dstConn))
				enc4.Apply(lz4.CompressionLevelOption(lz4_encLevel),
					lz4.ConcurrencyOption(conc))
				defer enc4.Close()
				_, err := io.Copy(enc4, srcConn)
				//log.Printf("Encode copied %d bytes from %s to %s", n, srcConn.RemoteAddr(), dstConn.RemoteAddr())
				if err != nil {
					log.Println("encoding LZ4 copy error", err)
					enc4.Close()
					done <- err
					return
				}
				err = enc4.Close()
				done <- err
				return
			} else {
				log.Printf("Wrong compress algorithm")
				os.Exit(-1)
			}
		}
	}

	// Decoding
	dec := func(srcConn, dstConn net.Conn) {
		if alg == "zstd" {
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
		} else {
			if alg == "lz4" {
				dec4 := lz4.NewReader(io.Reader(srcConn))
				_, err := io.Copy(dstConn, dec4)
				//log.Printf("Decode copied %d bytes from %s to %s", n, srcConn.RemoteAddr(), dstConn.RemoteAddr())
				done <- err
			} else {
				log.Printf("Wrong compress algorithm")
				os.Exit(-1)
			}
		}
	}

	// a=upConn, b=downConn
	// encode - applied to downConn (b)
	// decode - applied to upConn (a) listen

	switch dir {
	case "encode":
		go cp(a, b)
		go enc(b, a)
	case "decode":
		go cp(a, b)
		go dec(b, a)
	default:
		go cp(a, b)
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
