package rsync

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"sort"
	"time"
)

/* Receiver:
1. Receive File list
2. Request files by sending files' index
3. Receive Files, pass the files to storage
*/
type Receiver struct {
	conn     *Conn
	module   string
	path     string
	seed     int32
	lVer     int32
	rVer     int32
	storage  FS
	callback Callback
}

// Return a filelist from remote
func (r *Receiver) RecvFileList() (FileList, map[int][]byte, error) {
	filelist := make(FileList, 0, 1*M)
	symlinks := make(map[int][]byte)
	for {
		flags, err := r.conn.ReadByte()
		if err != nil {
			return filelist, symlinks, err
		}

		if flags == FLIST_END {
			break
		}
		//fmt.Printf("[%d]\n", flags)

		lastIndex := len(filelist) - 1
		var partial, pathlen uint32 = 0, 0

		/*
		 * Read our filename.
		 * If we have FLIST_NAME_SAME, we inherit some of the last
		 * transmitted name.
		 * If we have FLIST_NAME_LONG, then the string length is greater
		 * than byte-size.
		 */
		if (flags & FLIST_NAME_SAME) != 0 {
			val, err := r.conn.ReadByte()
			if err != nil {
				return filelist, symlinks, err
			}
			partial = uint32(val)
			//fmt.Println("Partical", partial)
		}

		/* Get the (possibly-remaining) filename length. */
		if (flags & FLIST_NAME_LONG) != 0 {
			val, err := r.conn.ReadInt()
			if err != nil {
				return filelist, symlinks, err
			}
			pathlen = uint32(val) // can't use for rsync 31

		} else {
			val, err := r.conn.ReadByte()
			if err != nil {
				return filelist, symlinks, err
			}
			pathlen = uint32(val)
		}
		//fmt.Println("PathLen", pathlen)

		/* Allocate our full filename length. */
		/* FIXME: maximum pathname length. */
		// TODO: if pathlen + partical == 0
		// malloc len error?

		p := make([]byte, pathlen)
		_, err = io.ReadFull(r.conn, p)
		if err != nil {
			return filelist, symlinks, err
		}

		path := make([]byte, 0, partial+pathlen)
		/* If so, use last */
		if (flags & FLIST_NAME_SAME) != 0 { // FLIST_NAME_SAME
			last := filelist[lastIndex]
			path = append(path, last.Path[0:partial]...)
		}
		path = append(path, p...)
		//fmt.Println("Path ", string(path))

		size, err := r.conn.ReadVarint()
		if err != nil {
			return filelist, symlinks, err
		}
		//fmt.Println("Size ", size)

		/* Read the modification time. */
		var mtime int32
		if (flags & FLIST_TIME_SAME) == 0 {
			mtime, err = r.conn.ReadInt()
			if err != nil {
				return filelist, symlinks, err
			}
		} else {
			mtime = filelist[lastIndex].Mtime
		}
		//fmt.Println("MTIME ", mtime)

		/* Read the file mode. */
		var mode FileMode
		if (flags & FLIST_MODE_SAME) == 0 {
			val, err := r.conn.ReadInt()
			if err != nil {
				return filelist, symlinks, err
			}
			mode = FileMode(val)
		} else {
			mode = filelist[lastIndex].Mode
		}
		//fmt.Println("Mode", uint32(mode))

		// TODO: Sym link
		if ((mode & 32768) != 0) && ((mode & 8192) != 0) {
			sllen, err := r.conn.ReadInt()
			if err != nil {
				return filelist, symlinks, err
			}
			slink := make([]byte, sllen)
			_, err = io.ReadFull(r.conn, slink)
			symlinks[lastIndex+1] = slink
			if err != nil {
				return filelist, symlinks, errors.New("Failed to read symlink")
			}
			//fmt.Println("Symbolic Len:", len, "Content:", slink)
		}

		fmt.Println("@", string(path), mode, size, mtime)

		filelist = append(filelist, FileInfo{
			Path:  path,
			Size:  size,
			Mtime: mtime,
			Mode:  mode,
		})
	}

	// Sort the filelist lexicographically
	sort.Sort(filelist)

	return filelist, symlinks, nil
}

// Generator: handle files: if it's a regular file, send its index. Otherwise, put them to storage
func (r *Receiver) Generator(remoteList FileList, downloadList []int, symlinks map[int][]byte) error {
	emptyBlocks := make([]byte, 16) // 4 + 4 + 4 + 4 bytes, all bytes set to 0
	content := new(bytes.Buffer)

	for _, v := range downloadList {
		if remoteList[v].Mode.IsREG() {
			if err := r.conn.WriteInt(int32(v)); err != nil {
				log.Println("Failed to send index")
				return err
			}
			//fmt.Println("Request: ", string(remoteList[v].Path), uint32(remoteList[v].Mode))
			if _, err := r.conn.Write(emptyBlocks); err != nil {
				return err
			}
		} else {
			// TODO: Supports more file mode
			// EXPERIMENTAL
			// Handle folders & symbol links
			content.Reset()
			size := remoteList[v].Size
			if remoteList[v].Mode.IsLNK() {
				if _, err := content.Write(symlinks[v]); err != nil {
					return err
				}
				size = int64(content.Len())
			}

			if _, err := r.storage.Put(string(remoteList[v].Path), content, size, FileMetadata{
				Mtime: remoteList[v].Mtime,
				Mode:  remoteList[v].Mode,
			}); err != nil {
				return err
			}
		}
	}

	// Send -1 to finish, then start to download
	if err := r.conn.WriteInt(INDEX_END); err != nil {
		log.Println("Can't send INDEX_END")
		return err
	}
	log.Println("Request completed")

	startTime := time.Now()
	err := r.FileDownloader(remoteList[:])
	log.Println("Downloaded duration:", time.Since(startTime))
	return err
}

func (r *Receiver) FileDownloader(localList FileList) (err error) {

	rmd4 := make([]byte, 16)
	rfile := NewReceivingFile(r.conn)

	for {
		var index int32
		index, err = r.conn.ReadInt()
		if err != nil || index == INDEX_END { // -1 means the end of transfer files
			return
		}

		var count int32 /* block count */
		count, err = r.conn.ReadInt()
		if err != nil {
			return
		}

		var blen int32 /* block length */
		blen, err = r.conn.ReadInt()
		if err != nil {
			return
		}

		var clen int32 /* checksum length */
		clen, err = r.conn.ReadInt()
		if err != nil {
			return
		}

		var remainder int32 /* block remainder */
		remainder, err = r.conn.ReadInt()
		if err != nil {
			return err
		}

		path := localList[index].Path
		log.Println("Downloading:", string(path), count, blen, clen, remainder, localList[index].Size)

		fileSize := localList[index].Size

		// Create MD4
		//lmd4 := md4.New()
		//if err := binary.Write(lmd4, binary.LittleEndian, r.seed); err != nil {
		//	log.Println("Failed to compute md4")
		//}

		// Use a wrapper for incoming file content, and put the wrapper to storage
		rfile.Reset()
		var n int64
		n, err = r.storage.Put(string(path), rfile, int64(fileSize), FileMetadata{
			Mtime: localList[index].Mtime,
			Mode:  localList[index].Mode,
		})
		if err != nil {
			return err
		}

		// Remote MD4
		// TODO: compare computed MD4 with remote MD4
		_, err = io.ReadFull(r.conn, rmd4)
		if err != nil {
			return err
		}
		// Compare two MD4

		log.Printf("Successfully downloaded %s of size %d\n", path, n)
	}
}

// Clean up local files
func (r *Receiver) FileCleaner(localList FileList, deleteList []int) error {
	// Since file list was already sorted, we can iterate it in the reverse direction to traverse the file tree in post-order
	// Thus it always cleans sub-files firstly
	for i := len(deleteList) - 1; i >= 0; i-- {
		fname := string(localList[i].Path)
		err := r.storage.Delete(fname, localList[i].Mode)
		log.Println("Deleted:", fname)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *Receiver) FinalPhase() error {
	//go func() {
	//	ioerror, err := r.conn.ReadInt()
	//	log.Println(ioerror, err)
	//}()

	err := r.conn.WriteInt(INDEX_END)
	if err != nil {
		return err
	}
	return r.conn.WriteInt(INDEX_END)
}

func (r *Receiver) Sync() error {
	defer func() {
		log.Println("Task completed", r.conn.Close()) // TODO: How to handle errors from Close
	}()

	lfiles, err := r.storage.List()
	if err != nil {
		return err
	}
	//for _, v := range lfiles {
	//	fmt.Println("Local File:", string(v.Path), v.Mode, v.Mtime)
	//}

	rfiles, symlinks, err := r.RecvFileList()
	if err != nil {
		return err
	}
	log.Println("Remote files count:", len(rfiles))

	ioerr, err := r.conn.ReadInt()
	if err != nil {
		return nil
	}
	log.Println("IOERR", ioerr)

	newfiles, oldfiles := lfiles.Diff(rfiles)
	if len(newfiles) == 0 && len(oldfiles) == 0 {
		log.Println("There is nothing to do")
	}
	// fmt.Print(newfiles, oldfiles)
	newfiles = r.callback.OnRequest(rfiles, newfiles)

	if err := r.Generator(rfiles, newfiles, symlinks); err != nil {
		return err
	}

	oldfiles = r.callback.OnDelete(rfiles, oldfiles)
	if err := r.FileCleaner(lfiles[:], oldfiles[:]); err != nil {
		return err
	}
	if err := r.FinalPhase(); err != nil {
		return err
	}
	return nil
}
