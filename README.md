# RSYNC-OS
## A rsync-compatible tool for object storage

![client](https://raw.githubusercontent.com/kaiakz/rsync-os/master/docs/client.jpg)

## Usage
### minio
1. install & run minio, you need to configure the `config.yaml`.
2. `go build`
3. `./rsync-os rsync://[USER@]HOST[:PORT]/SRC minio`, for example, `./rsync-os rsync://mirrors.tuna.tsinghua.edu.cn/ubuntu minio`

## Roadmap
### Client
#### Rsync wire protocol 27:
- [x] Parses rsync://
- [x] Connects to rsync server
- [x] Handshake
- [x] Sends argument list
- [x] Fetches the file list
- [x] Requests & download files
- [ ] Handles error
#### File List Caching
- [x] BoltDB backend
- [ ] Redis backend
- [x] Diff 
- [x] Update
#### Storage backend
- [x] Minio: supports regular files
- [ ] Minio: supports folder & symlink
- [ ] FS
#### Other
- [x] CLI

## Detailed Information
#### openrsync has a really good [documentation](https://github.com/kristapsdz/openrsync/blob/master/README.md) to describe how rsync algorithm works. 

#### Why do this?
Just as [rsyn](https://github.com/sourcefrog/rsyn#why-do-this) said, "The rsync C code is quite convoluted, with many interacting options and parameters stored in global variables affecting many different parts of the control flow, including how structures are encoded and decoded." I would like to provide a rsync written in clean and understandable Golang code. 

rsync has [a bad performance](https://github.com/tuna/rsync/blob/master/README-huai.md). Inspired by rsync-huai, rsync-os stores the file list in database to avoid recursively generating the list.

Modernized rsync: rsync-os supports both file storage and object storage.

#### What's the difference between rsync and rsync-os
rsync-os is the express edition of rsync, with object storage support. It uses a subset of rsync wire protocol(without rolling block checksum).

#### rsync-os and rclone are completely different
rclone does not support rsync wire protocol although it is called "rsync for object storage". With rclone you can't transfer files between rsync and object storage.

#### Why we don't need rolling block checksum for regular file?
In rsync algorithm, rsync requires random reading and writing of files to do the block exchange. But object storage does not support that.

rsync-os simplifies the rsync algorithm to avoid random reading and writing, since rsync-os don't need to do a rolling checksum scanning the file. 

As a client, when a file has different size or modified time compared to the remote file, rsync-os just pretend 'the file does not exist here', then send a reply to download the entire file from the server and finally replace it.

As a server, 

#### HandShake
rysnc-os supports rsync protocol 27. 
Now it sends the arguments "--server--sender-l-p-r-t" to the remote rsyncd by default.

#### The File List
According to the arguments rsync-os sent, the file list should contain path, size, modified time & mode. 
 
#### Request the file
rsync-os always saves the file list in its database(local file list). rsync2os doesn't compare each file with the file list from remote server(remote file list), but the latest local file list. If the file in the local file list has different size, modified time, or doesn't exist, rsync2os will download the whole file(without [block exchange](https://github.com/kristapsdz/openrsync#block-exchange)). To to do that, rsync2os sends the empty block checksum with the file's index of the remote file list. 

#### Download the file
The rsync server sends the entire file as a stream of bytes.

#### Multiplex & De-Multiplex
Most rsync transmissions are wrapped in a multiplexing envelope protocol. The code written in C to multiplex & de-multiplex is obscure. 
Unlike rsync, rsync-os reimplements this part: It just does multiplexing & de-multiplexing in a goroutine.
![de-multiplex](https://raw.githubusercontent.com/kaiakz/rsync-os/master/docs/demux.jpg)

### Limitations
* Do not support block exchange for regular files. If a file was modified, just downloads the whole file.
* rsync-os can only act as client/receiver now.

# Reference
* [rsync](https://rsync.samba.org/)
* [openrsync](https://github.com/openbsd/src/tree/master/usr.bin/rsync), a BSD-liscesed rsync
* [rync-huai](https://github.com/tuna/rsync), a modified version rsync by Tsinghua University TUNA Association
* [rsyn](https://github.com/sourcefrog/rsyn), wire-compaible rsync in Rust
* https://github.com/gilbertchen/acrosync-library
* https://rsync.samba.org/resources.html
* https://github.com/boundary/wireshark/blob/master/epan/dissectors/packet-rsync.c
* https://tools.ietf.org/html/rfc5781