package rsync

type Attribs struct {
	sender     bool // --sender
	server     bool // --server
	recursive  bool // -r
	dryRun     bool // -n
	hasModTime bool // -t
	hasPerms   bool // -p
	hasLinks   bool // -l
	hasGID     bool // -g
	hasUID     bool // -u
}

func (a *Attribs) Marshal() []byte {
	//"--server\n--sender\n-l\n-p\n-r\n-t\n.\n"
	args := make([]byte, 0, 64)
	if a.server {
		args = append(args, []byte("--server\n")...)
	}
	if a.sender {
		args = append(args, []byte("--sender\n")...)
	}
	if a.recursive {
		args = append(args, []byte("-r\n")...)
	}
	if a.hasModTime {
		args = append(args, []byte("-t\n")...)
	}
	if a.hasLinks {
		args = append(args, []byte("-l\n")...)
	}
	if a.hasPerms {
		args = append(args, []byte("-p\n")...)
	}
	if a.hasGID {
		args = append(args, []byte("-g\n")...)
	}
	if a.hasUID {
		args = append(args, []byte("-u\n")...)
	}
	return args
}