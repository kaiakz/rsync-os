package rsync

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
	"strings"
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

func ReadExact(conn net.Conn, b []byte) (int, error) {
	for i:= 0; i < len(b); {
		n, err := conn.Read(b[i:])
		if err != nil {
			return n, err
		}
		i += n
	}
	return len(b), nil
}


func SplitURIS(uri string) (string, int, string, string, error){

	var host, module, path string
	var first = []byte(uri)
	var second []byte

	if strings.HasPrefix(uri, "rsync://") {
		/* rsync://host[:port]/module[/path] */
		first = first[8:]
		i := bytes.IndexByte(first, '/')
		if i == -1 {
			// No module name
			panic("No module name")
		}
		second = first[i+1:]	//ignore '/'
		first = first[:i]
	} else {
		// Only for remote
		/* host::module[/path] */
		panic("No implement yet")
	}

	port := 873		// Default port: 873

	// Parse port
	i := bytes.IndexByte(first, ':')
	if i != -1  {
		var err error
		port, err = strconv.Atoi(string(first[i+1:]))
		if err != nil {
			// Wrong port
			panic("Wrong port")
		}
		first = first[:i]
	}
	host = string(first)

	// Parse path
	i = bytes.IndexByte(second, '/')
	if i != -1 {
		path = string(second[i:])
		second = second[:i]
	}
	module = string(second)

	return host, port, module, path, nil

}

// For rsync
func SplitURI(uri string) (string, string, string, error){

	var address, module, path string
	var first = []byte(uri)
	var second []byte

	if strings.HasPrefix(uri, "rsync://") {
		/* rsync://host[:port]/module[/path] */
		first = first[8:]
		i := bytes.IndexByte(first, '/')
		if i == -1 {
			// No module name
			panic("No module name")
		}
		second = first[i+1:]	//ignore '/'
		first = first[:i]
	} else {
		// Only for remote
		/* host::module[/path] */
		panic("No implement yet")
	}

	address = string(first)
	// Parse port
	i := bytes.IndexByte(first, ':')
	if i == -1  {
		address += ":873"	// Default port: 873
	}

	// Parse path
	i = bytes.IndexByte(second, '/')
	if i != -1 {
		path = string(second[i:])
		second = second[:i]
	}
	module = string(second)

	return address, module, path, nil

}