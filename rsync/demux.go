package rsync

import (
	"encoding/binary"
	"io"
	"os"

	//"io/ioutil"
	"log"
	"net"
)

//channel: read & write

//Multiplexing
//Most rsync transmissions are wrapped in a multiplexing envelope protocol.  It is
//composed as follows:
//
//1.   envelope header (4 bytes)
//2.   envelope payload (arbitrary length)
//
//The first byte of the envelope header consists of a tag.  If the tag is 7, the pay‐
//load is normal data.  Otherwise, the payload is out-of-band server messages.  If the
//tag is 1, it is an error on the sender's part and must trigger an exit.  This limits
//message payloads to 24 bit integer size, 0x00ffffff.
//
//The only data not using this envelope are the initial handshake between client and
//server

// Goroutine: Demultiplex the package, and push them to channel
// data: Buffered Channel
// FIXME: How to close the channel & goroutine
func DeMuxChan(conn net.Conn, data chan byte) {
	// conn read the multipex data & put them to channel
	header := make([]byte, 4)	// Header size: 4 bytes
	var dsize uint32 = 1 << 16	// Default size: 65536
	bytespool := make([]byte, dsize)

	for {
		n, err := ReadExact(conn, header)
		if n != 4 || err != nil {
			// panic("Mulitplex: Check your wired protocol")
			log.Println("Mulitplex: Check your wire protocol")
			return
		}

		tag := header[3]                                        // Little Endian
		size := (binary.LittleEndian.Uint32(header) & 0xffffff) // TODO: zero?

		log.Printf("<DEMUX> tag %d size %d\n", tag, size)

		if tag == (MUX_BASE + MSG_DATA) { // MUX_BASE + MSG_DATA
			if size > dsize {
				bytespool = make([]byte, size)
				dsize = size
			}

			body := bytespool[:size]

			_, err := ReadExact(conn, body)

			// FIXME: Never return EOF
			if err == io.EOF { // Finish
				panic("EOF")
			}

			for _, b := range body {
				data <- b
			}

		} else { // out-of-band data
			//otag := tag - 7
			panic("Error")
		}
	}
}

// Blocking: copy len(b) bytes from channel to b
func GetBytes(data chan byte, b []byte) {
	for i := 0; i < len(b); i++ {
		b[i] = <-data
	}
}

func GetShort(data chan byte) int16 {
	val := make([]byte, 2)
	GetBytes(data, val)
	return int16(binary.LittleEndian.Uint16(val))
}

func GetByte(data chan byte) byte {
	return <-data
}

func GetUint8(data chan byte) uint8 {
	return uint8(<-data)
}

func GetInteger(data chan byte) int32 {
	val := make([]byte, 4)
	GetBytes(data, val)
	return int32(binary.LittleEndian.Uint32(val))
}

func GetFileMode(data chan byte) os.FileMode {
	val := make([]byte, 4)
	GetBytes(data, val)
	return os.FileMode(binary.LittleEndian.Uint32(val))
}

func GetLong(data chan byte) int64 {
	val := make([]byte, 8)
	GetBytes(data, val)
	return int64(binary.LittleEndian.Uint64(val))
}

func GetVarint(data chan byte) int64 {
	sval := GetInteger(data)
	if sval != -1 {
		return int64(sval)
	}

	return GetLong(data)
}

