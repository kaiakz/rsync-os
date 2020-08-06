package rsync

import "io"

// This struct mainly has two attributes, both of them can be used for a plain socket or an SSH channel
type Conn struct {
	Writer    io.Writer // Read only
	Reader    io.Reader // Write only
	bytespool []byte    // Default size: 8 bytes
}

func (conn *Conn) Write(p []byte) (n int, err error) {
	return conn.Writer.Write(p)
}

func (conn *Conn) Read(p []byte) (n int, err error) {
	return conn.Reader.Read(p)
}

