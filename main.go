// Send @RSYNCD x.x\n
// Send modname\n
// Send arugment with mod list\0	filter list write(0)    \n
// handshake
// batch seed
// Recv file list
//

package main

import (
	"fmt"
	"io"
	"net"
	"rsync2os/rsync"
	"strings"
)

// type FList struct {
// 	path, wpath, link string
// 	FLStat struct {
// 		mode, uid, gid, rdev, size, time uint32
// 	} st
// }



func Client() {

	conn, err := net.Dial("tcp", "101.6.8.193:873")
	if err != nil {

	}

	defer conn.Close()

	// send my version
	// send("@RSYNCD: 31.0\n");
	conn.Write([]byte("@RSYNCD: 27.0\n"))

	// receive server's protocol version and seed
	version_str, _ := rsync.ReadLine(conn)

	var remote_protocol, remote_sub int
	fmt.Sscanf(version_str, "@RSYNCD: %d.%d", remote_protocol, remote_sub)
	fmt.Println(version_str)

	// recv(version)
	// scanf(version, "@RSYNCD: %d.%d", )

	// send mod name
	// send("Foo\n")
	conn.Write([]byte("epel\n"))
	// conn.Write([]byte("\n"))

	for {
		// Wait for '@RSYNCD: OK': until \n, then add \0
		res, _ := rsync.ReadLine(conn)
		fmt.Print(res)
		if strings.HasPrefix(res, "@RSYNCD: OK") {
			break
		}
	}

	// send parameters list
	//conn.Write([]byte("--server\n--sender\n-g\n-l\n-o\n-p\n-D\n-r\n-t\n.\nepel/7/SRPMS\n\n"))
	conn.Write([]byte("--server\n--sender\n-l\n-p\n-r\n-t\n.\nepel/7/SRPMS\n\n"))	// without gid, uid, mdev

	// read int32 as seed
	bseed := rsync.ReadInteger(conn)
	fmt.Println("SEED", bseed)

	// send filter_list, empty is 32-bit zero
	conn.Write([]byte("\x00\x00\x00\x00"))

	// fmt.Println(readInteger(conn))
	fmt.Println("Handshake OK")

	data := make(chan byte, 1024*1024)
	go rsync.DeMuxChan(conn, data)

	filelist := make([]rsync.FileInfo, 0, 3072)
	// recv_file_list
	for {
		if rsync.GetEntry(data, &filelist) == io.EOF {
			break
		}
	}
	fmt.Println("Received File List OK, total size is", len(filelist))

	ioerr := rsync.GetInteger(data)
	fmt.Println("IOERR", ioerr)

	// Generate target file list

}

func main() {
	Client()
}

