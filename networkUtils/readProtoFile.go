package networkUtils

import (
	"blaze/networkUtils/networkproto"
	"encoding/binary"
	"fmt"
	"net"

	"github.com/golang/protobuf/proto"
)

// ReadProto reads a from a connection and return a protobuf object
func ReadProto(conn *net.TCPConn) *networkproto.ACKNACK {
	buff := make([]byte, 4)
	bytesRead, err := conn.Read(buff)
	if err != nil || bytesRead != 4 {
		fmt.Println("Something went wrong when reading from the connection")
		return nil
	}
	sizeOfProtobuf := int(binary.LittleEndian.Uint32(buff))
	total := 0
	protoFromConn := make([]byte, 0)
	for total < sizeOfProtobuf {
		//recreate buff to be sure that it will never read more than the protobuf
		buff = make([]byte, int(sizeOfProtobuf-total))
		bytesRead, err = conn.Read(buff)
		total += bytesRead
		protoFromConn = append(protoFromConn, buff...)
		if err != nil {
			fmt.Println()
		}
	}
	answer := &networkproto.ACKNACK{}
	proto.Unmarshal(protoFromConn, answer)
	return answer
}

func ReadProtoEncrypted(conn *net.TCPConn, marshalled, key []byte) *networkproto.ACKNACK {
	// TODO: read length, read iv, decrypt payload , remove padding, return
	return nil
}