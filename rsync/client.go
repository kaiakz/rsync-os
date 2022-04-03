package rsync

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"strings"
)

/* As a Client, we need to:
1. connect to server by socket or ssh
2. handshake: version, args, ioerror
	PS: client always sends exclusions/filter list
3. construct a Receiver or a Sender, then excute it.
*/

// TODO: passes more arguments: cmd
// Connect to rsync daemon
func SocketClient(storage FS, address string, module string, path string) (SendReceiver, error) {
	skt, err := net.Dial("tcp", address)
	if err != nil {
		return nil, err
	}

	conn := &Conn{
		writer:    skt,
		reader:    skt,
		bytespool: make([]byte, 8),
	}

	/* HandShake by socket */
	// send my version
	_, err = conn.Write([]byte(RSYNC_VERSION))
	if err != nil {
		return nil, err
	}

	// receive server's protocol version and seed
	versionStr, _ := readLine(conn)

	// recv(version)
	var remoteProtocol, remoteProtocolSub int
	_, err = fmt.Sscanf(versionStr, "@RSYNCD: %d.%d", remoteProtocol, remoteProtocolSub)
	if err != nil {
		// FIXME: (panic)type not a pointer: int
		//panic(err)
	}
	log.Println(versionStr)

	buf := new(bytes.Buffer)

	// send mod name
	buf.WriteString(module)
	buf.WriteByte('\n')
	_, err = conn.Write(buf.Bytes())
	if err != nil {
		return nil, err
	}
	buf.Reset()

	// Wait for '@RSYNCD: OK'
	for {
		res, err := readLine(conn)
		if err != nil {
			return nil, err
		}
		log.Print(res)
		if strings.Contains(res, RSYNCD_OK) {
			break
		}
	}

	// Send arguments
	buf.Write([]byte(SAMPLE_ARGS))
	buf.Write([]byte(module))
	buf.Write([]byte(path))
	buf.Write([]byte("\n\n"))
	_, err = conn.Write(buf.Bytes())
	if err != nil {
		return nil, err
	}

	// read int32 as seed
	seed, err := conn.ReadInt()
	if err != nil {
		return nil, err
	}
	log.Println("SEED", seed)

	// HandShake OK
	log.Println("Handshake completed")

	// Begin to demux
	conn.reader = NewMuxReader(conn.reader)

	// As a client, we need to send filter list
	err = conn.WriteInt(EXCLUSION_END)
	if err != nil {
		return nil, err
	}

	// TODO: Sender

	return &Receiver{
		conn:    conn,
		module:  module,
		path:    path,
		seed:    seed,
		storage: storage,
	}, nil
}

func SocketRecvClient(storage FS, address string, module string, path string, callback Callback) (SendReceiver, error) {
	skt, err := net.Dial("tcp", address)
	if err != nil {
		return nil, err
	}

	conn := &Conn{
		writer:    skt,
		reader:    skt,
		bytespool: make([]byte, 8),
	}

	/* HandShake by socket */
	// send my version
	_, err = conn.Write([]byte(RSYNC_VERSION))
	if err != nil {
		return nil, err
	}

	// receive server's protocol version and seed
	versionStr, _ := readLine(conn)

	// recv(version)
	var remoteProtocol, remoteProtocolSub int
	_, err = fmt.Sscanf(versionStr, "@RSYNCD: %d.%d", remoteProtocol, remoteProtocolSub)
	if err != nil {
		// FIXME: (panic)type not a pointer: int
		//panic(err)
	}
	log.Println(versionStr)

	buf := new(bytes.Buffer)

	// send mod name
	buf.WriteString(module)
	buf.WriteByte('\n')
	_, err = conn.Write(buf.Bytes())
	if err != nil {
		return nil, err
	}
	buf.Reset()

	// Wait for '@RSYNCD: OK'
	for {
		res, err := readLine(conn)
		if err != nil {
			return nil, err
		}
		log.Print(res)
		if strings.Contains(res, RSYNCD_OK) {
			break
		}
	}

	// Send arguments
	buf.Write([]byte(SAMPLE_ARGS))
	buf.Write([]byte(module))
	buf.Write([]byte(path))
	buf.Write([]byte("\n\n"))
	_, err = conn.Write(buf.Bytes())
	if err != nil {
		return nil, err
	}

	// read int32 as seed
	seed, err := conn.ReadInt()
	if err != nil {
		return nil, err
	}
	log.Println("SEED", seed)

	// HandShake OK
	log.Println("Handshake completed")

	// Begin to demux
	conn.reader = NewMuxReader(conn.reader)

	// As a client, we need to send filter list
	err = conn.WriteInt(EXCLUSION_END)
	if err != nil {
		return nil, err
	}

	// TODO: Sender

	return &Receiver{
		conn:    conn,
		module:  module,
		path:    path,
		seed:    seed,
		storage: storage,
		callback: callback,
	}, nil
}

// Connect to sshd, and start a rsync server on remote
func SshClient(storage FS, address string, module string, path string, options map[string]string) (SendReceiver, error) {
	// TODO: build args

	ssh, err := NewSSH(address, "", "", "rsync --server --sender -l -p -r -t")
	if err != nil {
		return nil, err
	}
	conn := &Conn{
		writer:    ssh,
		reader:    ssh,
		bytespool: make([]byte, 8),
	}

	// Handshake
	lver, err := conn.ReadInt()
	if err != nil {
		return nil, err
	}

	rver, err := conn.ReadInt()
	if err != nil {
		return nil, err
	}

	seed, err := conn.ReadInt()
	if err != nil {
		return nil, err
	}

	// TODO: Sender

	return &Receiver{
		conn:    conn,
		module:  module,
		path:    path,
		seed:    seed,
		lVer:    lver,
		rVer:    rver,
		storage: storage,
	}, nil
}
