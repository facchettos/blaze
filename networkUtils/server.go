package networkUtils

import (
	"blaze/networkUtils/networkproto"
	"blaze/security"
	"bytes"
	"crypto/rsa"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"
)

// ListenRequest listens for incoming tcp connections
func ListenRequest(port int, blockSize uint64, packetSize int) error {
	rsakey, err := security.ParseRsaPublicKeyFromPubFile("rsa.pub")

	if err != nil {
		fmt.Println("couldn't read the rsa public key from file")
	}
	tcpAddres, _ := net.ResolveTCPAddr("tcp", ":"+strconv.Itoa(port))
	serverConn, err := net.ListenTCP("tcp", tcpAddres)
	if err != nil {
		return err
	}

	channel := make(chan *net.TCPConn, 10)

	for i := 0; i < 5; i++ {
		go connectionHandler(channel, rsakey, packetSize, blockSize)
	}

	for {
		connection, err := serverConn.AcceptTCP()
		if err != nil {
			log.Fatal(err)
		} else {
			channel <- connection
		}
	}
}

func sendChallenge(conn net.Conn, rsaKey *rsa.PublicKey) (bool, []byte) {
	aeskey := security.CreateAESKey()
	fmt.Println("key from server is : " + hex.EncodeToString(aeskey))
	aeshash := security.ComputeHash(aeskey)
	fmt.Println("hash from server is : " + hex.EncodeToString(aeshash))
	binarySize := make([]byte, 2)
	encryptedKey := security.EncryptWithPublicKey(aeskey, rsaKey)
	binary.LittleEndian.PutUint16(binarySize, uint16(len(encryptedKey)))
	conn.Write(binarySize)
	conn.Write(encryptedKey)
	//set the deadline to 5 seconds, passed this delay, the connection will fail
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	readbuffer := make([]byte, 32)
	_, err := conn.Read(readbuffer)
	//reset the deadline,
	conn.SetReadDeadline(time.Time{})
	if bytes.Equal(aeshash, readbuffer) && err == nil {
		fmt.Println("challenge success, they have the private key")
		return true, aeskey
	}
	fmt.Println("challenge failed, either wrong key or connection timeout")
	return false, nil
}

func connectionHandler(channel chan *net.TCPConn, rsaKey *rsa.PublicKey, packetSize int, blockSize uint64) {
	for i := range channel {
		fileName, fileSize := readFileNameAndSize(i)
		fileName = "testServer.jpg"
		fmt.Println("File name is : ", fileName, " file size is: ", fileSize)
		if success, key := sendChallenge(i, rsaKey); !success {
			i.Close()
		} else {
			var port uint32
			var portAsString string
			for {
				port = (rand.Uint32() % 60000) + 2000
				portAsString = strconv.Itoa(int(port))
				udpAddressTest, _ := net.ResolveUDPAddr("udp", ":"+portAsString)
				testConn, err := net.ListenUDP("udp", udpAddressTest)
				if err == nil {
					testConn.Close()
					break
				} else {
					testConn.Close()
				}

			}
			fmt.Println("the key is : ", hex.EncodeToString(key))
			distantIP := strings.Split(i.RemoteAddr().String(), ":")[0]
			go OpenUdp(packetSize, portAsString, fileSize, fileName, key, distantIP, blockSize, 1024, i)
			sendPort(i, port)

		}
	}
}

func readFileNameAndSize(conn *net.TCPConn) (string, uint64) {
	buff := make([]byte, 8)
	conn.Read(buff)
	toRead := binary.LittleEndian.Uint64(buff)
	packet := make([]byte, toRead)
	io.ReadFull(conn, packet)
	return string(packet[8:]), binary.LittleEndian.Uint64(packet[:8])
}

func handleConnRequest(publicKey *rsa.PublicKey, conn *net.TCPConn) (accepted bool, key []byte) {
	//create a random key
	firstrequest := ReadProto(conn)
	if firstrequest.GetMessageType() != "connrequest" {
		return false, nil
	}
	aeskey := security.CreateAESKey()
	//encrypt the aeskey with the public aeskey
	cipher := security.EncryptWithPublicKey(aeskey, publicKey)
	//compute the hash of the aeskey
	hash := security.ComputeHash(cipher)
	request := &networkproto.ACKNACK{
		MessageType: "connchallenge",
		ToDecrypt:   cipher,
	}
	marshalled, err := proto.Marshal(request)
	if err != nil {
		fmt.Println("error while creating protobuf connchallenge")
		return false, nil
	}

	WriteProto(conn, marshalled, nil)
	challengeanswer := ReadProto(conn)

	return bytes.Equal(challengeanswer.GetHash(), hash), aeskey
}

type receivedChunksIndex struct {
	lastWrite               uint64
	highestBuffered         uint64
	numberOfBufferedPackets uint64
	buff                    [][]byte
	buffSize                uint64
	blockSize               int
	dropHigherThanHighest   bool
	receivedFirst           bool
}

func sendPort(tcpconn *net.TCPConn, port uint32) {
	buff := make([]byte, 4)
	binary.LittleEndian.PutUint32(buff, port)
	tcpconn.Write(buff)
}
