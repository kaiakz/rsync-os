package rsync

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/kaiakz/ubuffer"
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
func (conn *SocketConn) HandShake(module string, path string) error {
	var err error = nil

	// send my version
	_, err = conn.RawConn.Write([]byte(RSYNC_VERSION))
	if err != nil {
		return err
	}

	// receive server's protocol version and seed
	versionStr, _ := ReadLine(conn.RawConn)

	// recv(version)
	var remoteProtocol, remoteProtocolSub int
	_, err = fmt.Sscanf(versionStr, "@RSYNCD: %d.%d", remoteProtocol, remoteProtocolSub)
	if err != nil {
		// FIXME: (panic)type not a pointer: int
		//panic(err)
	}
	log.Println(versionStr)

	// send mod name
	_, err = conn.RawConn.Write([]byte(module))
	if err != nil {
		return err
	}
	_, err = conn.RawConn.Write([]byte("\n"))
	if err != nil {
		return err
	}

	for {
		// Wait for '@RSYNCD: OK': until \n, then add \0
		res, _ := ReadLine(conn.RawConn)
		log.Print(res)
		if strings.Contains(res, RSYNCD_OK) {
			break
		}
	}

	err = conn.SendArgs(module, path)
	if err != nil {
		return err
	}

	// read int32 as seed
	conn.CksSeed = ReadInteger(conn.RawConn)
	log.Println("SEED", conn.CksSeed)

	return conn.SendEmptyExclusion()
}

func (conn *SocketConn) SendArgs(module string, path string) error {
	// send parameters list
	// Sample "--server\n--sender\n-g\n-l\n-o\n-p\n-D\n-r\n-t\n.\nepel/7/SRPMS\n\n"
	args := new(bytes.Buffer)
	args.Write([]byte(SAMPLE_ARGS))
	args.Write([]byte(module))
	args.Write([]byte(path))
	args.Write([]byte("\n\n"))
	_, err := conn.RawConn.Write(args.Bytes())
	return err
}

func (conn *SocketConn) ListOnly(module string, path string) error {
	var err error
	_, err = conn.RawConn.Write([]byte(RSYNC_VERSION))
	if err != nil {
		return err
	}

	versionStr, _ := ReadLine(conn.RawConn)
	log.Println(versionStr)

	_, err = conn.RawConn.Write([]byte(module))
	if err != nil {
		return err
	}

	_, err = conn.RawConn.Write([]byte("\n"))
	if err != nil {
		return err
	}

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
	_, err = conn.RawConn.Write(args.Bytes())
	if err != nil {
		return err
	}

	seed := ReadInteger(conn.RawConn)
	log.Println("SEED: ", seed)

	_, err = conn.RawConn.Write(make([]byte, 4))
	if err != nil {
		return err
	}

	return conn.FinalPhase()
}

func (conn *SocketConn) SendEmptyExclusion() error {
	// send filter_list, empty is 32-bit zero
	//_, err := conn.RawConn.Write([]byte("\x00\x00\x00\x00"))
	return binary.Write(conn.RawConn, binary.LittleEndian, EMPTY_EXCLUSION)
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

func (conn *SocketConn) RequestFiles(filelist FileList, downloadList []int, osClient IO, prepath string) error {
	emptyBlocks := make([]byte, 16) // 4 + 4 + 4 + 4 bytes, all bytes set to 0
	var err error = nil
	for _, v := range downloadList {
		// TODO: Supports more file mode
		if filelist[v].Mode == 0100644 || filelist[v].Mode == 0100755 {
			err = binary.Write(conn.RawConn, binary.LittleEndian, int32(v))
			if err != nil {
				log.Println("Failed to send index")
				return err
			}

			fmt.Println("Request: ", string(filelist[v].Path), uint32(filelist[v].Mode))
			_, err := conn.RawConn.Write(emptyBlocks)
			if err != nil {
				return err
			}
		}

		/* EXPERIMENTAL else {
			// Handle folders & symbol links
			emptyCtx := new(bytes.Buffer)
			osClient.Write(prepath+string((*filelist)[i].Path), emptyCtx, int64(emptyCtx.Len()), FileMetadata{
				Mtime: (*filelist)[i].Mtime,
				Mode: (*filelist)[i].Mode,
			})
		}*/
	}

	// Send -1 to finish, then start to download
	err = binary.Write(conn.RawConn, binary.LittleEndian, INDEX_END)
	if err != nil {
		log.Println("Can't send INDEX_END")
		return err
	}
	log.Println("Request completed")

	startTime := time.Now()
	Downloader(conn.DemuxIn, filelist[:], osClient, prepath)
	log.Println("Downloaded duration:", time.Since(startTime))
	return nil
}

// TODO: It is better to update files in goroutine
func Downloader(data chan byte, filelist FileList, osClient IO, prepath string) {

	ppath := []byte(prepath)

	for {
		index := GetInteger(data)
		if index == -1 {
			return
		}
		fmt.Println("INDEX:", index)
		path := filelist[index].Path
		count := GetInteger(data)     /* block count */
		blen := GetInteger(data)      /* block length */
		clen := GetInteger(data)      /* checksum length */
		remainder := GetInteger(data) /* block remainder */

		log.Println("Downloading:", string(path), count, blen, clen, remainder, filelist[index].Size)

		// If the file is too big to store in memory, creates a temporary file in the directory 'tmp'
		buffer := ubuffer.NewBuffer(filelist[index].Size)
		downloadeSize := 0
		bufwriter := bufio.NewWriter(buffer)
		for {
			token := GetInteger(data)
			log.Println("TOKEN", token)
			if token == 0 {
				break
			} else if token < 0 {
				panic("Does not support block checksum")
				// Reference
			} else {
				ctx := make([]byte, token)		// FIXME: memory leak?
				GetBytes(data, ctx)
				downloadeSize += int(token)
				log.Println("Downloaded:", downloadeSize, "byte")
				_, err := bufwriter.Write(ctx)
				if err != nil {
					panic(err)
				}
			}
		}
		if bufwriter.Flush() != nil {
			panic("Failed to flush buffer")
		}
		// Put file to object storage
		objectName := string(append(ppath[:], path[:]...))	// prefix + path

		var (
			n int64
			err error
		)
		n, err = buffer.Seek(0, io.SeekStart)

		n, err = osClient.Write(objectName, buffer, int64(downloadeSize), FileMetadata{
			Mtime: filelist[index].Mtime,
			Mode:  filelist[index].Mode,
		})
		if err != nil {
			panic(err)
		}

		if buffer.Finalize() != nil {
			panic("Buffer can't be finalized")
		}

		log.Printf("Successfully uploaded %s of size %d\n", path, n)

		// Remote MD4
		// TODO: compare computed MD4 with remote MD4
		rmd4 := make([]byte, 16)
		GetBytes(data, rmd4)
		fmt.Println("Remote MD4:", rmd4)

		//lmd4 := md4.New()
		//lmd4.Write(buffer.Bytes())
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

func (conn *SocketConn) FinalPhase() error {
	go func() {
		ioerror := GetInteger(conn.DemuxIn)
		log.Println(ioerror)
	}()

	err := binary.Write(conn.RawConn, binary.LittleEndian, INDEX_END)
	if err != nil {
		return err
	}
	return binary.Write(conn.RawConn, binary.LittleEndian, INDEX_END)
}
