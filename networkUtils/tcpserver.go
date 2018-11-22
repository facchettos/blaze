package networkUtils

import (
	"blaze/networkUtils/networkproto"
	"blaze/security"
	"bytes"
	"crypto/rsa"
	"fmt"
	"log"
	"net"
	"strconv"

	"github.com/golang/protobuf/proto"
)

func ListenRequest(port int) error {
	serverConn, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		return err
	}

	channel := make(chan net.Conn, 10)

	for i := 0; i < 5; i++ {
		go connectionHandler(channel)
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

func connectionHandler(channel chan net.Conn) {
	for i := range channel {
		i.Write([]byte("toto\n"))
		i.Close()
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
