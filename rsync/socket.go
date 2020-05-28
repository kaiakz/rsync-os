package rsync

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
)

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
	fmt.Println(data)
	return data[0]
}

func ReadUint8(conn net.Conn) uint8 {
	data := make([]byte, 1)
	conn.Read(data)
	fmt.Println(data)
	return uint8(data[0])
}

func ReadInteger(conn net.Conn) int32 {
	data := make([]byte, 4)
	conn.Read(data)
	fmt.Println(data)
	return int32(binary.LittleEndian.Uint32(data))
}

func ReadLong(conn net.Conn) int64 {
	data := make([]byte, 8)
	conn.Read(data)
	fmt.Println(data)
	return int64(binary.LittleEndian.Uint64(data))
}

func ReadFList(conn net.Conn) {

	flags := ReadByte(conn)

	var partial, pathlen uint32 = 0, 0

	fmt.Println(flags)
	/*
	 * Read our filename.
	 * If we have FLIST_NAME_SAME, we inherit some of the last
	 * transmitted name.
	 * If we have FLIST_NAME_LONG, then the string length is greater
	 * than byte-size.
	 */
	if (0x20 & flags) != 0 {
		partial = uint32(ReadByte(conn))
		fmt.Println("Partical", partial)
	}

	/* Get the (possibly-remaining) filename length. */
	if (0x40 & flags) != 0 {
		pathlen = uint32(ReadInteger(conn)) // can't use for rsync 31
		// Var int
		// i := readByte(conn)
		// var j byte, len int
		// for j = 0; j<=6; ++j {
		// 	if ((i & 0x80) == 0)	break
		// 	c =
		// }

	} else {
		pathlen = uint32(ReadByte(conn))
	}
	fmt.Println("PathLen", pathlen)

	/* Allocate our full filename length. */
	/* FIXME: maximum pathname length. */
	// if pathlen + partical == 0
	// malloc len error?
	//
	if (0x20 & flags) != 0 {
		// return last 4096bytes
	}

	path, _ := ReadBuffer(conn, pathlen)
	fmt.Println("Path", path)

	size := ReadInteger(conn)
	fmt.Print("Size", size)

	if (flags & 0x80) == 0 {
		fmt.Println("MTIME", ReadInteger(conn))
	}

	var mode int32
	if (flags & 0x02) == 0 {
		mode = ReadInteger(conn)
		fmt.Println("Mode", mode)
	}

	if ((mode & 32768) != 0) && ((mode & 8192) != 0) {
		len := uint32(ReadInteger(conn))
		slink, _ := ReadBuffer(conn, len)
		fmt.Println("Symbolic", len, "is", slink)
	}

	//return "\n"
}

func ReadExact(conn net.Conn, b []byte) (int, error) {
	for i:= 0; i < len(b); {
		n, err := conn.Read(b[i:])
		if err != nil {
			return n, nil
		}
		i += n
	}
	return len(b), nil
}
