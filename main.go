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



func Client(uri string) {

	addr, module, path, _ := rsync.SplitURI(uri)

	fmt.Println(module, path)

	conn, err := net.Dial("tcp", addr)
	// tuna: mirrors.tuna.tsinghua.edu.cn 101.6.8.193:873

	if err != nil {
		// TODO
	}

	defer conn.Close()

	rsync.HandShake(conn, module, path)

	// fmt.Println(readInteger(conn))
	log.Println("HandShake OK")

	data := make(chan byte, 16 * 1024 * 1024)


	// Start De-Multiplexing
	go rsync.DeMuxChan(conn, data)

	filelist := make(rsync.FileList, 0, 4096)
	// recv_file_list
	for {
		if rsync.GetFileList(data, &filelist) == io.EOF {
			break
		}
	}
	log.Println("File List Received, total size is", len(filelist))

	ioerr := rsync.GetInteger(data)
	log.Println("IOERR", ioerr)

	// Sort the filelist lexicographically
	sort.Sort(filelist)

	// Generate target file list
	//rsync.RequestAFile(conn, "libnemo-extension1_1.8.1+maya_amd64.deb", &filelist)
	//rsync.GetFiles(data, conn, &filelist)


	rsync.RequestFiles(conn, data, &filelist)
	//go rsync.Downloader(data, &filelist)
	//fmt.Println(filelist)




}

func main() {
	Client("rsync://mirrors.kernel.org/linuxmint-packages/pool/romeo/n/nemo/")
}


