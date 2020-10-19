package rsync

import (
	"compress/flate"
	"io"
)

/*
	rsync uses zlib to do compression, the windowsBits is -15: raw data
 */

const (
	END_FLAG = 0
	TOKEN_LONG = 0x20
	TOKENRUN_LONG = 0x21
	DEFLATED_DATA = 0x40
	TOKEN_REL = 0x80
	TOKENRUN_REL = 0xc0

)

// RFC 1951: https://tools.ietf.org/html/rfc1951
type flatedtokenReader struct {
	in           Conn
	flatedwraper *flatedWraper
	decompressor io.ReadCloser
	savedflag    byte
	flag         byte
	remains      uint32
}



func NewflatedtokenReader(reader Conn) *flatedtokenReader {
	w := &flatedWraper{
		raw: &reader,
		end: [4]byte{0, 0, 0xff, 0xff},
	}
	return &flatedtokenReader{
		in: reader,
		flatedwraper: w,
		decompressor: flate.NewReader(w),
		savedflag: -1,
		flag: 0,
		remains: 0,
	}
}

// Update flag & len of remain data
func (f *flatedtokenReader) readFlag() error {
	if f.savedflag != 0 {
		f.flag = f.savedflag & 0xff
		f.savedflag = 0
	} else {
		var err error
		if f.flag, err = f.in.ReadByte(); err != nil {
			return err
		}
	}
	if (f.flag & 0xc0) == DEFLATED_DATA {
		l, err := f.in.ReadByte()
		if err != nil {
			return err
		}
		f.remains = uint32(f.flag & 0x3f) << 8 + uint32(l)
	}
	return nil
}

func (f *flatedtokenReader) Read(p []byte) (n int, err error) {
	n, err = f.decompressor.Read(p)
	f.remains -= uint32(n)
	return
}

func (f *flatedtokenReader) Close() error {
	return f.decompressor.Close()
}

// Hack only: rsync need to append 4 bytes(0, 0, ff, ff) at the end.
type flatedWraper struct {
	raw io.Reader
	end [4]byte
}

func (f *flatedWraper) Read(p []byte) (n int, err error) {
	// Just append 4 bytes to the end of stream
	return f.raw.Read(p)
}