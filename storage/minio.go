package storage

import (
	"github.com/minio/minio-go/v6"
	"io"
	"log"
	"os"
	"path"
	"rsync-os/rsync"
	"strconv"
	"strings"
)


/*
rsync-os will add addition information for each file that was uploaded to minio
rsync-os stores the information of a folder in the metadata of an empty file called "..."
rsync-os also uses a strange file to represent a soft link
*/

// A bucketName
type Minio struct {
	client     *minio.Client
	bucketName string
}

//endpoint := "127.0.0.1:9000"
//accessKeyID := "minioadmin"
//secretAccessKey := "minioadmin"

func NewMinio(bucket string, endpoint string, accessKeyID string, secretAccessKey string, secure bool) *Minio {
	minioClient, err := minio.New(endpoint, accessKeyID, secretAccessKey, false)
	if err != nil {
		panic("Failed to init a minio client")
	}
	// Create a bucket for the module
	err = minioClient.MakeBucket(bucket, "us-east-1")
	if err != nil {
		// Check to see if we already own this bucket (which happens if you run this twice)
		exists, errBucketExists := minioClient.BucketExists(bucket)
		if errBucketExists == nil && exists {
			log.Printf("We already own %s\n", bucket)
		} else {
			log.Fatalln(err)
		}
	} else {
		log.Printf("Successfully created %s\n", bucket)
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
func (m *Minio) Uploader() {

}

// object can be a regualar file, folder or symlink
func (m *Minio) Write(objectName string, reader io.Reader, objectSize int64, metadata rsync.FileMetadata) (n int64, err error) {
	data := make(map[string] string)
	data["mtime"] = strconv.Itoa(int(metadata.Mtime))
	data["mode"] = strconv.Itoa(int(metadata.Mode))

	/* EXPERIMENTAL
	// Folder
	if metadata.Mode.IsDir() {
		ctx := new(bytes.Buffer)
		signName := objectName + "/..."
		// FIXME: How to handle a file named "..." as well ?
		return m.client.PutObject(m.bucketName, signName, reader, int64(sign.Len()), minio.PutObjectOptions{UserMetadata: data})
	}
	// TODO: symlink
	if metadata.Mode & os.ModeSymlink != 0 {
		ctx := new(bytes.Buffer)
		// Additional data of symbol link
		return m.client.PutObject(m.bucketName, objectName, reader, int64(sign.Len()), minio.PutObjectOptions{UserMetadata: data})
	}
	*/

	return m.client.PutObject(m.bucketName, objectName, reader, objectSize, minio.PutObjectOptions{UserMetadata: data})
}

func (m *Minio) Delete(objectName string) error {
	// TODO: How to delete a folder
	return m.client.RemoveObject(m.bucketName, objectName)
}

// EXPERIMENTAL
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

		// FIXME: Handle folder
		objectName := object.Key
		if strings.Compare(path.Base(objectName), "...") == 0 {
			objectName = path.Dir(objectName)
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
			Path:  []byte(objectName),
			Size:  object.Size,
			Mtime: int32(mtime),
			Mode:  os.FileMode(mode),
		})
	}
	return filelist[:]
}

func (m *Minio) DeleteAll(prefix []byte, deleteList [][]byte) error {
	for _, rkey := range deleteList	{
		key := string(append(prefix, rkey...))
		// FIXME: ignore folder & symlink
		err := m.Delete(key)
		if err != nil {
			return err
		}
	}
	return nil
}