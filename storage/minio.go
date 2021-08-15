package storage

import (
	"bytes"
	"io"
	"log"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/kaiakz/rsync-os/rsync"
	"github.com/kaiakz/rsync-os/storage/cache"
	"github.com/minio/minio-go/v6"
	bolt "go.etcd.io/bbolt"
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
	prefix     string
	/* Cache */
	cache  *bolt.DB
	tx     *bolt.Tx
	bucket *bolt.Bucket
}

const S3_DIR = ".dir.rsync-os"

func NewMinio(bucket string, prefix string, cachePath string, endpoint string, accessKeyID string, secretAccessKey string, secure bool) (*Minio, error) {
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

	// If bucket does not exist, create the bucket
	mod, err := tx.CreateBucketIfNotExists([]byte(bucket))
	if err != nil {
		return nil, err
	}

	return &Minio{
		client:     minioClient,
		bucketName: bucket,
		prefix:     prefix,
		cache:      db,
		tx:         tx,
		bucket:     mod,
	}, nil
}

// object can be a regualar file, folder or symlink
func (m *Minio) Put(fileName string, content io.Reader, fileSize int64, metadata rsync.FileMetadata) (written int64, err error) {
	data := make(map[string]string)
	data["mtime"] = strconv.Itoa(int(metadata.Mtime))
	data["mode"] = strconv.Itoa(int(metadata.Mode))

	fpath := filepath.Join(m.prefix, fileName)
	fsize := fileSize
	fname := fpath
	/* EXPERIMENTAL */
	// Folder
	if metadata.Mode.IsDIR() {
		fname = filepath.Join(m.prefix, fileName, S3_DIR)
		fsize = 0
		// FIXME: How to handle a file named ".rsync-os.dir"?
	}

	if metadata.Mode.IsLNK() {
		// Additional data of symbol link
	}

	written, err = m.client.PutObject(m.bucketName, fname, content, fsize, minio.PutObjectOptions{UserMetadata: data})

	value, err := proto.Marshal(&cache.FInfo{
		Size:  fileSize,
		Mtime: metadata.Mtime,
		Mode:  int32(metadata.Mode), // FIXME: convert uint32 to int32
	})
	if err != nil {
		return -1, err
	}
	if err := m.bucket.Put([]byte(fpath), value); err != nil {
		return -1, err
	}

	return
}

func (m *Minio) Delete(fileName string, mode rsync.FileMode) (err error) {
	fpath := filepath.Join(m.prefix, fileName)
	// TODO: How to delete a folder
	if mode.IsDIR() {
		err = m.client.RemoveObject(m.bucketName, filepath.Join(fpath, S3_DIR))
	} else {
		if err = m.client.RemoveObject(m.bucketName, fpath); err != nil {
			return
		}
	}
	log.Println(fileName)
	return m.bucket.Delete([]byte(fpath))
}

// EXPERIMENTAL
func (m *Minio) List() (rsync.FileList, error) {
	filelist := make(rsync.FileList, 0, 1<<16)

	// We don't list all files directly

	info := &cache.FInfo{}

	// Add files in the work dir
	c := m.bucket.Cursor()
	prefix := []byte(m.prefix)
	k, v := c.Seek(prefix)
	hasdot := false
	for k != nil && bytes.HasPrefix(k, prefix) {
		p := k[len(prefix):]
		if bytes.Equal(p, []byte(".")) {
			hasdot = true
		}

		if err := proto.Unmarshal(v, info); err != nil {
			return filelist, err
		}
		filelist = append(filelist, rsync.FileInfo{
			Path:  p, // ignore prefix
			Size:  info.Size,
			Mtime: info.Mtime,
			Mode:  rsync.FileMode(info.Mode),
		})
		k, v = c.Next()
	}

	// Add current dir as .
	if !hasdot {
		workdir := []byte(filepath.Clean(m.prefix)) // If a empty string, we get "."
		v := m.bucket.Get(workdir)
		if v == nil {
			return filelist, nil
		}
		if err := proto.Unmarshal(v, info); err != nil {
			return filelist, err
		}
		filelist = append(filelist, rsync.FileInfo{
			Path:  []byte("."),
			Size:  info.Size,
			Mtime: info.Mtime,
			Mode:  rsync.FileMode(info.Mode),
		})
	}

	sort.Sort(filelist)

	return filelist, nil
}

func (m *Minio) ListObj() (rsync.FileList, error) {
	filelist := make(rsync.FileList, 0, 1<<16)

	// Create a done channel to control 'ListObjects' go routine.
	doneCh := make(chan struct{})

	// Indicate to our routine to exit cleanly upon return.
	defer close(doneCh)

	// FIXME: objectPrefix, recursive
	objectCh := m.client.ListObjectsV2(m.bucketName, "", true, doneCh)
	for object := range objectCh {
		if object.Err != nil {
			log.Println(object.Err)
			return filelist, object.Err
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
			Mode:  rsync.FileMode(mode),
		})
	}
	return filelist, nil
}

func (m *Minio) Close() error {
	defer m.cache.Close()
	if err := m.tx.Commit(); err != nil {
		return err
	}
	return nil
}
