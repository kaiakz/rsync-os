package storage

import (
	"bytes"
	"errors"
	"github.com/golang/protobuf/proto"
	"github.com/minio/minio-go/v6"
	bolt "go.etcd.io/bbolt"
	"io"
	"log"
	"os"
	"path/filepath"
	"rsync-os/fldb"
	"rsync-os/rsync"
	"strconv"
)

/*
rsync-os will add addition information for each file that was uploaded to minio
rsync-os stores the information of a folder in the metadata of an empty file called "..."
rsync-os also uses a strange file to represent a soft link
*/

// S3 with cache
type Minio struct {
	client     *minio.Client
	bucketName string
	prefix string
	/* Cache */
	cache      *bolt.DB
	tx *bolt.Tx
	bucket *bolt.Bucket
}

//endpoint := "127.0.0.1:9000"
//accessKeyID := "minioadmin"
//secretAccessKey := "minioadmin"

func NewMinio(bucket string, prefix string, cachePath string, endpoint string, accessKeyID string, secretAccessKey string, secure bool) *Minio {
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

	// Initialize cache
	db, err := bolt.Open(cachePath, 0666, nil)
	if err != nil {
		panic("Can't init cache: boltdb")
	}
	tx, err := db.Begin(true)
	if err != nil {
		panic(err)
	}

	mod := tx.Bucket([]byte(bucket))
	// If bucket does not exist, create the bucket
	if mod == nil {
		var err error
		mod, err = tx.CreateBucket([]byte(bucket))
		if err != nil {
			panic(err)
		}
	}

	return &Minio{
		client:     minioClient,
		bucketName: bucket,
		prefix:     prefix,
		cache:      db,
		tx:         tx,
		bucket:     mod,
	}
}

// object can be a regualar file, folder or symlink
func (m *Minio) Put(fileName string, content io.Reader, fileSize int64, metadata rsync.FileMetadata) (written int64, err error) {
	data := make(map[string]string)
	data["mtime"] = strconv.Itoa(int(metadata.Mtime))
	data["mode"] = strconv.Itoa(int(metadata.Mode))

	/* EXPERIMENTAL
	// Folder
	if metadata.Mode.IsDir() {
		ctx := new(bytes.Buffer)
		signName := fileName + "/..."
		// FIXME: How to handle a file named "..." as well ?
		return m.client.PutObject(m.bucketName, signName, content, int64(sign.Len()), minio.PutObjectOptions{UserMetadata: data})
	}
	// TODO: symlink
	if metadata.Mode & os.ModeSymlink != 0 {
		ctx := new(bytes.Buffer)
		// Additional data of symbol link
		return m.client.PutObject(m.bucketName, fileName, content, int64(sign.Len()), minio.PutObjectOptions{UserMetadata: data})
	}
	*/
	value, err := proto.Marshal(&fldb.FInfo{
		Size:  fileSize,
		Mtime: metadata.Mtime,
		Mode:  int32(metadata.Mode),	// FIXME: convert uint32 to int32
	})
	if err != nil {
		return -1, err
	}
	if err := m.bucket.Put([]byte(fileName), value); err != nil {
		return -1, err
	}
	return m.client.PutObject(m.bucketName, fileName, content, fileSize, minio.PutObjectOptions{UserMetadata: data})
}

func (m *Minio) Delete(objectName string) error {
	// TODO: How to delete a folder
	return m.client.RemoveObject(m.bucketName, objectName)
}

// EXPERIMENTAL
func (m *Minio) List() (rsync.FileList, error) {
	filelist := make(rsync.FileList, 0, 1 << 16)

	/*
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
	*/


	info := &fldb.FInfo{}

	// Add current dir as .
	workdir := []byte(filepath.Clean(m.prefix))
	v := m.bucket.Get(workdir)
	if v == nil {
		return filelist[:], errors.New("Work Dir's info does not exists")
	}
	if err := proto.Unmarshal(v, info); err != nil {
		return filelist, err
	}
	filelist = append(filelist, rsync.FileInfo{
		Path:  workdir,
		Size:  info.Size,
		Mtime: info.Mtime,
		Mode:  os.FileMode(info.Mode),
	})

	// Add files in the work dir
	c := m.bucket.Cursor()
	prefix := []byte(m.prefix)
	k, v := c.Seek(prefix)
	for k != nil && bytes.HasPrefix(k, prefix) {
		if err := proto.Unmarshal(v, info); err != nil {
			return filelist, err
		}
		filelist = append(filelist, rsync.FileInfo{
			Path:  k[len(prefix):],
			Size:  info.Size,
			Mtime: info.Mtime,
			Mode:  os.FileMode(info.Mode),
		})
		k, v = c.Next()
	}

	return filelist, nil
}

func (m *Minio) DeleteAll(prefix []byte, deleteList [][]byte) error {
	for _, rkey := range deleteList {
		key := string(append(prefix, rkey...))
		// FIXME: ignore folder & symlink
		err := m.Delete(key)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *Minio) Close() error {
	if err := m.tx.Commit(); err != nil {
		return err
	}
	if err := m.cache.Close(); err != nil {
		return err
	}
	return nil
}
