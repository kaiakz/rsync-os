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
	"github.com/minio/minio-go/v6"
	"github.com/spf13/viper"
	"io"
	"log"
	"net"
	"rsync-os/fldb"
	"rsync-os/rsync"
	"sort"
	"time"
)

func Socket(uri string) {

	addr, module, path, _ := rsync.SplitURI(uri)

	fmt.Println(module, path)

	conn, err := net.Dial("tcp", addr)
	//rAddr, err := net.ResolveTCPAddr("tcp", addr)
	//tconn, err := net.DialTCP("tcp", nil, rAddr)
	//tconn.SetReadBuffer(4096)
	//tconn.SetWriteBuffer(4096)

	if err != nil {
		// TODO
		panic("Network Error")
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
	dbconf := viper.GetStringMapString("boltdb")
	cache := fldb.Open(dbconf["path"], []byte(module), []byte(ppath))
	if cache == nil {
		// TODO
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
	minioConf := viper.GetStringMapString("minio")

	minioClient, err := minio.New(minioConf["endpoint"], minioConf["accesskeyid"], minioConf["secretaccesskey"], false)
	if err != nil {
		panic("minio Client failed to init")
	}

	// Create a bucket for the module
	err = minioClient.MakeBucket(module, "us-east-1")
	if err != nil {
		// Check to see if we already own this bucket (which happens if you run this twice)
		exists, errBucketExists := minioClient.BucketExists(module)
		if errBucketExists == nil && exists {
			log.Printf("We already own %s\n", module)
		} else {
			log.Fatalln(err)
		}
	} else {
		log.Printf("Successfully created %s\n", module)
	}

	// Generate target file list
	//rsync.RequestAFile(conn, "libnemo-extension1_1.8.1+maya_amd64.deb", &filelist)
	//rsync.GetFiles(data, conn, &filelist)

	// rsync.RequestFiles(conn, data, &filelist, minioClient, module, path)
	//go rsync.Downloader(data, &filelist)
	//fmt.Println(filelist)

}

func main() {
	//FIXME: Can't handle wrong module/path rsync://mirrors.tuna.tsinghua.edu.cn/linuxmint-packages/pool/romeo/libf/libfm/
	// rsync://rsync.monitoring-plugins.org/plugins/
	// rsync://rsync.mirrors.ustc.edu.cn/repo/monitoring-plugins
	// rsync://rsync.monitoring-plugins.org/plugins/
	// rsync://mirrors.tuna.tsinghua.edu.cn/ubuntu
	// rsync://mirrors.tuna.tsinghua.edu.cn/elvish
	LoadYAMLConfig()
	startTime := time.Now()
	Socket("rsync://mirrors.tuna.tsinghua.edu.cn/ubuntu-ports")
	log.Println("Duration:", time.Since(startTime))
}
