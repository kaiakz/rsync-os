# rsync API
**STATUS: UNSTABLE** With the API, you can create an rsync receiver/sender and customize how it works by your arguments and callbacks.

### Preknowledge
https://rsync.samba.org/how-rsync-works.html

## v0.2.0
### Client
#### receiver: 
First of all, you need to implement the rsync.FS interface to specify a storage backend used for the task.
```go
type FS interface {
	Put(fileName string, content io.Reader, fileSize int64, metadata FileMetadata) (written int64, err error)
	Get(fileName string, metadata FileMetadata) (File, error)
	Delete(fileName string, mode FileMode) error
	List() (FileList, error)
	Stats() (seekable bool)
}
```
Then, decide what arguments you would like to use:
```go
type Attribs struct {
	sender        bool // --sender
	server        bool // --server
	recursive     bool // -r
	dryRun        bool // -n
	hasTimes bool // -t
	hasPerms      bool // -p
	hasLinks      bool // -l
	hasGID        bool // -g
	hasUID        bool // -u
}
```

```go
// via socket
func SocketClient(storage FS, address string, module string, path string, options map[string]string) (SendReceiver, error)

// via ssh
func SshClient(storage FS, address string, module string, path string, options map[string]string) (SendReceiver, error)
```
Call `Run()` for the receiver to start a syncing task.
