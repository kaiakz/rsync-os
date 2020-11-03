package rsync

type SumStruct struct {
	fileLen uint64		// totol file length
	count uint64		// how many chunks
	remainder uint64	// fileLen % blockLen
	blockLen uint64		// block length
	sum2Len uint64		// sum2 length
	sumList []SumChunk		// chunks
}

type SumChunk struct {
	fileOffset int64
	chunkLen uint
	sum1 uint32
	sum2 []byte
}


