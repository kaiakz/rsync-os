package rsync

import (
	"bytes"
)

type Sender struct {
	conn    *Conn
	module  string
	path    string
	seed    int32
	lVer    int32
	rVer    int32
	storage FS
}

func (s *Sender) SendFileList() error {
	list, err := s.storage.List()
	if err != nil {
		return err
	}

	// Send list to receiver
	var last *FileInfo = nil
	for _, f := range list{
		var flags byte = 0


		if bytes.Equal(f.Path, []byte(".")) {
			if f.Mode.IsDIR() {
				flags |= FLIST_TOP_LEVEL
			}
		} else {
			if f.Mode.IsDIR() { // TODO: recursive
				// flags |= Flags.NO_CONTENT_DIR | Flags.XFLAGS;
			}
		}

		lPathCount := 0
		if last != nil {
			lPathCount = longestMatch(last.Path, f.Path)
			if lPathCount > 255 { // Limit to 255 chars
				lPathCount = 255
			}
			if lPathCount > 0 {
				flags |= FLIST_NAME_SAME
			}
			if last.Mode == f.Mode {
				flags |= FLIST_MODE_SAME
			}
			if last.Mtime == f.Mtime {
				flags |= FLIST_TIME_SAME
			}
			//
			//
			//
		}

		rPathCount := int32(len(f.Path) - lPathCount)
		if  rPathCount > 255 {
			flags |= FLIST_NAME_LONG
		}

		/* we must make sure we don't send a zero flags byte or the other
		   end will terminate the flist transfer */
		if flags == 0 && !f.Mode.IsDIR() {
			flags |= 1<<0
		}
		if flags == 0 {
			flags |= FLIST_NAME_LONG
		}
		/* Send flags */
		if err != s.conn.WriteByte(flags) {
			return err
		}

		/* Send len of path, and bytes of path */
		if flags& FLIST_NAME_SAME != 0 {
			if err = s.conn.WriteByte(flags); err != nil {
				return err
			}
		}

		if flags& FLIST_NAME_LONG != 0 {
			if err = s.conn.WriteInt(rPathCount); err != nil {
				return err
			}
		} else {
			if err = s.conn.WriteByte(byte(rPathCount)); err != nil {
				return err
			}
		}

		if _, err = s.conn.Write(f.Path[lPathCount:]); err != nil {
			return err
		}

		/* Send size of file */
		if err = s.conn.WriteLong(f.Size); err != nil {
			return err
		}

		/* Send Mtime, GID, UID, RDEV if needed */
		if flags& FLIST_TIME_SAME == 0 {
			if err = s.conn.WriteInt(f.Mtime); err != nil {
				return err
			}
		}
		if flags& FLIST_MODE_SAME == 0 {
			if err = s.conn.WriteInt(int32(f.Mode)); err != nil {
				return err
			}
		}
		// TODO: UID GID RDEV

		// TODO: Send symlink

		// TODO: if always_checksum?

		last = &f
	}
	return nil
}

func (s *Sender) Generator(fileList FileList) error {
	for {
		index, err := s.conn.ReadInt()
		if err != nil {
			return err
		} else if index == INDEX_END {
			break
		}

		// Receive block checksum from receiver
		count, err := s.conn.ReadInt()
		if err != nil {
			return nil
		}

		blen, err := s.conn.ReadInt()
		if err != nil {
			return nil
		}

		s2len, err := s.conn.ReadInt()
		if err != nil {
			return nil
		} else if s2len > 16 {
			// FIXME: check if sum2 length is valid
		}

		remainder, err := s.conn.ReadInt()
		if err != nil {
			return nil
		}

		sums := make([]SumChunk, 0, count)

		var (
			i int32 = 0
			offset int64 = 0
		)


		for ; i < count; i++ {
			sum1, err := s.conn.ReadInt() // sum1:
			if err != nil {
				return err
			}

			sum2 := make([]byte, 16)
			if _, err := s.conn.Read(sum2); err != nil {
				return err
			}

			chunk := new(SumChunk)
			chunk.sum1 = uint32(sum1)
			chunk.sum2 = sum2
			chunk.fileOffset = offset

			if i == count-1 && remainder != 0 {
				chunk.chunkLen = uint(remainder);
			} else {
				chunk.chunkLen = uint(blen)
			}
			offset += int64(chunk.chunkLen)
			sums = append(sums, )
		}
		result := new(SumStruct)
		result.fileLen = uint64(offset)
		result.count = uint64(count)
		result.blockLen = uint64(blen)
		result.sum2Len = uint64(s2len)
		result.remainder = uint64(remainder)

	}
	if err := s.FileUploader(); err != nil {
		return err
	}
	return nil
}

func (s *Sender) FileUploader() error {
	panic("Not implemented yet");
	return nil
}

func (s *Sender) FinalPhase() error {
	panic("Not implemented yet");
	return nil
}

func (s *Sender) Sync() error {
	panic("Not implemented yet");
	return nil
}