package rsync

type SumStruct struct {
	fileLen uint64		// totol file length
	count uint64		// how many chunks
	remainder uint64	// fileLen % blockLen
	blockLen uint64		// block length
	sum2Len uint64		// longSum length
	sumList []SumChunk		// chunks
}

type SumChunk struct {
	fileOffset int64
	chunkLen   uint
	shortSum   int32  // short checksum
	longSum    []byte // long checksum
}


