package networkUtils

import (
	"blaze/networkUtils/networkproto"
	"blaze/security"
	"bytes"
	"crypto/rsa"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"
)

func ListenRequest(port int, rsakey *rsa.PublicKey) error {
	serverConn, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		return err
	}

	channel := make(chan net.Conn, 10)

	for i := 0; i < 5; i++ {
		go connectionHandler(channel, rsakey)
	}

	for {
		connection, err := serverConn.Accept()
		if err != nil {
			log.Fatal(err)
		} else {
			channel <- connection
		}
	}
}

func sendChallenge(conn net.Conn, rsaKey *rsa.PublicKey) bool {
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
		return true
	}
	fmt.Println("challenge failed, either wrong key or connection timeout")
	return false
}

func connectionHandler(channel chan net.Conn, rsaKey *rsa.PublicKey) {
	for i := range channel {

		if !sendChallenge(i, rsaKey) {
			i.Close()
		} else {

		}
		i.Close()
	}
}

func udpListener(port int, address net.Addr) {
	packetConn, err := net.Listen("udp", ":"+strconv.Itoa(port))
	defer packetConn.Close()
	if err != nil {
		return
	}
	conn, _ := packetConn.Accept()
	defer conn.Close()
	if strings.Split(conn.RemoteAddr().String(), ":")[0] != strings.Split(address.String(), ":")[0] {
		conn.Close()
	}
	for {
		//TODO
	}
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

	WriteProto(conn, marshalled)
	challengeanswer := ReadProto(conn)

	return bytes.Equal(challengeanswer.GetHash(), hash), aeskey
}

func createACK(conn *net.TCPConn, blockNumber uint64) ([]byte, error) {
	ack := &networkproto.ACKNACK{
		MessageType: "ACK",
		ACK:         blockNumber,
	}
	return proto.Marshal(ack)
}

func createNACK(nonReceived []uint64) ([]byte, error) {
	nacks := &networkproto.ACKNACK{
		MessageType: "NACK",
		NACKs:       nonReceived,
	}

	return proto.Marshal(nacks)
}

type receivedChunksIndex struct {
	startIndex      uint64
	highestReceived uint64
	receivedIndex   []bool
	lastACK         uint64
	blockSize       uint32
}

func newIndex(size uint64, blockSize uint32) receivedChunksIndex {
	return receivedChunksIndex{0, 0, make([]bool, size), 0, blockSize}
}

func (index receivedChunksIndex) update(chunkid uint64) {
	index.receivedIndex[chunkid] = true
	if chunkid > index.highestReceived {
		index.highestReceived = chunkid
	}
	for i := index.startIndex; i < chunkid; i++ {
		if index.receivedIndex[i] == true {
			index.startIndex = i
		} else {
			return
		}
	}
}
