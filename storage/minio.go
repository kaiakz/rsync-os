package storage

import (
	"github.com/minio/minio-go/v6"
	"io"
	"log"
	"rsync-os/rsync"
	"strconv"
)

// A bucketName
type Minio struct {
	client     *minio.Client
	bucketName string
}

//endpoint := "127.0.0.1:9000"
//accessKeyID := "minioadmin"
//secretAccessKey := "minioadmin"

func New(bucket string, endpoint string, accessKeyID string, secretAccessKey string, secure bool) *Minio {
	minioClient, err := minio.New(endpoint, accessKeyID, secretAccessKey, false)
	if err != nil {
		panic("Failed to init a minio client")
	}
	return &Minio{
		client:     minioClient,
		bucketName: bucket,
	}
}

func (m *Minio) SetBucket(bucket string) {
	m.bucketName = bucket
}

func (m *Minio) GetBucket() string {
	return m.bucketName
}


// Upload a file in goroutine
func Uploader() {

}

func (m *Minio) Write(fileName string, reader io.Reader, fileSize int64, metadata FileMetadata) (n int64, err error) {
	data := make(map[string] string)
	data["mtime"] = strconv.Itoa(int(metadata.Mtime))
	data["mode"] = strconv.Itoa(int(metadata.Mode))
	return m.client.PutObject(m.bucketName, fileName, reader, fileSize, minio.PutObjectOptions{UserMetadata: data})
}

func (m *Minio) Delete(fileName string) error {
	// TODO: How to delete a folder
	return m.client.RemoveObject(m.bucketName, fileName)
}

func (m *Minio) List() rsync.FileList {
	filelist := make(rsync.FileList, 0, 1024 * 1024)

	// Create a done channel to control 'ListObjects' go routine.
	doneCh := make(chan struct{})

	// Indicate to our routine to exit cleanly upon return.
	defer close(doneCh)

	// FIXME: objectPrefix, recursive
	objectCh := m.client.ListObjectsV2(m.bucketName, "", true, doneCh)
	for object := range objectCh {
		if object.Err != nil {
			log.Println(object.Err)
			return nil
		}
		mtime, err := strconv.Atoi(object.UserMetadata["mtime"])
		if err != nil {
			panic("Can't get the mode from minio")
		}

		mode, err := strconv.Atoi(object.UserMetadata["mtime"])
		if err != nil {
			panic("Can't get the mode from minio")
		}

		filelist = append(filelist, rsync.FileInfo{
			Path:  []byte(object.Key),
			Size:  object.Size,
			Mtime: int32(mtime),
			Mode:  int32(mode),
		})
	}
	return filelist[:]
}