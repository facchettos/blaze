package networkUtils

import (
	udpUtils "blaze/networkUtils/udpUtils"
	"blaze/security"
	"crypto/aes"
	"crypto/rsa"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

//CLIENT TAKES care of the sending. a user could trigger a
// file retrieval from the client application
// and then would be considered as the server

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
func TCPClient(fileName string,
	packetSize uint64, address string, blockSize uint64, maxBuff uint64) {
	conn, err := EstablishConnection(address)
	sendFileInfos(fileName, conn, packetSize)
	if err != nil {
		fmt.Println("impossible to connect to remote host")
		return
	}
	privateKey := security.ReadPrivateKeyFromFile("rsa.pem")
	aesKey := doChallenge(privateKey, conn)
	fmt.Println("the key is: " + hex.EncodeToString(aesKey))
	buff := make([]byte, 4)
	_, err = conn.Read(buff)
	if err != nil {
		fmt.Println("reading from the connection to get the udp port failed")
	}

	udpPort := binary.LittleEndian.Uint32(buff)
	fmt.Println("the udp port is: ", udpPort)

	remoteAddressUDP, _ := net.ResolveUDPAddr("udp", strings.Split(address, ":")[0]+":"+strconv.Itoa(int(udpPort)))
	udpConn, err := net.DialUDP("udp", nil, remoteAddressUDP)
	if err != nil {
		fmt.Println(err)
	}
	udpConn.SetWriteBuffer(0)
	if err != nil {
		fmt.Println("error while trying to dial server")
		fmt.Println(err)
	}
	var keyAsArray [aes.BlockSize]byte
	copy(keyAsArray[:], aesKey)
	SendFile(fileName, keyAsArray, blockSize, udpConn, packetSize, conn, maxBuff)
	// SendChunksToChannel(filename string, channel chan []byte, buffsize int, key [16]byte)
	//TODO open udp dial with the port
	// add channel, and write files to channel
	// readFrom file and encrypt it
	//ADD loop for ACK
}

func SendFile(fileName string, key [aes.BlockSize]byte,
	blockSize uint64, udpConn *net.UDPConn, packetSize uint64,
	conn net.Conn, maxBuff uint64) {

	fmt.Println("file size is: ", getFileSize(fileName, packetSize))
	packetChanSender := make(chan []byte)
	orderChan := make(chan order, 100)
	chanToSender := make(chan []byte)
	go udpUtils.SendLoop(chanToSender, udpConn, time.Millisecond, time.Millisecond)
	go SendChunksToChannel(fileName, packetChanSender, int(packetSize), key)

	go packetBuffHandler(orderChan, packetChanSender, chanToSender,
		getFileSize(fileName, packetSize), maxBuff, blockSize)

	for {
		messageLength := make([]byte, 8)
		if conn != nil {
			_, err := conn.Read(messageLength)
			if err != nil {

				fmt.Println(err)
				fmt.Println("error trying to read message length")
				conn.Close()
				break

			}

			size := binary.LittleEndian.Uint64(messageLength)
			messageBuff := make([]byte, size)
			_, err = io.ReadFull(conn, messageBuff)
			if err != nil {
				fmt.Println("error reading from the conn to get the message")
			}

			orderProto := byteToProto(messageBuff)
			orderObject := convertProtoToOrder(orderProto, blockSize)
			orderChan <- orderObject
			if orderObject.orderType == done {
				close(orderChan)
				close(chanToSender)
				fmt.Println("receiving side sent a 'done order'")
				break
			}
			if err != nil {
				conn.Close()
				break
			}
		}
	}
	fmt.Println("SENDFILE DONE")
}

func sendFileInfos(fileName string, conn *net.TCPConn, packetSize uint64) {
	tcpPacketSize := len(fileName)
	binarySize := make([]byte, 8)
	binary.LittleEndian.PutUint64(binarySize, uint64(tcpPacketSize+8))
	//send the number of bytes to read
	conn.Write(binarySize)
	//send the filesize
	fileSizeBinary := make([]byte, 8)
	binary.LittleEndian.PutUint64(fileSizeBinary, getFileSize(fileName, packetSize))
	conn.Write(fileSizeBinary)

	//send the filename
	conn.Write([]byte(fileName))
}

func getFileSize(fileName string, packetSize uint64) uint64 {
	fi, _ := os.Stat(fileName)
	actualSize := (packetSize - (packetSize % 16))
	// get the size
	size := fi.Size()
	if uint64(size)%actualSize == 0 {
		return uint64(size) / actualSize
	}

	return (uint64(size) / actualSize) + 1

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
	// fmt.Print("number to read is: ")
	// fmt.Println(numberToRead)
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
