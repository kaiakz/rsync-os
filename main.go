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
	"rsync-os/fldb"
	"rsync-os/rsync"
	"sort"

	"github.com/minio/minio-go/v6"
)

func Socket(uri string) {

	addr, module, path, _ := rsync.SplitURI(uri)

	fmt.Println(module, path)

	conn, err := net.Dial("tcp", addr)
	// tuna: mirrors.tuna.tsinghua.edu.cn 101.6.8.193:873

	if err != nil {
		// TODO
		panic("Network Error")
	}

	defer conn.Close()

	rsync.HandShake(conn, module, path)

	// fmt.Println(readInteger(conn))
	log.Println("HandShake OK")

	data := make(chan byte, 16*1024*1024)

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

	ppath := rsync.TrimPrepath(path)
	fldb.Save(&filelist, module, ppath)
	log.Println("File List Saved")

	return

	// Init the object storage
	// For test
	endpoint := "127.0.0.1:9000"
	accessKeyID := "minioadmin"
	secretAccessKey := "minioadmin"

	minioClient, err := minio.New(endpoint, accessKeyID, secretAccessKey, false)
	if err != nil {
		panic("Failed")
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
	Socket("rsync://mirrors.tuna.tsinghua.edu.cn/elvish")
	//Client("rsync://rsync.monitoring-plugins.org/plugins/")
	//Client("rsync://rsync.mirrors.ustc.edu.cn/repo/monitoring-plugins")
	//	rsync://rsync.monitoring-plugins.org/plugins/
}
