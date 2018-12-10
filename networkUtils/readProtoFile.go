package networkUtils

import (
	"blaze/networkUtils/networkproto"
	"encoding/binary"
	"fmt"
	"net"

	"github.com/golang/protobuf/proto"
)

// ReadProto reads a from a connection and return a protobuf object
func ReadProto(conn net.Conn) *networkproto.ACKNACK {
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
	err = proto.Unmarshal(protoFromConn, answer)
	if err != nil {
		return nil
	}
	return answer
}

func byteToProto(protobyte []byte) *networkproto.ACKNACK {
	res := &networkproto.ACKNACK{}
	err := proto.Unmarshal(protobyte, res)
	if err != nil {
		fmt.Println("error unmarshalling protobuf")
		return nil
	}
	return res
}

func convertProtoToOrder(proto *networkproto.ACKNACK, blockSize uint64) order {
	orderFromProto := order{}
	if proto.GetMessageType() == "NACK" {
		orderFromProto.orderType = send
		orderFromProto.packetNumber = proto.GetNACKs()

	} else if proto.GetMessageType() == "done" {
		orderFromProto.orderType = done
	} else if proto.GetMessageType() == "ACK" {
		orderFromProto.orderType = remove
		orderFromProto.from = proto.GetACK()
		orderFromProto.to = orderFromProto.from + blockSize
	}
	return orderFromProto
}
