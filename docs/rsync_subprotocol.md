# Rsync subprotocol
A subset of rsync protocol without block checksum, designed for the compatibility of any version of standard rsync receiver/sender (client/server). 

## Introdution
In the rsync algorithm, rsync requires random access of files to do the block exchange. It will cause a high CPU and disk usage. `rsync subprotocol` simplifies the block exchange to avoid that. It can **work with sequential files**.

### **NOTICE** Although `rsync subprotocol` don't need to do rolling checksum, a whole-file MD4/MD5 checksum is still required to validate the transfer. We can compute MD5 checksums during receiving/sending files.

## Receiver Side
In the `rsync algorithm`, after the receiver reads the file list, it interates through each file in the list and handles three situations when processes each file:
1. same as remote file: skip
2. exists but was modified (has different size or last modification time): the receiver examines each file in blocks of a fixed size. Each block will be hashed twice (Adler-32 and MD4/MD5), and sent to the sender.
3. does not exist or empty: can't hash blocks of file, so just send a zero block to require the entire file.

But in `rsync subprotocol`, receiver do not support block checksum, so it won't send any hashes to the sender, but zero blocks instead. For a modified file, receiver always pretends `the file does not exist` to get the entire file from sender. **Just like you specify the argument `--whole-file` in standard rsync.**


## Sender Side
In the `rsync algorithm`, when receiver find a modfied file, it will send the file's block hashes to sender. Once accepted, the sender will also calculate the hashes of each file's blocks: 
1. If a block is matched, just send the index of the block to receiver so receiver will copy relative data from the file locally.
2. If a block is not matched, it means the block from receiver was modified. The block of the file will be sent as a stream of bytes, and receiver will copy it to file directly.

In `rsync subprotocol`, sender will ignore any block that comes from receiver so there are no matched blocks. For each file required by receiver, sender always sends the entire file. 