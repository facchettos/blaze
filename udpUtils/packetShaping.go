package udpUtils

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
)

func TestBuffer() {
	addr, err := net.ResolveUDPAddr("udp", ":8089")
	conn, err := net.DialUDP("udp", nil, addr)

	err = conn.SetWriteBuffer(0)
	if err != nil {
		log.Print(err)
	}
	myint, err := conn.Write([]byte("hello my little friend how are you doing over here"))
	fmt.Println(myint)
	myint, err = conn.Write([]byte("hello my little friend how are you doing over here"))
	fmt.Println(myint)
	myint, err = conn.Write([]byte("hello my little friend how are you doing over here"))
	fmt.Println(myint)
	myint, err = conn.Write([]byte("hello my little friend how are you doing over here"))
	fmt.Println(myint)

	if err != nil {
		fmt.Println(err)
		return
	} else {
		fmt.Println("err is nil")
	}

}

//ExtractSerialNumber extracts the serial number of the packet
func ExtractSerialNumber(serialNumber []byte) uint64 {
	return binary.LittleEndian.Uint64(serialNumber)
}

//SplitPacket returns the 8 first bytes (serial number) and then the rest of the packet
func SplitPacket(array []byte) (serialNumber, payload []byte) {
	return array[:8], array[8:]
}

//GetBlockNumber extracts the block number (7 first bytes)
func GetBlockNumber(serialNumber []byte) uint64 {
	return binary.LittleEndian.Uint64(serialNumber) - binary.LittleEndian.Uint64(serialNumber[7:7])
}
