receiver选项 sender File List receiver块校验码 sender发送文件

# The File List
file list不仅包括了路径名, 还包含了拷贝模式, 所有者, 权限, 文件大小, mtime等属性。如果使用了"--checksum"选项，则还包括文件级的校验码。
 
# The Generator
generator进程将file list与本地目录树进行比较。如果指定了"--delete"选项，则在generator主功能开始前，它将首先识别出不在sender端的本地的文件(译者注：因为此generator为receiver端的进程)，并在recevier端删除这些文件。

然后generator将开始它的主要工作，它会从file list中一个文件一个文件地向前处理。每个文件都会被检测以确定它是否需要跳过。如果文件的mtime或大小不同，最常见的文件操作模式不会忽略它。如果指定了"--checksum"选项，则会生成文件级别的checksum并做比较。目录、块设备和符号链接都不会被忽略。缺失的目录在目标上也会被创建。

如果文件不被忽略，所有目标路径下已存在的文件版本将作为基准文件(basis file)(译者注：请记住这个词，它贯穿整个rsync工作机制)，这些基准文件将作为数据匹配源，使得sender端可以不用发送能匹配上这些数据源的部分(译者注：从而实现增量传输)。为了实现这种远程数据匹配，将会为basis file创建块校验码(block checksum)，并放在文件索引号(文件id)之后立即发送给sender端。如果指定了"--whole-file"选项，则对文件列表中的所有文件都将发送空的块校验码，使得rsync强制采用全量传输而非增量传输。(译者注：也就是说，generator每计算出一个文件的块校验码集合，就立即发送给sender，而不是将所有文件的块校验码都计算完成后才一次性发送)

每个文件被分割成的块的大小以及块校验和的大小是根据文件大小计算出来的(译者注：rsync命令支持手动指定block size)。


https://github.com/boundary/wireshark/blob/master/epan/dissectors/packet-rsync.c