package rsync

import (
	"encoding/binary"
	"errors"
	"io"
	"log"
)

type dMuxReader struct {
	in     io.ReadCloser
	remain uint32 // Default value: 0
	header []byte // Size: 4 bytes
}

func NewdMuxReader(reader io.ReadCloser) *dMuxReader {
	return &dMuxReader{
		in:     reader,
		remain: 0,
		header: make([]byte, 4),
	}
}

func (r *dMuxReader) Read(p []byte) (n int, err error) {
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

func (r *dMuxReader) readHeader() error {
	for {
		// Read header
		if _, err := io.ReadFull(r.in, r.header[:4]); err != nil {
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
			if _, err := r.in.Read(msg); err != nil {
				return err
			}
			return errors.New(string(msg))
		}
	}
}

func (r *dMuxReader) Close() error {
	return r.in.Close()
}

