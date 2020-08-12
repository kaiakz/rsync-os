package rsync

import (
	"encoding/binary"
	"io"
	"log"
)

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

type MuxReader struct {
	in      io.ReadCloser
	Data    chan byte
	closeCh chan byte
}

func NewMuxReader(reader io.ReadCloser) *MuxReader {
	mr := &MuxReader{
		in:      reader,
		Data:    make(chan byte, 16 * MB),
		closeCh: make(chan byte),
	}
	// Demux in Goroutine
	go func() {

		header := make([]byte, 4)	// Header size: 4 bytes
		var dsize uint32 = 1 << 16	// Default size: 65536
		bytespool := make([]byte, dsize)

		for {
			select {
			case <-mr.closeCh: // Close the channel, then exit the goroutine
				close(mr.Data)
				return
			default:
				// read the multipex data & put them to channel
				_, err := reader.Read(header)
				if err != nil {
					// panic("Multiplex: wire protocol error")
					log.Println("Multiplex: wire protocol error")
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
					_, err := reader.Read(body)

					// FIXME: Never return EOF
					if err != nil { // The connection was closed by server
						panic(err)
					}

					for _, b := range body {
						mr.Data <- b
					}

				} else { // out-of-band data
					//otag := tag - 7
					panic("Error: out-of-band")
				}
			}
		}
	}()
	return mr
}

// Never return error
func (r *MuxReader) Read(p []byte) (n int, err error) {
	for i, _ := range p {
		p[i] = <- r.Data
	}
	return len(p), nil
}

func (r *MuxReader) Close() error {
	r.closeCh <- 0	// close the channel Data & exit the demux goroutine
	return r.in.Close()
}
