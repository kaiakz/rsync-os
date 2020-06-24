package rsync

import (
	"bytes"
	"encoding/binary"
	"net"
)

/* Raw data from socket */

func ReadLine(conn net.Conn) (string, error) {
	ch := make([]byte, 1)
	line := new(bytes.Buffer)
	for {
		_, err := conn.Read(ch)
		if err != nil {
			return "", err
		}

		if ch[0] == '\r' {
			continue
		}

		_, err = line.Write(ch)
		if err != nil {
			return "", err
		}

		if ch[0] == '\n' {
			line.WriteByte(0)
			break
		}

		if ch[0] == 0 {
			break
		}
	}
	return line.String(), nil

}

func ReadBuffer(conn net.Conn, size uint32) (string, error) {
	c := make([]byte, 1)
	buffer := new(bytes.Buffer)
	var i uint32
	for i = 0; i < size; i++ {
		_, err := conn.Read(c)
		if err != nil {
			return "", err
		}
		buffer.Write(c)
	}
	return buffer.String(), nil
}

func ReadShort(conn net.Conn) int16 {
	data := make([]byte, 2)
	conn.Read(data)
	return int16(binary.LittleEndian.Uint16(data))
}

func ReadByte(conn net.Conn) byte {
	data := make([]byte, 1)
	conn.Read(data)
	return data[0]
}

func ReadUint8(conn net.Conn) uint8 {
	data := make([]byte, 1)
	conn.Read(data)
	return uint8(data[0])
}

func ReadInteger(conn net.Conn) int32 {
	data := make([]byte, 4)
	conn.Read(data)
	return int32(binary.LittleEndian.Uint32(data))
}

func ReadLong(conn net.Conn) int64 {
	data := make([]byte, 8)
	conn.Read(data)
	return int64(binary.LittleEndian.Uint64(data))
}

func ReadExact(conn net.Conn, b []byte) (int, error) {
	for i := 0; i < len(b); {
		n, err := conn.Read(b[i:])
		if err != nil {
			return n, err
		}
		i += n
	}
	return len(b), nil
}
