package rsync

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"github.com/kaiakz/ubuffer"
	"io"
	"log"
	"net"
	"os"
	"sort"
	"strings"
	"time"
)

/* Receiver:
1. Receive File list
2. Request files by sending files' index
3. Receive Files, pass the files to storage
*/
type Receiver struct {
	conn    *Conn
	module  string
	path    string
	seed    int32
	storage FS
}

func NewSocket(address string, module string, path string) (*Receiver, error) {
	skt, err := net.Dial("tcp", address)
	if err != nil {
		return nil, err
	}

	conn := new(Conn)
	conn.reader = skt
	conn.writer = skt

	/* HandShake by socket */
	// send my version
	_, err = conn.Write([]byte(RSYNC_VERSION))
	if err != nil {
		return nil, err
	}

	// receive server's protocol version and seed
	versionStr, _ := readLine(conn)

	// recv(version)
	var remoteProtocol, remoteProtocolSub int
	_, err = fmt.Sscanf(versionStr, "@RSYNCD: %d.%d", remoteProtocol, remoteProtocolSub)
	if err != nil {
		// FIXME: (panic)type not a pointer: int
		//panic(err)
	}
	log.Println(versionStr)

	buf := new(bytes.Buffer)

	// send mod name
	buf.WriteString(module)
	buf.WriteByte('\n')
	_, err = conn.Write(buf.Bytes())
	if err != nil {
		return nil, err
	}
	buf.Reset()

	// Wait for '@RSYNCD: OK'
	for {
		res, err := readLine(conn)
		if err != nil {
			return nil, err
		}
		log.Print(res)
		if strings.Contains(res, RSYNCD_OK) {
			break
		}
	}

	// Send arguments
	buf.Write([]byte(SAMPLE_ARGS))
	buf.Write([]byte(module))
	buf.Write([]byte(path))
	buf.Write([]byte("\n\n"))
	_, err = conn.Write(buf.Bytes())
	if err != nil {
		return nil, err
	}

	// read int32 as seed
	seed, err := conn.ReadInt()
	if err != nil {
		return nil, err
	}
	log.Println("SEED", seed)

	// HandShake OK
	// Begin to demux
	conn.reader = NewMuxReader(conn.reader)

	return &Receiver{
		conn:   conn,
		module: module,
		path:   path,
		seed:   seed,
	}, nil
}

func NewSsh(address string, module string, path string) (*Receiver, error) {
	return nil, nil
}

func (r *Receiver) BuildArgs() string {
	return ""
}

// DeMux was started here
func (r *Receiver) StartMuxIn() {
	r.conn.reader = NewMuxReader(r.conn.reader)
}

func (r *Receiver) SendExclusions() error {
	// Send exclusion
	return r.conn.WriteInt(EMPTY_EXCLUSION)
}

func (r *Receiver) GetFileList() (FileList, error) {
	filelist := make(FileList, 0, 1 *M)
	for {
		flags, _ := r.conn.ReadByte()

		var partial, pathlen uint32 = 0, 0

		//fmt.Printf("[%d]\n", flags)

		// TODO: refactor
		if flags == 0 {
			break
		}

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
				return filelist[:], err
			}
			partial = uint32(val)
			//fmt.Println("Partical", partial)
		}

		/* Get the (possibly-remaining) filename length. */
		if (flags & FLIST_NAME_LONG) != 0 {
			val, err := r.conn.ReadInt()
			if err != nil {
				return filelist[:], err
			}
			pathlen = uint32(val) // can't use for rsync 31

		} else {
			val, err := r.conn.ReadByte()
			if err != nil {
				return filelist[:], err
			}
			pathlen = uint32(val)
		}
		//fmt.Println("PathLen", pathlen)

		/* Allocate our full filename length. */
		/* FIXME: maximum pathname length. */
		// TODO: if pathlen + partical == 0
		// malloc len error?

		p := make([]byte, pathlen)
		_, err := r.conn.Read(p)
		if err != nil {
			panic("Failed to read path")
		}

		path := make([]byte, 0, partial + pathlen)
		/* If so, use last */
		if (flags & FLIST_NAME_SAME) != 0 { // FLIST_NAME_SAME
			last := filelist[len(filelist)-1]
			path = append(path, last.Path[0:partial]...)
		}
		path = append(path, p...)
		//fmt.Println("Path ", string(path))

		size, err := r.conn.ReadVarint()
		if err != nil {
			return filelist[:], err
		}
		//fmt.Println("Size ", size)

		/* Read the modification time. */
		var mtime int32
		if (flags & FLIST_TIME_SAME) == 0 {
			mtime, err = r.conn.ReadInt()
			if err != nil {
				return filelist[:], err
			}
		} else {
			mtime = filelist[len(filelist)-1].Mtime
		}
		//fmt.Println("MTIME ", mtime)

		/* Read the file mode. */
		var mode os.FileMode
		if (flags & FLIST_MODE_SAME) == 0 {
			val, err := r.conn.ReadInt()
			if err != nil {
				return filelist[:], err
			}
			mode = os.FileMode(val)
		} else {
			mode = filelist[len(filelist)-1].Mode
		}
		//fmt.Println("Mode", uint32(mode))

		// TODO: Sym link
		if ((mode & 32768) != 0) && ((mode & 8192) != 0) {
			sllen, err := r.conn.ReadInt()
			if err != nil {
				return filelist[:], err
			}
			slink := make([]byte, sllen)
			_, err = r.conn.Read(slink)
			if err != nil {
				panic("Failed to read symlink")
			}
			//fmt.Println("Symbolic Len:", len, "Content:", slink)
		}

		filelist = append(filelist, FileInfo{
			Path:  path,
			Size:  size,
			Mtime: mtime,
			Mode:  mode,
		})
	}

	// Sort the filelist lexicographically
	sort.Sort(filelist)

	return filelist[:], nil
}

func (r *Receiver) RequestFiles(remoteList FileList, downloadList []int) error {
	emptyBlocks := make([]byte, 16) // 4 + 4 + 4 + 4 bytes, all bytes set to 0
	var err error = nil
	for _, v := range downloadList {
		// TODO: Supports more file mode
		if remoteList[v].Mode == 0100644 || remoteList[v].Mode == 0100755 {
			err = r.conn.WriteInt(int32(v))
			if err != nil {
				log.Println("Failed to send index")
				return err
			}

			fmt.Println("Request: ", string(remoteList[v].Path), uint32(remoteList[v].Mode))
			_, err := r.conn.Write(emptyBlocks)
			if err != nil {
				return err
			}
		}

		/* EXPERIMENTAL else {
			// Handle folders & symbol links
			emptyCtx := new(bytes.Buffer)
			osClient.Put(prepath+string((*filelist)[i].Path), emptyCtx, int64(emptyCtx.Len()), FileMetadata{
				Mtime: (*filelist)[i].Mtime,
				Mode: (*filelist)[i].Mode,
			})
		}*/
	}

	// Send -1 to finish, then start to download
	err = r.conn.WriteInt(INDEX_END)
	if err != nil {
		log.Println("Can't send INDEX_END")
		return err
	}
	log.Println("Request completed")

	startTime := time.Now()
	err = r.Downloader(remoteList[:])
	log.Println("Downloaded duration:", time.Since(startTime))
	return err
}

// TODO: It is better to update files in goroutine
func (r *Receiver) Downloader(localList FileList) error {

	ppath := []byte(r.path)
	rmd4 := make([]byte, 16)

	for {
		index, err := r.conn.ReadInt()
		if err != nil {
			return err
		}
		if index == INDEX_END { // -1 means the end of transfer files
			return nil
		}
		fmt.Println("INDEX:", index)

		count, err := r.conn.ReadInt() /* block count */
		if err != nil {
			return err
		}

		blen, err := r.conn.ReadInt() /* block length */
		if err != nil {
			return err
		}

		clen, err := r.conn.ReadInt() /* checksum length */
		if err != nil {
			return err
		}

		remainder, err := r.conn.ReadInt() /* block remainder */
		if err != nil {
			return err
		}

		path := localList[index].Path
		log.Println("Downloading:", string(path), count, blen, clen, remainder, localList[index].Size)

		// If the file is too big to store in memory, creates a temporary file in the directory 'tmp'
		buffer := ubuffer.NewBuffer(localList[index].Size)
		downloadeSize := 0
		bufwriter := bufio.NewWriter(buffer)
		for {
			token, err := r.conn.ReadInt()
			if err != nil {
				return err
			}
			log.Println("TOKEN", token)
			if token == 0 {
				break
			} else if token < 0 {
				return errors.New("Does not support block checksum")
				// Reference
			} else {
				ctx := make([]byte, token) // FIXME: memory leak?
				_, err = io.ReadFull(r.conn, ctx)
				if err != nil {
					return err
				}
				downloadeSize += int(token)
				log.Println("Downloaded:", downloadeSize, "byte")
				_, err := bufwriter.Write(ctx)
				if err != nil {
					return err
				}
			}
		}
		if bufwriter.Flush() != nil {
			return errors.New("Failed to flush buffer")
		}
		// Put file to object storage
		objectName := string(append(ppath[:], path[:]...)) // prefix + path

		var n int64
		n, err = buffer.Seek(0, io.SeekStart)

		n, err = r.storage.Put(objectName, buffer, int64(downloadeSize), FileMetadata{
			Mtime: localList[index].Mtime,
			Mode:  localList[index].Mode,
		})
		if err != nil {
			return err
		}

		if buffer.Finalize() != nil {
			return errors.New("Buffer can't be finalized")
		}

		log.Printf("Successfully uploaded %s of size %d\n", path, n)

		// Remote MD4
		// TODO: compare computed MD4 with remote MD4
		_, err = io.ReadFull(r.conn, rmd4)
		if err != nil {
			return err
		}
		fmt.Println("Remote MD4:", rmd4)

		//lmd4 := md4.New()
		//lmd4.Write(buffer.Bytes())
		//if bytes.Compare(rmd4, lmd4.Sum(nil)) == 0 {
		//
		//}
	}
}

// Clean up local files
func (r *Receiver) Cleaner(localList FileList, deleteList []int) error {
	prefix := []byte(r.path)
	for i := range deleteList {
		if localList[i].Mode.IsRegular() {
			name := append(prefix, localList[i].Path...)
			err := r.storage.Delete(string(name))
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *Receiver) FinalPhase() error {
	go func() {
		ioerror, err := r.conn.ReadInt()
		log.Println(ioerror, err)
	}()

	err := r.conn.WriteInt(INDEX_END)
	if err != nil {
		return err
	}
	return r.conn.WriteInt(INDEX_END)
}

func (r *Receiver) Run() error {
	rfiles, err := r.GetFileList()
	if err != nil {
		return err
	}
	lfiles, err := r.storage.List()
	if err != nil {
		return err
	}
	newfiles, oldfiles := lfiles.Diff(rfiles[:])
	if err := r.RequestFiles(rfiles[:], newfiles[:]); err != nil {
		return err
	}
	if err := r.Cleaner(lfiles[:], oldfiles[:]); err != nil {
		return err
	}
	if err := r.FinalPhase(); err != nil {
		return err
	}
	return nil
}

func readLine(conn *Conn) (string, error) {
	// until \n, then add \0
	line := new(bytes.Buffer)
	for {
		c, err := conn.ReadByte()
		if err != nil {
			return "", err
		}

		if c == '\r' {
			continue
		}

		err = line.WriteByte(c)
		if err != nil {
			return "", err
		}

		if c == '\n' {
			line.WriteByte(0)
			break
		}

		if c == 0 {
			break
		}
	}
	return line.String(), nil
}
