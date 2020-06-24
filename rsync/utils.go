package rsync

import (
	"bytes"
	"strconv"
	"strings"
)

func SplitURIS(uri string) (string, int, string, string, error) {

	var host, module, path string
	var first = []byte(uri)
	var second []byte

	if strings.HasPrefix(uri, "rsync://") {
		/* rsync://host[:port]/module[/path] */
		first = first[8:]
		i := bytes.IndexByte(first, '/')
		if i == -1 {
			// No module name
			panic("No module name")
		}
		second = first[i+1:] //ignore '/'
		first = first[:i]
	} else {
		// Only for remote
		/* host::module[/path] */
		panic("No implement yet")
	}

	port := 873 // Default port: 873

	// Parse port
	i := bytes.IndexByte(first, ':')
	if i != -1 {
		var err error
		port, err = strconv.Atoi(string(first[i+1:]))
		if err != nil {
			// Wrong port
			panic("Wrong port")
		}
		first = first[:i]
	}
	host = string(first)

	// Parse path
	i = bytes.IndexByte(second, '/')
	if i != -1 {
		path = string(second[i:])
		second = second[:i]
	}
	module = string(second)

	return host, port, module, path, nil

}

// For rsync
func SplitURI(uri string) (string, string, string, error) {

	var address, module, path string
	var first = []byte(uri)
	var second []byte

	if strings.HasPrefix(uri, "rsync://") {
		/* rsync://host[:port]/module[/path] */
		first = first[8:]
		i := bytes.IndexByte(first, '/')
		if i == -1 {
			// No module name
			panic("No module name")
		}
		second = first[i+1:] //ignore '/'
		first = first[:i]
	} else {
		// Only for remote
		/* host::module[/path] */
		panic("No implement yet")
	}

	address = string(first)
	// Parse port
	i := bytes.IndexByte(first, ':')
	if i == -1 {
		address += ":873" // Default port: 873
	}

	// Parse path
	i = bytes.IndexByte(second, '/')
	if i != -1 {
		path = string(second[i:])
		second = second[:i]
	}
	module = string(second)

	return address, module, path, nil

}

func TrimPrepath(prepath string) string {
	//pre-path shouldn't use "/" as prefix, and must have a "/" suffix
	//pre-path can be: "xx", "xx/", "/xx", "/xx/", "", "/"
	ppath := prepath
	if !strings.HasSuffix(ppath, "/") {
		ppath += "/"
	}
	if strings.HasPrefix(ppath, "/") {
		ppath = ppath[1:]
	}
	return ppath
}
