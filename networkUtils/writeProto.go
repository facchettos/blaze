package networkUtils

import (
	"encoding/binary"
	"net"
)

// WriteProto writes proto object's size and the object itself
func WriteProto(conn *net.TCPConn, marshalled []byte) {
	bs := make([]byte, 4)
	binary.LittleEndian.PutUint32(bs, uint32(len(marshalled)))
	conn.Write(bs)
	conn.Write(marshalled)
}

func WriteProtoEncrypted(conn *net.TCPConn, marshalled []byte, key []byte) {
	// TODO:  encrypt all protobuf + padding + number of padding
	//        characters with key, send first size(ciphertext + iv),
	//        then payload (number of padding characters +proto object + padding)
}
