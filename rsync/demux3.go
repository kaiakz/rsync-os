package rsync

import (
	"encoding/binary"
	"errors"
	"io"
	"log"
)

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
			if _, err := r.in.Read(msg); err != nil {
				return err
			}
			return errors.New(string(msg))
		}
	}
}

func (r *MuxReader) Close() error {
	return r.in.Close()
}

