package networkUtils

import (
	"blaze/security"
	streams "blaze/streamsutils"
	"crypto/aes"
	"crypto/rsa"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"time"
)

//EstablishConnection returns a tcp connection to the address given
func EstablishConnection(address string) (*net.TCPConn, error) {
	tcpaddress, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		return nil, errors.New("impossible to resolve address")
	}
	conn, err := net.DialTCP("tcp", nil, tcpaddress)

	if err != nil {
		return nil, errors.New("Error trying to open connection to " + address)
	}
	return conn, nil
}

// TCPClient instantiate the tcp connection with a daemon
func TCPClient(privateKey *rsa.PrivateKey) {
	conn, err := EstablishConnection("localhost:8080")
	if err != nil {
		fmt.Println("impossible to connect to remote host")
		return
	}
	aesKey := doChallenge(privateKey, conn)
	fmt.Println("the key is: " + hex.EncodeToString(aesKey))
	buff := make([]byte, 2)
	_, err = conn.Read(buff)
	if err != nil {
		fmt.Println("reading from the connection to get the udp port failed")
	}
	udpPort := binary.LittleEndian.Uint16(buff)
	fmt.Println(udpPort)
	fileChannel := make(chan []byte, 5000)
	fmt.Println(len(fileChannel))
	//TODO open udp dial with the port
	// add channel, and write files to channel
	// readFrom file and encrypt it
}

func doChallenge(privateKey *rsa.PrivateKey, conn net.Conn) []byte {

	buffForSize := make([]byte, 2)
	n, err := conn.Read(buffForSize)
	if err != nil || n != 2 {
		return nil
	}
	numberToRead := binary.LittleEndian.Uint16(buffForSize)
	err = conn.SetReadDeadline(time.Now().Add(time.Second * 2))
	if err != nil {
		return nil
	}
	buff := make([]byte, 0)
	tempBuff := make([]byte, numberToRead)
	bytesRead := 0
	fmt.Print("number to read is: ")
	fmt.Println(numberToRead)
	for uint16(bytesRead) != numberToRead {
		n, err = conn.Read(tempBuff)
		bytesRead += n
		fmt.Println(bytesRead)
		if err != nil {
			fmt.Println("problem while reading from the connection")
			return nil
		}
		buff = append(buff, tempBuff...)
		tempBuff = make([]byte, numberToRead-uint16(bytesRead))
	}

	errDeadline := conn.SetReadDeadline(time.Time{})

	if errDeadline != nil {
		return nil
	}

	if err != nil || uint16(n) != numberToRead {
		return nil
	}

	aeskey := security.DoChallenge(buff[:n], privateKey)
	fmt.Println("decryption gave: " + hex.EncodeToString(aeskey))
	aeshash := security.ComputeHash(aeskey)
	fmt.Println("hash from client is: " + hex.EncodeToString(aeshash))
	_, err = conn.Write(aeshash)
	if err != nil {
		return nil
	}
	return aeskey
}

func sendChunksToChannel(filename string, channel chan []byte, buffsize int, key [aes.BlockSize]byte) {
	// make buffsize to be a multiple of 16, to ensure that encrypted packet does fit into it
	buffsize16 := buffsize - (buffsize % 16)
	pr, pw := io.Pipe()
	pr2, pw2 := io.Pipe()
	pr3, pw3 := io.Pipe()
	go streams.FilePiper(filename, pw)
	go streams.PacketGenerator(pr, pw2, buffsize16)
	go streams.PacketEncryptor(pr2, pw3, key[:])
	for {
		buff := make([]byte, buffsize16)
		n, err := io.ReadFull(pr3, buff)
		channel <- buff
		if err != nil || n < len(buff) {
			return
		}
	}
}

type order struct {
	orderType    uint16
	packetNumber []uint64
	from         uint64
	to           uint64
}

const send = 1
const remove = 2
const done = 3

type sentChunk struct {
	startIndex      uint64
	highestReceived uint64
	receivedIndex   []bool
	lastACK         uint64
	blockSize       uint32
}

//CHAN OUT (to packetsender) HAS TO BE UNBUFFERED
func packetBuffHandler(
	orders chan order,
	packets chan []byte,
	chanOut chan []byte,
	numbOfPackets uint64,
	maxBuff uint64) {

	packetBuff := make([][]byte, numbOfPackets)
	var packetIndex uint64
	var numberOfBufferedPackets uint64
	var firstPacketIndex uint64
	packetIndex = 0
	numberOfBufferedPackets = 0
	firstPacketIndex = 0

Loop:
	for {
		select {
		case order := <-orders:
			if order.orderType == send {
				for _, n := range order.packetNumber {
					chanOut <- packetBuff[n]
				}
			} else if order.orderType == remove {
				for i := order.from; i <= order.to; i++ {
					packetBuff[i] = nil
					numberOfBufferedPackets++
				}
				if order.from == firstPacketIndex {
					for packetBuff[firstPacketIndex] == nil {
						firstPacketIndex++
					}
				}
			} else if order.orderType == done {
				break Loop
			}

			fmt.Println(order)
		default:
			if numberOfBufferedPackets <= maxBuff {
				packet := <-packets
				packetBuff[packetIndex] = packet
				packetIndex++
				numberOfBufferedPackets++
				chanOut <- packet
			}
		}
	}
}
