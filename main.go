// Send @RSYNCD x.x\n
// Send modname\n
// Send arugment with mod list\0	filter list write(0)    \n
// handshake
// batch seed
// Recv file list
//

package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"github.com/spf13/viper"
	"io"
	"log"
	"net"
	"rsync-os/fldb"
	"rsync-os/rsync"
	"rsync-os/storage"
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

	receiver := &rsync.SocketConn{
		RawConn: conn,
		DemuxIn: make(chan byte, 16*1024*1024),
	}

	if receiver.HandShake(module, path) != nil {
		log.Println("HandShake Failed")
		return
	}

	// fmt.Println(readInteger(conn))
	log.Println("HandShake OK")

	// Start De-Multiplexing
	go rsync.DeMuxChan(conn, receiver.DemuxIn)

	filelist := make(rsync.FileList, 0, 1024 * 1024)
	// recv_file_list
	for {
		if rsync.GetFileList(receiver.DemuxIn, &filelist) == io.EOF {
			break
		}
	}
	log.Println("File List Received, total size is", len(filelist))

	ioerr := rsync.GetInteger(receiver.DemuxIn)
	log.Println("IOERR", ioerr)

	// Sort the filelist lexicographically
	sort.Sort(filelist)

	ppath := rsync.TrimPrepath(path)
	//fldb.Snapshot(filelist[:], module, ppath)
	if viper.GetStringMapString(dest) == nil {
		log.Println("Lack of ", dest)
		return
	}
	dbconf := viper.GetStringMapString(dest + ".boltdb")
	cache := fldb.Open(dbconf["path"], []byte(module), []byte(ppath))
	if cache == nil {
		// TODO
		log.Println("Failed to init cache")
		return
	}

	// Diff
	downloadList, deleteList := cache.Diff(filelist[:])

	// Update file list && start downloading
	// Init the object storage
	minioConf := viper.GetStringMapString(dest)
	log.Println(minioConf)
	if len(minioConf) == 0 {
		// test
		log.Println("Failed to read config about ", dest)
		return
	}
	osClient := storage.NewMinio(module, minioConf["endpoint"], minioConf["keyaccess"], minioConf["keysecret"], false)
	if osClient == nil {
		log.Println("object storage failed to init")
		return
	}


	if len(downloadList) == 0 && len(deleteList) == 0 {
		// Send -1 to finish, then start to download
		if binary.Write(receiver.RawConn, binary.LittleEndian, rsync.INDEX_END) != nil {
			log.Println("Can't send INDEX_END")
			return
		}
		log.Println("There is nothing to do")
	} else {
		// Start downloading
		if receiver.RequestFiles(filelist[:], downloadList[:], osClient, ppath) != nil {
			log.Println("Failed to request file")
			return
		}

		// Delete old file
		if osClient.DeleteAll([]byte(ppath), deleteList[:]) != nil {
			log.Println("Failed to delete old files")
		}

		// Update cache
		if cache.Update(filelist[:], downloadList[:], deleteList[:]) != nil {
			log.Println("Failed to Update")
		}
		log.Println("Updated cache")
	}

	if receiver.FinalPhase() != nil {
		log.Println("Failed to say goodbye")
	}

	// TODO: Need to close fldb & network
	defer cache.Close()
	return

}

func main() {
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
