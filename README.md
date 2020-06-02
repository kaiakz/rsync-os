# rsync2os
## rsync the remote files to your object storage

# HandShake
rysnc2os uses rsync protocol 27. It sends the arguments "--server--sender-l-p-r-t" to the remote rsyncd.

# The File List
According the arguments rsync2os sent, the file list will contain path, size, modified time & mode.
 
# The Generator


# Reference
* https://git.samba.org/?p=rsync.git
* https://github.com/openbsd/src/tree/master/usr.bin/rsync
* https://github.com/tuna/rsync
* https://github.com/sourcefrog/rsyn
* https://github.com/gilbertchen/acrosync-library
* https://github.com/boundary/wireshark/blob/master/epan/dissectors/packet-rsync.c