package rsync

import (
	"encoding/binary"
)

// io.Reader + io.Closer
type Reader interface {
	Read(p []byte) (n int, err error)
	Close() error
}

// io.Writer + io.Closer
type Writer interface {
	Write(p []byte) (n int, err error)
	Close() error
}

// This struct has two main attributes, both of them can be used for a plain socket or an SSH channel
type Conn struct {
	writer    Writer // Read only
	reader    Reader // Write only
	bytespool []byte // Default size: 8 bytes
}

func (conn *Conn) Write(p []byte) (n int, err error) {
	return conn.writer.Write(p)
}

func (conn *Conn) Read(p []byte) (n int, err error) {
	return conn.reader.Read(p)
}

/* Encoding: little endian */
// size of: int: 4, long: 8, varint: 4 or 8
func (conn *Conn) ReadByte() byte {
	val := conn.bytespool[:1]
	_, _ = conn.Read(val)
	return conn.bytespool[0]
}

func (conn *Conn) ReadShort() int16 {
	val := conn.bytespool[:2]
	_, _ = conn.Read(val)
	return int16(binary.LittleEndian.Uint16(val))
}

func (conn *Conn) ReadInt() int32 {
	val := conn.bytespool[:4]
	_, _ = conn.Read(val)
	return int32(binary.LittleEndian.Uint32(val))
}

func (conn *Conn) ReadLong() int64 {
	val := conn.bytespool[:8]
	_, _ = conn.Read(val)
	return int64(binary.LittleEndian.Uint64(val))
}

func (conn *Conn) ReadVarint() int64 {
	sval := conn.ReadInt()
	if sval != -1 {
		return int64(sval)
	}
	return conn.ReadLong()
}

// For Byte, Short, Int or Long (excepts Varint)

func  (conn *Conn) WriteByte(data byte) error {
	return binary.Write(conn.writer, binary.LittleEndian, data)
}

func  (conn *Conn) WriteShort(data int16) error {
	return binary.Write(conn.writer, binary.LittleEndian, data)
}

func  (conn *Conn) WriteInt(data int32) error {
	return binary.Write(conn.writer, binary.LittleEndian, data)
}

func  (conn *Conn) WriteLong(data int64) error {
	return binary.Write(conn.writer, binary.LittleEndian, data)
}

// TODO: In fact, both writer and reader are based on a same Connection (socket, SSH), how to close them twice?
func (conn *Conn) Close() error {
	_ = conn.writer.Close()
	_ = conn.reader.Close()
	return nil
}
