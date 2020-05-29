package rsync

import (
	"fmt"
	"io"
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


// file list: ends with '\0'
func GetEntry(ds chan byte, filelist *[]FileInfo) error {

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


func Generator() {
	// Compare all local files with file list, pick up the files that has different size, mtime
	// Those files are `basis files`
}

// a block: [file id + block checksum + '\0']

func exchangeBlock() {
// Here we get a list stores old files
// Rolling Checksum & Hash value
// Loop until all file are updated, each time handle a file.
	// Send a empty signature block (no Rolling Checksum & Hash value)
	// Download the data blocks, and write them into a file
}