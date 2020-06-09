package rsync

import (
	"bytes"
	"encoding/binary"
	"github.com/minio/minio-go/v6"
	"golang.org/x/crypto/md4"
	"io"
	//"io/ioutil"
	"log"
	"net"
)

//channel: read & write

//Multiplexing
//Most rsync transmissions are wrapped in a multiplexing envelope protocol.  It is
//composed as follows:
//
//1.   envelope header (4 bytes)
//2.   envelope payload (arbitrary length)
//
//The first byte of the envelope header consists of a tag.  If the tag is 7, the pay‐
//load is normal data.  Otherwise, the payload is out-of-band server messages.  If the
//tag is 1, it is an error on the sender's part and must trigger an exit.  This limits
//message payloads to 24 bit integer size, 0x00ffffff.
//
//The only data not using this envelope are the initial handshake between client and
//server

// Buffer version
type DeMuxer struct {
	conn net.Conn
	buf bytes.Buffer
}

func RunDeMuxer(conn net.Conn) *DeMuxer {
	buf := new(bytes.Buffer)
	go DeMuxBuf(conn, buf)
	return &DeMuxer{
		conn: conn,
		buf: *buf,
	}
}

// Deprecated
func DeMuxBuf(conn net.Conn, buf *bytes.Buffer)  {
	for {
		// socket read the multipex data & put them to channel
		header := make([]byte, 4)		// Header size: 4 bytes
		n, err := ReadExact(conn, header)

		if n != 4 || err != nil {
			return
		}

		tag := header[3]	// Little Endian
		size := (binary.LittleEndian.Uint32(header) & 0xffffff)		// TODO: zero?

		log.Println("TAG", tag, "SIZE", size)

		if tag == 7 {
			body := make([]byte, size)

			_, err := ReadExact(conn, body)

			if err == io.EOF {	// Finish
				return
			}



		} else {	// out-of-band data
			//otag := tag - 7

		}
	}
}

// Deprecated
func DeMultiplex(conn net.Conn) error {
	// socket read the multipex data & put them to channel
		header := make([]byte, 4)		// Header size: 4 bytes

		n, err := ReadExact(conn, header)
		if n != 4 || err != nil {
			return io.ErrUnexpectedEOF
		}

		tag := header[3]	// Little Endian
		size := (binary.LittleEndian.Uint32(header) & 0xffffff)		// TODO: zero?

		log.Println("TAG", tag, "SIZE", size)

		if tag == 7 {
			body := make([]byte, size)

			_, err := ReadExact(conn, body)

			if err == io.EOF {	// Finish
				return err
			}

			log.Println(body)

			//if (body[size-1] | body[size-2] | body[size-3] | body[size-4] | body[size-5]) == 0 {
			//	fmt.Println("END")
			//	return io.EOF
			//}



			//for _, b := range body {
			//	data <- b
			//}

		} else {	// out-of-band data
			//otag := tag - 7

		}
		return nil
}

// Goroutine: Demultiplex the package, and push them to channel
// data: Buffered Channel
func DeMuxChan(conn net.Conn, data chan byte) {
	for {
		// socket read the multipex data & put them to channel
		header := make([]byte, 4)		// Header size: 4 bytes

		n, err := ReadExact(conn, header)
		if n != 4 || err != nil {
			panic("Mulitplex: Check your wired protocol")
		}

		tag := header[3]	// Little Endian
		size := (binary.LittleEndian.Uint32(header) & 0xffffff)		// TODO: zero?

		log.Println("*****TAG", tag, "SIZE", size, "*****")

		if tag == 7 {	// MUL_BASE + MSG_DATA
			body := make([]byte, size)

			_, err := ReadExact(conn, body)

			if err == io.EOF {	// Finish
				panic("EOF")
			}

			for _, b := range body {
				data <- b
			}

		} else {	// out-of-band data
			//otag := tag - 7
			panic("Error")
		}
	}
}

// Blocking: copy len(b) bytes from channel to b
func GetBytes(data chan byte, b []byte) {
	for i:=0; i < len(b); i++ {
		b[i] = <- data
	}
}

func GetShort(data chan byte) int16 {
	val:= make([]byte, 2)
	GetBytes(data, val)
	return int16(binary.LittleEndian.Uint16(val))
}

func GetByte(data chan byte) byte {
	return <-data
}

func GetUint8(data chan byte) uint8 {
	return uint8(<- data)
}

func GetInteger(data chan byte) int32 {
	val := make([]byte, 4)
	GetBytes(data, val)
	return int32(binary.LittleEndian.Uint32(val))
}

func GetLong(data chan byte) int64 {
	val := make([]byte, 8)
	GetBytes(data, val)
	return int64(binary.LittleEndian.Uint64(val))
}

func GetVarint(data chan byte) int64 {
	sval := GetInteger(data)
	if sval != -1 {
		return int64(sval)
	}

	return GetLong(data)
}

// FIXME
func GetFiles(data chan byte, conn net.Conn, filelist *FileList) {
	for {
		idx := GetInteger(data)
		if idx == -1 {
			return
		}
		log.Println("Server send the file: ", idx)
		// TODO: idx out of range?

		GetFile(data, idx, filelist)
	}
}

func lookup(size int64, filelist *FileList) {
	for i,f := range(*filelist) {
		if f.Size == size {
			log.Println("True File:", i, f)
		}
	}
}

// Test
func GetFile(data chan byte, index int32, filelist *FileList) {

	path := (*filelist)[index].Path

	count := GetInteger(data)  /* block count */
	blen := GetInteger(data)  /* block length */
	clen := GetInteger(data)  /* checksum length */
	remainder := GetInteger(data)  /* block remainder */

	log.Println(path, count, blen, clen, remainder, (*filelist)[index].Size)
	buf := new(bytes.Buffer)
	for {
		token := GetInteger(data)
		log.Println("TOKEN", token)
		if token == 0 {
			break
		} else if token < 0 {
			// Reference
		} else {
			ctx := make([]byte, token)
			GetBytes(data, ctx)
			log.Println("Buff size:", buf.Len())
			buf.Write(ctx)
		}
	}
	log.Println("Buff Total size:", buf.Len())
	//lookup(int64(buf.Len()), filelist)
	//ioutil.WriteFile("temp.txt", buf.Bytes(), 0644)
	WriteOS(buf, path)

	lmd4 := md4.New()
	lmd4.Write(buf.Bytes())

	// Remote MD4
	rmd4 := make([]byte, 16)
	GetBytes(data, rmd4)
	log.Println("MD4", rmd4)
	log.Println("Compute MD4", lmd4.Sum(nil))
}

func WriteOS(buf *bytes.Buffer, fname string) {
	// For test
	endpoint := "127.0.0.1:9000"
	accessKeyID := "minioadmin"
	secretAccessKey := "minioadmin"

	minioClient, err := minio.New(endpoint, accessKeyID, secretAccessKey, false)
	if err != nil {
		log.Println("MINIO")
		return
	}

	// Make a new bucket called mymusic.
	bucketName := "test"
	//location := "cn"
	log.Println("Making Bucket: test")
	err = minioClient.MakeBucket(bucketName, "us-east-1")
	if err != nil {
		// Check to see if we already own this bucket (which happens if you run this twice)
		exists, errBucketExists := minioClient.BucketExists(bucketName)
		if errBucketExists == nil && exists {
			log.Printf("We already own %s\n", bucketName)
		} else {
			log.Fatalln(err)
		}
	} else {
		log.Printf("Successfully created %s\n", bucketName)
	}

	log.Println("Updating")
	// Upload the zip file
	//objectName := "golden-oldies.zip"
	//contentType := "application/x-rpm"

	// Upload the zip file with FPutObject
	n, err := minioClient.PutObject(bucketName, fname, buf, int64(buf.Len()), minio.PutObjectOptions{})
	if err != nil {
		log.Fatalln(err)
	}

	log.Printf("Successfully uploaded %s of size %d\n", fname, n)
}
