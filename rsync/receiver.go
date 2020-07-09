package rsync

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"strings"

	"github.com/minio/minio-go/v6"
)

type SocketConn struct {
	Conn    net.Conn
	DemuxIn chan byte
	CksSeed int32
	// Options
}

// Header: '@RSYNCD: 31.0\n' + ? + '\n' + arguments + '\0'
// Header len 8		AUTHREQD: 18	"@RSYNCD: EXIT" 13		RSYNC_MODULE_LIST_QUERY "\n"

// See clienserver.c start_inband_exchange
func (c *SocketConn) HandShake(module string, path string) {
	// send my version
	// send("@RSYNCD: 31.0\n");
	c.Conn.Write([]byte("@RSYNCD: 27.0\n"))

	// receive server's protocol version and seed
	versionStr, _ := ReadLine(c.Conn)

	// recv(version)
	var remoteProtocol, remoteProtocolSub int
	fmt.Sscanf(versionStr, "@RSYNCD: %d.%d", remoteProtocol, remoteProtocolSub)
	log.Println(versionStr)

	// send mod name
	// send("Foo\n")
	c.Conn.Write([]byte(module))
	c.Conn.Write([]byte("\n"))

	for {
		// Wait for '@RSYNCD: OK': until \n, then add \0
		res, _ := ReadLine(c.Conn)
		log.Print(res)
		if strings.Contains(res, "@RSYNCD: OK") {
			break
		}
	}

	c.SendArgs(module, path)

	// read int32 as seed
	c.CksSeed = ReadInteger(c.Conn)
	log.Println("SEED", c.CksSeed)

	c.SendEmptyExclusion()
}

func (c *SocketConn) SendArgs(module string, path string) {
	// send parameters list
	//conn.Write([]byte("--server\n--sender\n-g\n-l\n-o\n-p\n-D\n-r\n-t\n.\nepel/7/SRPMS\n\n"))
	//conn.Write([]byte("--server\n--sender\n-l\n-p\n-r\n-t\n.\nepel/7/SRPMS\n\n"))	// without gid, uid, mdev
	args := new(bytes.Buffer)
	args.Write([]byte("--server\n--sender\n-l\n-p\n-r\n-t\n.\n"))
	args.Write([]byte(module))
	args.Write([]byte(path))
	args.Write([]byte("\n\n"))
	c.Conn.Write(args.Bytes())
}

func (c *SocketConn) ListOnly(module string, path string) {
	c.Conn.Write([]byte("@RSYNCD: 27.0\n"))
	versionStr, _ := ReadLine(c.Conn)
	log.Println(versionStr)

	c.Conn.Write([]byte(module))
	c.Conn.Write([]byte("\n"))
	for {
		// Wait for '@RSYNCD: OK': until \n, then add \0
		res, _ := ReadLine(c.Conn)
		log.Print(res)
		if strings.Contains(res, "@RSYNCD: OK") {
			break
		}
	}
	args := new(bytes.Buffer)
	args.Write([]byte("--server\n--sender\n-l\n-p\n-r\n-t\n.\n"))
	args.Write([]byte(module))
	args.Write([]byte(path))
	args.Write([]byte("\n\n"))

	c.Conn.Write(args.Bytes())

	seed := ReadInteger(c.Conn)
	log.Println("SEED: ", seed)

	c.Conn.Write(make([]byte, 4))
}

func (c *SocketConn) SendEmptyExclusion() {
	// send filter_list, empty is 32-bit zero
	c.Conn.Write([]byte("\x00\x00\x00\x00"))
}

type FileInfo struct {
	Path  []byte
	Size  int64
	Mtime int32
	Mode  int32
}

type FileList []FileInfo

func (I FileList) Len() int {
	return len(I)
}

func (I FileList) Less(i, j int) bool {
	if bytes.Compare(I[i].Path, I[j].Path) == -1 {
		return true
	}
	return false
}

func (I FileList) Swap(i, j int) {
	I[i], I[j] = I[j], I[i]
}

// file list: ends with '\0'
func GetFileList(data chan byte, filelist *FileList) error {

	flags := <-data

	var partial, pathlen uint32 = 0, 0

	//fmt.Printf("[%d]\n", flags)

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
	if (0x20 & flags) != 0 {
		partial = uint32(GetByte(data))
		//fmt.Println("Partical", partial)
	}

	/* Get the (possibly-remaining) filename length. */
	if (0x40 & flags) != 0 {
		pathlen = uint32(GetInteger(data)) // can't use for rsync 31

	} else {
		pathlen = uint32(<-data)
	}
	//fmt.Println("PathLen", pathlen)

	/* Allocate our full filename length. */
	/* FIXME: maximum pathname length. */
	// TODO: if pathlen + partical == 0
	// malloc len error?

	// last := (*filelist)[len(*filelist) - 1]	// FIXME

	p := make([]byte, pathlen)
	GetBytes(data, p)
	path := make([]byte, 0, pathlen)
	/* If so, use last */
	if (0x20 & flags) != 0 { // FLIST_NAME_SAME
		last := (*filelist)[len(*filelist)-1]
		path = append(path, last.Path[0:partial]...)
	}
	path = append(path, p...)
	//path += string(p)
	//fmt.Println("Path ", string(path))

	size := GetVarint(data)
	//fmt.Println("Size ", size)

	/* Read the modification time. */
	var mtime int32
	if (flags & 0x80) == 0 {
		mtime = GetInteger(data)

	} else {
		mtime = (*filelist)[len(*filelist)-1].Mtime
	}
	//fmt.Println("MTIME ", mtime)

	/* Read the file mode. */
	var mode int32
	if (flags & 0x02) == 0 {
		mode = GetInteger(data)

	} else {
		mode = (*filelist)[len(*filelist)-1].Mode
	}
	//fmt.Println("Mode", uint32(mode))

	// FIXME: Sym link
	if ((mode & 32768) != 0) && ((mode & 8192) != 0) {
		sllen := uint32(GetInteger(data))
		slink := make([]byte, sllen)
		GetBytes(data, slink)
		//fmt.Println("Symbolic Len", len, "CTX", slink)
	}

	*filelist = append(*filelist, FileInfo{
		Path:  path,
		Size:  size,
		Mtime: mtime,
		Mode:  mode,
	})

	return nil
}

func (c *SocketConn) GetFL() (FileList, error) {
	filelist := make(FileList, 0, 4096)
	// recv_file_list
	for {
		if GetFileList(c.DemuxIn, &filelist) == io.EOF {
			break
		}
	}
	log.Println("File List Received, total size is", len(filelist))
	return filelist[:], nil
}

/* Generator */

func RequestFiles(conn net.Conn, data chan byte, filelist *FileList, os *minio.Client, module string, prepath string) {
	empty := make([]byte, 16) // 4 + 4 + 4 + 4 bytes
	for i := 0; i < len(*filelist); i++ {
		// TODO: Supports more file mode
		if (*filelist)[i].Mode == 0100644 {
			binary.Write(conn, binary.LittleEndian, int32(i))

			fmt.Println((*filelist)[i].Path)
			conn.Write(empty)

		}

	}
	log.Println("Request completed")
	// Finish
	binary.Write(conn, binary.LittleEndian, int32(-1))
	Downloader(data, filelist, os, module, prepath)
}

func RequestAFile(conn net.Conn, target string, filelist *FileList) {
	// Compare all local files with file list, pick up the files that has different size, mtime
	// Those files are `basis files`
	var idx int32

	// TODO: Supports multi files
	// For test: here we request a file
	for i := 0; i < len(*filelist); i++ {
		if bytes.Contains((*filelist)[i].Path, []byte(target)) {
			idx = int32(i)
			log.Println("Pick:", (*filelist)[i], idx)
			// identifier
			binary.Write(conn, binary.LittleEndian, idx)
			// block count, block length(default is 32768?), checksum length(default is 2?), block remainder, blocks(short+long)
			// Just let them be empty(zero)
			empty := make([]byte, 16) // 4 + 4 + 4 + 4 bytes
			conn.Write(empty)         // ENDIAN?
			//conn.Write(empty)
			//binary.Write(conn, binary.LittleEndian, int32(0))	// 32768
			//binary.Write(conn, binary.LittleEndian, int32(0))	// 2
			//conn.Write(empty)
			// Empty checksum
			break
		}
	}
	// Finish
	binary.Write(conn, binary.LittleEndian, int32(-1))
}

func Downloader(data chan byte, filelist *FileList, os *minio.Client, module string, prepath string) {

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
		n, err := os.PutObject(module, objectName, buf, int64(buf.Len()), minio.PutObjectOptions{})
		if err != nil {
			log.Fatalln(err)
		}

		log.Printf("Successfully uploaded %s of size %d\n", path, n)

		// Remote MD4
		rmd4 := make([]byte, 16)
		GetBytes(data, rmd4)
		fmt.Println("OK:", rmd4)
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

func FinalPhase(conn net.Conn, data chan byte) {

	binary.Write(conn, binary.LittleEndian, int32(-1))
	ioerror := GetInteger(data)
	fmt.Println(ioerror)

}
