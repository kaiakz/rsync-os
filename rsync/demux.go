package rsync

import (
	"encoding/binary"
	"errors"
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
	in     io.ReadCloser
	remain uint32 // Default value: 0
	header []byte // Size: 4 bytes
}

func NewMuxReader(reader io.ReadCloser) *MuxReader {
	return &MuxReader{
		in:     reader,
		remain: 0,
		header: make([]byte, 4),
	}
}

func (r *MuxReader) Read(p []byte) (n int, err error) {
	if r.remain == 0 {
		err := r.readHeader()
		if err != nil {
			return 0, err
		}
	}
	rlen := uint32(len(p))
	if rlen > r.remain {	// Min(len(p), remain)
		rlen = r.remain
	}
	n, err = r.in.Read(p[:rlen])
	r.remain = r.remain - uint32(n)
	return
}

func (r *MuxReader) readHeader() error {
	for {
		// Read header
		if _, err := io.ReadFull(r.in, r.header); err != nil {
			return err
		}
		tag := r.header[3]                                        // Little Endian
		size := (binary.LittleEndian.Uint32(r.header) & 0xffffff) // TODO: zero?

		log.Printf("<DEMUX> tag %d size %d\n", tag, size)

		if tag == (MUX_BASE + MSG_DATA) { // MUX_BASE + MSG_DATA
			r.remain = size
			return nil
		} else { // out-of-band data
			// otag := tag - 7
			msg := make([]byte, size)
			if _, err := io.ReadFull(r.in, msg); err != nil {
				return err
			}
			return errors.New(string(msg))
		}
	}
}

func (r *MuxReader) Close() error {
	return r.in.Close()
}

