// Send @RSYNCD x.x\n
// Send modname\n
// Send arugment with mod list\0	filter list write(0)    \n
// handshake
// batch seed
// Recv file list
//

package main

import (
	"flag"
	"fmt"
	"github.com/spf13/viper"
	"io"
	"log"
	"net"
	"rsync-os/fldb"
	"rsync-os/rsync"
	"sort"
	"time"
)

func Socket(uri string, dest string) {

	addr, module, path, err := rsync.SplitURI(uri)

	if err != nil {
		log.Println("Invaild URI")
		return
	}

	log.Println(module, path)

	conn, err := net.Dial("tcp", addr)
	//rAddr, err := net.ResolveTCPAddr("tcp", addr)
	//tconn, err := net.DialTCP("tcp", nil, rAddr)
	//tconn.SetReadBuffer(4096)
	//tconn.SetWriteBuffer(4096)

	if err != nil {
		// TODO
		log.Fatalln("Network Error")
	}

	defer conn.Close()

	c := &rsync.SocketConn{
		Conn: conn,
		DemuxIn: make(chan byte, 16*1024*1024),
	}

	c.HandShake(module, path)

	// fmt.Println(readInteger(conn))
	log.Println("HandShake OK")

	// Start De-Multiplexing
	go rsync.DeMuxChan(conn, c.DemuxIn)

	filelist := make(rsync.FileList, 0, 1024 * 1024)
	// recv_file_list
	for {
		if rsync.GetFileList(c.DemuxIn, &filelist) == io.EOF {
			break
		}
	}
	log.Println("File List Received, total size is", len(filelist))

	ioerr := rsync.GetInteger(c.DemuxIn)
	log.Println("IOERR", ioerr)

	// Sort the filelist lexicographically
	sort.Sort(filelist)

	ppath := rsync.TrimPrepath(path)
	//fldb.Snapshot(filelist[:], module, ppath)
	if viper.GetStringMapString(dest) == nil {
		log.Fatalln("Lack of ", dest)
	}
	dbconf := viper.GetStringMapString(dest + ".boltdb")
	cache := fldb.Open(dbconf["path"], []byte(module), []byte(ppath))
	if cache == nil {
		// TODO
		log.Fatalln("Failed to init cache")
	}
	downloadList, deleteList := cache.Diff(filelist[:])
	fmt.Println(len(downloadList))
	for _, d := range downloadList {
		fmt.Println(string(filelist[d].Path))
	}
	fmt.Println(len(deleteList))
	for _, d := range deleteList {
		fmt.Println(string(d))
	}

	// Update file list && start downloading

	log.Println("File List Saved")

	//c.FinalPhase()

	// FIXME: Close fldb & network
	cache.Close()
	return

	// Init the object storage
	// For test
	//minioConf := viper.GetStringMapString("minio")

	//minioClient, err := minio.New(minioConf["endpoint"], minioConf["keyaccess"], minioConf["keysecret"], false)
	//if err != nil {
	//	panic("minio Client failed to init")
	//}



	// Generate target file list
	//rsync.GetFiles(data, conn, &filelist)

	// rsync.RequestFiles(conn, data, &filelist, minioClient, module, path)
	//go rsync.Downloader(data, &filelist)
	//fmt.Println(filelist)

}

func main() {
	//FIXME: Can't handle wrong module/path rsync://mirrors.tuna.tsinghua.edu.cn/linuxmint-packages/pool/romeo/libf/libfm/
	loadConfigIfExists()
	flag.Parse()
	args := flag.Args()
	if len(args) < 2 {
		fmt.Println("Usage: rsync-os [OPTION]... rsync://[USER@]HOST[:PORT]/SRC")
		return
	}
	startTime := time.Now()
	Socket(args[0], args[1])
	log.Println("Duration:", time.Since(startTime))
}
