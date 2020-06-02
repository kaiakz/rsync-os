// Send @RSYNCD x.x\n
// Send modname\n
// Send arugment with mod list\0	filter list write(0)    \n
// handshake
// batch seed
// Recv file list
//

package main

import (
	"io"
	"log"
	"net"
	"rsync2os/rsync"
	"sort"
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
		// TODO
	}

	defer conn.Close()

	rsync.HandShake(conn)

	// fmt.Println(readInteger(conn))
	log.Println("HandShake OK")

	data := make(chan byte, 16 * 1024 * 1024)


	go rsync.DeMuxChan(conn, data)

	filelist := make(rsync.FileList, 0, 3072)
	// recv_file_list
	for {
		if rsync.GetFileList(data, &filelist) == io.EOF {
			break
		}
	}
	log.Println("Received File List OK, total size is", len(filelist))

	ioerr := rsync.GetInteger(data)
	log.Println("IOERR", ioerr)

	// Sort the filelist lexicographically
	sort.Sort(filelist)

	// Generate target file list
	rsync.Generate(conn, &filelist)

	//fmt.Println(filelist)

	rsync.GetFiles(data, conn, &filelist)



}

func main() {
	Client()
}

