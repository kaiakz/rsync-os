package rsync

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

type SocketConn struct {
	RawConn net.Conn
	DemuxIn chan byte
	CksSeed int32
	// Options
}

// Header: '@RSYNCD: 31.0\n' + ? + '\n' + arguments + '\0'
// Header len 8		AUTHREQD: 18	"@RSYNCD: EXIT" 13		RSYNC_MODULE_LIST_QUERY "\n"

// See clienserver.c start_inband_exchange
func (conn *SocketConn) HandShake(module string, path string) {
	// send my version
	conn.RawConn.Write([]byte(RSYNC_VERSION))

	// receive server's protocol version and seed
	versionStr, _ := ReadLine(conn.RawConn)

	// recv(version)
	var remoteProtocol, remoteProtocolSub int
	fmt.Sscanf(versionStr, "@RSYNCD: %d.%d", remoteProtocol, remoteProtocolSub)
	log.Println(versionStr)

	// send mod name
	conn.RawConn.Write([]byte(module))
	conn.RawConn.Write([]byte("\n"))

	for {
		// Wait for '@RSYNCD: OK': until \n, then add \0
		res, _ := ReadLine(conn.RawConn)
		log.Print(res)
		if strings.Contains(res, RSYNCD_OK) {
			break
		}
	}

	conn.SendArgs(module, path)

	// read int32 as seed
	conn.CksSeed = ReadInteger(conn.RawConn)
	log.Println("SEED", conn.CksSeed)

	conn.SendEmptyExclusion()
}

func (conn *SocketConn) SendArgs(module string, path string) {
	// send parameters list
	// Sample "--server\n--sender\n-g\n-l\n-o\n-p\n-D\n-r\n-t\n.\nepel/7/SRPMS\n\n"
	args := new(bytes.Buffer)
	args.Write([]byte(SAMPLE_ARGS))
	args.Write([]byte(module))
	args.Write([]byte(path))
	args.Write([]byte("\n\n"))
	conn.RawConn.Write(args.Bytes())
}

func (conn *SocketConn) ListOnly(module string, path string) {
	conn.RawConn.Write([]byte("@RSYNCD: 27.0\n"))
	versionStr, _ := ReadLine(conn.RawConn)
	log.Println(versionStr)

	conn.RawConn.Write([]byte(module))
	conn.RawConn.Write([]byte("\n"))
	for {
		res, _ := ReadLine(conn.RawConn)
		log.Print(res)
		if strings.Contains(res, "@RSYNCD: OK") {
			break
		}
	}

	args := new(bytes.Buffer)
	args.Write([]byte(SAMPLE_LIST_ARGS))
	args.Write([]byte(module))
	args.Write([]byte(path))
	args.Write([]byte("\n\n"))
	conn.RawConn.Write(args.Bytes())

	seed := ReadInteger(conn.RawConn)
	log.Println("SEED: ", seed)

	conn.RawConn.Write(make([]byte, 4))

	conn.FinalPhase()

}

func (conn *SocketConn) SendEmptyExclusion() {
	// send filter_list, empty is 32-bit zero
	conn.RawConn.Write([]byte("\x00\x00\x00\x00"))
}

// file list: ends with '\0'
func GetFileList(data chan byte, filelist *FileList) error {

	flags := <-data

	var partial, pathlen uint32 = 0, 0

	//fmt.Printf("[%d]\n", flags)

	// TODO: refactor
	if flags == 0 {
		return io.EOF
	}

	/*
	 * Read our filename.
	 * If we have FLIST_NAME_SAME, we inherit some of the last
	 * transmitted name.
	 * If we have FLIST_NAME_LONG, then the string length is greater
	 * than byte-size.
	 */
	if (flags & FLIST_NAME_SAME) != 0 {
		partial = uint32(GetByte(data))
		//fmt.Println("Partical", partial)
	}

	/* Get the (possibly-remaining) filename length. */
	if (flags & FLIST_NAME_LONG) != 0 {
		pathlen = uint32(GetInteger(data)) // can't use for rsync 31

	} else {
		pathlen = uint32(<-data)
	}
	//fmt.Println("PathLen", pathlen)

	/* Allocate our full filename length. */
	/* FIXME: maximum pathname length. */
	// TODO: if pathlen + partical == 0
	// malloc len error?

	p := make([]byte, pathlen)
	GetBytes(data, p)
	path := make([]byte, 0, pathlen)
	/* If so, use last */
	if (flags & FLIST_NAME_SAME) != 0 { // FLIST_NAME_SAME
		last := (*filelist)[len(*filelist)-1]
		path = append(path, last.Path[0:partial]...)
	}
	path = append(path, p...)
	//fmt.Println("Path ", string(path))

	size := GetVarint(data)
	//fmt.Println("Size ", size)

	/* Read the modification time. */
	var mtime int32
	if (flags & FLIST_TIME_SAME) == 0 {
		mtime = GetInteger(data)

	} else {
		mtime = (*filelist)[len(*filelist)-1].Mtime
	}
	//fmt.Println("MTIME ", mtime)

	/* Read the file mode. */
	var mode os.FileMode
	if (flags & FLIST_MODE_SAME) == 0 {
		mode = GetFileMode(data)

	} else {
		mode = (*filelist)[len(*filelist)-1].Mode
	}
	//fmt.Println("Mode", uint32(mode))

	// FIXME: Sym link
	if ((mode & 32768) != 0) && ((mode & 8192) != 0) {
		sllen := uint32(GetInteger(data))
		slink := make([]byte, sllen)
		GetBytes(data, slink)
		//fmt.Println("Symbolic Len:", len, "Content:", slink)
	}

	*filelist = append(*filelist, FileInfo{
		Path:  path,
		Size:  size,
		Mtime: mtime,
		Mode:  mode,
	})

	return nil
}

func (conn *SocketConn) GetFL() (FileList, error) {
	filelist := make(FileList, 0, 4096)
	// recv_file_list
	for {
		if GetFileList(conn.DemuxIn, &filelist) == io.EOF {
			break
		}
	}
	log.Println("File List Received, total size is", len(filelist))
	return filelist[:], nil
}

/* Generator */

func (conn *SocketConn) RequestFiles(filelist *FileList, osClient IO, prepath string) {
	empty := make([]byte, 16) // 4 + 4 + 4 + 4 bytes, all bytes set to 0
	for i := 0; i < len(*filelist); i++ {
		// TODO: Supports more file mode
		if (*filelist)[i].Mode.IsRegular() {
			if binary.Write(conn.RawConn, binary.LittleEndian, int32(i)) != nil {
				panic("Failed to send index")
			}

			fmt.Println((*filelist)[i].Path)
			conn.RawConn.Write(empty)
		}

	}
	log.Println("Request completed")
	// Send -1 to finish, then start to download
	binary.Write(conn.RawConn, binary.LittleEndian, INDEX_END)


	startTime := time.Now()
	Downloader(conn.DemuxIn, filelist, osClient, prepath)
	log.Println("Downloaded duration:", time.Since(startTime))
}

// TODO: It is better to update files in goroutine
func Downloader(data chan byte, filelist *FileList, osClient IO, prepath string) {

	ppath := []byte(TrimPrepath(prepath))

	for {
		index := GetInteger(data)
		if index == -1 {
			return
		}
		fmt.Println("INDEX:", index)
		path := (*filelist)[index].Path
		count := GetInteger(data)     /* block count */
		blen := GetInteger(data)      /* block length */
		clen := GetInteger(data)      /* checksum length */
		remainder := GetInteger(data) /* block remainder */

		log.Println(path, count, blen, clen, remainder, (*filelist)[index].Size)
		buf := new(bytes.Buffer)
		for {
			token := GetInteger(data)
			log.Println("TOKEN", token)
			if token == 0 {
				break
			} else if token < 0 {
				panic("Does not support block checksum")
				// Reference
			} else {
				ctx := make([]byte, token)
				GetBytes(data, ctx)
				log.Println("Buff size:", buf.Len())
				buf.Write(ctx)
			}
		}

		// Put file to object storage
		objectName := string(append(ppath[:], path[:]...))	// prefix + path

		n, err := osClient.Write(objectName, buf, int64(buf.Len()), FileMetadata{
			Mtime: (*filelist)[index].Mtime,
			Mode:  (*filelist)[index].Mode,
		})
		if err != nil {
			log.Fatalln(err)
		}

		log.Printf("Successfully uploaded %s of size %d\n", path, n)

		// Remote MD4
		// TODO: compare computed MD4 with remote MD4
		rmd4 := make([]byte, 16)
		GetBytes(data, rmd4)
		fmt.Println("Remote MD4:", rmd4)

		//lmd4 := md4.New()
		//lmd4.Write(buf.Bytes())
		//if bytes.Compare(rmd4, lmd4.Sum(nil)) == 0 {
		//
		//}
	}
}

// a block: [file id + block checksum + '\0']
func exchangeBlock() {
	// Here we get a list stores old files
	// Rolling Checksum & Hash value
	// Loop until all file are updated, each time handle a file.
	// Send a empty signature block (no Rolling Checksum & Hash value)
	// Download the data blocks, and write them into a file
}

func (conn *SocketConn) FinalPhase() {
	ioerror := GetInteger(conn.DemuxIn)
	fmt.Println(ioerror)

	binary.Write(conn.RawConn, binary.LittleEndian, INDEX_END)
	binary.Write(conn.RawConn, binary.LittleEndian, INDEX_END)
}
