# RSYNC-OS
## A rsync-compatible tool for object storage

![client](https://raw.githubusercontent.com/kaiakz/rsync-os/master/docs/client.jpg)

## Usage
### minio
1. install & run minio, you need to configure the `config.yaml`.
2. `go build && go run rsync-os`

## Roadmap
### Client
- [x] Parse rsync://
- [x] Connect to rsync server
- [x] Hand shake
- [x] Send argument list
- [x] Fetch the file list
- [x] Request & download files
- [x] File List Caching
- [ ] CLI

## Detailed Information
#### What's the difference between rsync and rsync-os
rsync-os is the express edition of rsync, with object storage support. It uses a subset of rsync wire protocol(without block checksum).

#### rsync-os and rclone are completely different
rclone does not support rsync wire protocol although it is called "rsync for object storage". With rclone you can't transfer files between rsync and object storage with rclone.

#### Why we don't need block checksum?
rsync requires random reading and writing of files to do the block exchange. But object storage does not support that.
rsync-os simplifies the rsync algorithm to avoid random reading and writing. When a file needs to be updated, we just download the entire file from the server and then replace it.

#### HandShake
rysnc-os supports rsync protocol 27. 
It sends the arguments "--server--sender-l-p-r-t" to the remote rsyncd by default.

#### The File List
According the arguments rsync-os sent, the file list should contain path, size, modified time & mode. 
 
#### Request the file
rsync-os always saves the file list in its database(local file list). rsync2os doesn't compare each file with the file list from remote server(remote file list), but the latest local file list. If the file in the local file list has different size, modified time, or doesn't exist, rsync2os will download the whole file(without [block exchange](https://github.com/kristapsdz/openrsync#block-exchange)). To to do that, rsync2os sends the empty block checksum with the file's index of the remote file list. 

#### Download the file
The rsync server sends the entire file as a stream of bytes.

#### Multiplex & De-Multiplex
![de-multiplex](https://raw.githubusercontent.com/kaiakz/rsync-os/master/docs/demux.jpg)

### Limitations

# Reference
* https://git.samba.org/?p=rsync.git
* https://rsync.samba.org/resources.html
* https://github.com/openbsd/src/tree/master/usr.bin/rsync
* https://github.com/tuna/rsync
* https://github.com/sourcefrog/rsyn
* https://github.com/gilbertchen/acrosync-library
* https://github.com/boundary/wireshark/blob/master/epan/dissectors/packet-rsync.c
* https://tools.ietf.org/html/rfc5781