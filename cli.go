package main

type Cli struct {
	Src string
	Dest string
	// Flags -aDglnoprtvx
}

/*
Usage: rsync [OPTION]... SRC [SRC]... DEST
  or   rsync [OPTION]... SRC [SRC]... [USER@]HOST:DEST
  or   rsync [OPTION]... SRC [SRC]... [USER@]HOST::DEST
  or   rsync [OPTION]... SRC [SRC]... rsync://[USER@]HOST[:PORT]/DEST
  or   rsync [OPTION]... [USER@]HOST:SRC [DEST]
  or   rsync [OPTION]... [USER@]HOST::SRC [DEST]
  or   rsync [OPTION]... rsync://[USER@]HOST[:PORT]/SRC [DEST]
*/
//func ParseArgs() *Cli {
	//cli := &Cli{}

	//syncCommand := flag.NewFlagSet("sync", flag.ExitOnError)
	//recoverCommand := flag.NewFlagSet("recover", flag.ExitOnError)
	//lsCommand := flag.NewFlagSet("ls", flag.ExitOnError)
	//testCommand := flag.NewFlagSet("test", flag.ExitOnError)

//}