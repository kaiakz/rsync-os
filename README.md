# rsync2os
## rsync the remote files to your object storage

![client](https://github.com/kaikaz/rsync2os/blob/master/docs/client.jpg)

# HandShake
rysnc2os uses rsync protocol 27. It sends the arguments "--server--sender-l-p-r-t" to the remote rsyncd.

# The File List
According the arguments rsync2os sent, the file list should contain path, size, modified time & mode.
 
# Request the file
rsync2os always saves the file list in its database(local file list). rsync2os doesn't compare each file with the file list from remote server(remote file list), but the latest local file list. If the file in the local file list has different size, modified time, or doesn't exist, rsync2os will download the whole file(without [block exchange](https://github.com/kristapsdz/openrsync#block-exchange)). To to do that, rsync2os sends the empty block checksum with the file's index in the remote file list. 

# Download the file
The rsync server sends the entire file as a stream of bytes.

# Multiplex & De-Multiplex


# How to use the demo?
1. install & run minio, you can configure in WriteOS
2. go run main.go

# Reference
* https://git.samba.org/?p=rsync.git
* https://github.com/openbsd/src/tree/master/usr.bin/rsync
* https://github.com/tuna/rsync
* https://github.com/sourcefrog/rsyn
* https://github.com/gilbertchen/acrosync-library
* https://github.com/boundary/wireshark/blob/master/epan/dissectors/packet-rsync.c