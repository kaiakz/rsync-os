package rsync

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strings"
)

// Header: '@RSYNCD: 31.0\n' + ? + '\n' + arguments + '\0'
// Header len 8		AUTHREQD: 18	"@RSYNCD: EXIT" 13		RSYNC_MODULE_LIST_QUERY "\n"
// See clienserver.c start_inband_exchange
func handshake() {
	
}

type FileInfo struct {
	Path string
	Size int64
	Mtime int32
	Mode int32
}

type FileList []FileInfo

func (I FileList) Len() int {
	return len(I)
}

func (I FileList) Less(i, j int) bool {
	if strings.Compare(I[i].Path, I[j].Path) == -1 {
		return true
	}
	return false
}

func (I FileList) Swap(i, j int) {
	I[i], I[j] = I[j], I[i]
}

// file list: ends with '\0'
func GetEntry(ds chan byte, filelist *FileList) error {

	flags := <- ds

	var partial, pathlen uint32 = 0, 0

	fmt.Println(flags)

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
		partial = uint32(GetByte(ds))
		fmt.Println("Partical", partial)
	}

	/* Get the (possibly-remaining) filename length. */
	if (0x40 & flags) != 0 {
		pathlen = uint32(GetInteger(ds)) // can't use for rsync 31

	} else {
		pathlen = uint32(<-ds)
	}
	fmt.Println("PathLen", pathlen)

	/* Allocate our full filename length. */
	/* FIXME: maximum pathname length. */
	// TODO: if pathlen + partical == 0
	// malloc len error?

	// last := (*filelist)[len(*filelist) - 1]	// FIXME


	p := make([]byte, pathlen)
	GetBytes(ds, p)
	var path string
	/* If so, use last */
	if (0x20 & flags) != 0 {	// FLIST_NAME_SAME
		last := (*filelist)[len(*filelist) - 1]
		path = last.Path[0: partial]
	}
	path += string(p)
	fmt.Println("Path ", path)

	size := GetVarint(ds)
	fmt.Println("Size ", size)

	/* Read the modification time. */
	var mtime int32
	if (flags & 0x80) == 0 {
		mtime = GetInteger(ds)

	} else {
		mtime = (*filelist)[len(*filelist) - 1].Mtime
	}
	fmt.Println("MTIME ", mtime)

	/* Read the file mode. */
	var mode int32
	if (flags & 0x02) == 0 {
		mode = GetInteger(ds)

	} else {
		mode = (*filelist)[len(*filelist) - 1].Mode
	}
	fmt.Println("Mode", uint32(mode))

	// FIXME: Sym link
	if ((mode & 32768) != 0) && ((mode & 8192) != 0) {
		len := uint32(GetInteger(ds))
		slink := make([]byte, len)
		GetBytes(ds, slink)
		fmt.Println("Symbolic Len", len, "CTX", slink)
	}

	*filelist = append(*filelist, FileInfo{
		Path:  path,
		Size:  size,
		Mtime: mtime,
		Mode:  mode,
	})

	return nil
}




func Generate(conn net.Conn, filelist *FileList) {
	// Compare all local files with file list, pick up the files that has different size, mtime
	// Those files are `basis files`
	var idx int32
	for i:=0; i < len(*filelist); i++ {
		if strings.Index((*filelist)[i].Path, "SRPMS/Packages/0/0ad-0.0.22-1.el7.src.rpm") != -1 {	// 95533 SRPMS/Packages/z/zanata-python-client-1.5.1-1.el7.src.rpm
			idx = int32(i)
			fmt.Println("Pick:", (*filelist)[i], idx)
			break
		}
	}
		//buf := new(bytes.Buffer)
		binary.Write(conn, binary.LittleEndian, idx)
		//fmt.Println(buf.Bytes())
		//conn.Write(buf.Bytes())
		//binary.Write(conn, binary.LittleEndian, uint16(0x8000))
		empty := []byte{0,0,0,0}

		// identifier, block count, block length, checksum length, block remainder, blocks(short+long)
		conn.Write(empty)
		binary.Write(conn, binary.LittleEndian, int32(32768))
		binary.Write(conn, binary.LittleEndian, int32(2))
		conn.Write(empty)
		// Empty checksum


		//buf.Reset()
		binary.Write(conn, binary.LittleEndian, int32(-1))
		//conn.Write(buf.Bytes())


}

// a block: [file id + block checksum + '\0']

func exchangeBlock() {
// Here we get a list stores old files
// Rolling Checksum & Hash value
// Loop until all file are updated, each time handle a file.
	// Send a empty signature block (no Rolling Checksum & Hash value)
	// Download the data blocks, and write them into a file
}