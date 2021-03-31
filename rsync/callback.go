package rsync

type Callback interface {
	OnRequest(remotefiles FileList, list []int) []int	// Receiver: Choose which files you want to download
	OnDelete(localfiles FileList, list []int) []int	// Receiver: Choose which files you want to delete
}

type SimpleCallback struct { }

// Default callback
func (s SimpleCallback) OnRequest(remotefiles FileList, list []int) []int {
	return list		// Do nothing
}

func (s SimpleCallback) OnDelete(localfiles FileList, list []int) []int {
	return list		// Do nothing
}

