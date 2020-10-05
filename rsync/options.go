package rsync

type Options struct {
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
