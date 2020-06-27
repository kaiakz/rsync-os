# RSYNC-OS
## A rsync gateway for object storage.

![client](https://raw.githubusercontent.com/kaiakz/rsync2os/master/docs/client.jpg)

## Why we don't need block checksum?
Rsync requires random reading and writing of files to do the block exchange. But object storage does not support that.
rsync-os simplifies the rsync algorithm to avoid random reading and writing. When a file needs to be updated, we just download the entire file from the server and then replace it.

## HandShake
rysnc-os uses rsync protocol 27. 
It sends the arguments "--server--sender-l-p-r-t" to the remote rsyncd by default.

## The File List
According the arguments rsync-os sent, the file list should contain path, size, modified time & mode. 
 
## Request the file
rsync-os always saves the file list in its database(local file list). rsync2os doesn't compare each file with the file list from remote server(remote file list), but the latest local file list. If the file in the local file list has different size, modified time, or doesn't exist, rsync2os will download the whole file(without [block exchange](https://github.com/kristapsdz/openrsync#block-exchange)). To to do that, rsync2os sends the empty block checksum with the file's index of the remote file list. 

## Download the file
The rsync server sends the entire file as a stream of bytes.

## Multiplex & De-Multiplex
![de-multiplex](https://raw.githubusercontent.com/kaiakz/rsync2os/master/docs/demux.jpg)

## How to use the demo?
### Use minio
1. install & run minio, you need to configure your setting of minio in the main.go.
2. go run main.go

# Reference
* https://git.samba.org/?p=rsync.git
* https://rsync.samba.org/resources.html
* https://github.com/openbsd/src/tree/master/usr.bin/rsync
* https://github.com/tuna/rsync
* https://github.com/sourcefrog/rsyn
* https://github.com/gilbertchen/acrosync-library
* https://github.com/boundary/wireshark/blob/master/epan/dissectors/packet-rsync.c
* https://tools.ietf.org/html/rfc5781