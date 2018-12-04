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
	"io/ioutil"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"
)

// ListenRequest listens for incoming tcp connections
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

func createACK(blockNumber uint64) ([]byte, error) {
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

func receiveUDP(packetSize, bufferSize int, conn net.Conn, chanOut chan []byte) {
	for {
		buff := make([]byte, bufferSize)
		n, err := conn.Read(buff)
		if err == nil {
			chanOut <- buff[:n]
		} else {
			fmt.Println("something went wrong with reading the packet")
			close(chanOut)
			break
		}
	}
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

func (buffer receivedChunksIndex) packetsToWrite(packet []byte) [][]byte {
	if getPacketNumber(packet) == 0 {
		buffer.receivedFirst = true
	}
	toWrite := make([][]byte, 0)

	buffer.Buffer(packet)
	buffer.numberOfBufferedPackets++

	for {
		toWrite = append(toWrite, buffer.buff[buffer.lastWrite+1])
		buffer.buff[buffer.lastWrite+1] = nil
		buffer.numberOfBufferedPackets--
		buffer.lastWrite++
		if buffer.numberOfBufferedPackets < buffer.buffSize {
			buffer.dropHigherThanHighest = false
		}
		if buffer.lastWrite%uint64(buffer.blockSize) == 0 && buffer.lastWrite != 0 {
			//TODO send ACK for block
			ack, err := createACK(buffer.lastWrite - uint64(buffer.blockSize))
			if err == nil {
				fmt.Println(ack)
				fmt.Println("created ack")
				//TODO send ack
			} else {
				fmt.Println("problem while marshalling ack")
			}
		}

		// if packet is last of file, return
		if buffer.lastWrite == uint64(len(buffer.buff))-1 {
			return toWrite
		}

		//if next packet hasnt arrived yet, return
		if buffer.buff[buffer.lastWrite+1] == nil {
			return toWrite
		}
	}
}

func (buffer receivedChunksIndex) Buffer(packet []byte) {

	if !(getPacketNumber(packet) > buffer.highestBuffered && buffer.dropHigherThanHighest) {
		buffer.buff[getPacketNumber(packet)] = packet[:len(packet)-8]
		buffer.numberOfBufferedPackets++
		if buffer.numberOfBufferedPackets >= buffer.buffSize {
			buffer.dropHigherThanHighest = true
		}
		fmt.Println(buffer.highestBuffered)
		//TODO check for nack
	} else {
		//send NACK for missing packets
		nonReceived := make([]uint64, 0)
		for i := buffer.lastWrite; i < buffer.highestBuffered; i++ {
			if buffer.buff[i] == nil {
				nonReceived = append(nonReceived, uint64(i))
			}
		}
		nack, err := createNACK(nonReceived)
		if err == nil {
			fmt.Println(nack)
			fmt.Println("created nack")
			//TODO send NACK
		} else {
			fmt.Println("error while marshalling nack")
		}

	}
}

func getPacketNumber(packet []byte) uint64 {

	return binary.LittleEndian.Uint64(packet[len(packet)-8:])
}

func writer(chanIn chan []byte, chanOut chan bool, fileSize uint64, pipeOut *io.PipeWriter) {
	buff := make([]bool, fileSize)
	var lastWrite uint64
	//last write takes the max value of uint64 so uint+1 == 0
	lastWrite = ^uint64(0)
	for packet := range chanIn {
		packetNumber := getPacketNumber(packet)
		before, after := findFileBeforeAfter(packetNumber)
		if buff[packetNumber] == false {
			if packetNumber != lastWrite+1 {
				writeToDisk(packet[:len(packet)-8], packetNumber, before, after)
			} else {
				lastWrite = writeToPipe(pipeOut, packet[:len(packet)-8], after)
			}
			//HANDLE NACK / ACK HERE
			buff[packetNumber] = true
			if lastWrite == fileSize-1 {
				pipeOut.Close()
				//chanout notifies higher level routine to shutdown connection and channels
				chanOut <- true
			}
		}
	}
}

func writeToPipe(pipeOut *io.PipeWriter, packet []byte, after string) uint64 {
	pipeOut.Write(packet[:len(packet)-8])
	var lastWrite uint64
	lastWrite = getPacketNumber(packet)
	if after != "" {
		afterFile, _ := ioutil.ReadFile(after)
		pipeOut.Write(afterFile)
		lastWrite, _ = strconv.ParseUint(strings.Split(after, "-")[1], 10, 64)
		os.Remove(after)
	}
	return lastWrite
}

func writeToDisk(packet []byte, packetNumber uint64, before string, after string) {
	afterArray := strings.Split(after, "-")
	suffix := afterArray[len(afterArray)-1]
	if after == "" {
		suffix = strconv.FormatUint(packetNumber, 36)
	}
	beforeArray := strings.Split(before, "-")
	prefix := beforeArray[0]
	var f *os.File
	var err error
	if before != "" {
		f, err = os.OpenFile(before, os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			panic(err)
		}

	} else {
		//if nothing is provide for a before file, change prefix to packet number
		before = strconv.FormatUint(packetNumber, 36)
		f, _ = os.OpenFile(before+"-"+
			suffix, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)

	}
	_, err = f.Write(packet)
	if err != nil {
		fmt.Println("error while writing to file")
	}
	if after != "" {
		fAfter, _ := os.Open(after)

		io.Copy(f, fAfter)
		err = fAfter.Close()
		if err != nil {
			fmt.Println(err)
		}
		os.Remove(after)
		err = f.Close()
		if err != nil {
			fmt.Println("error closing file " + f.Name())
		}
		fmt.Println("renaming file to " + prefix + "-" + suffix)
		os.Rename(before, prefix+"-"+
			suffix)
	}

	if before != "" && after == "" {
		os.Rename(before, strings.Split(before, "-")[0]+"-"+
			strconv.FormatUint(packetNumber, 36))
	}
}

func findFileBeforeAfter(packetNumber uint64) (before, after string) {
	files, err := ioutil.ReadDir("./")
	before, after = "", ""
	if err != nil {
		log.Fatal(err)
	}

	for _, f := range files {

		if strings.Compare(strings.Split(f.Name(), "-")[len(strings.Split(f.Name(), "-"))],
			strconv.FormatUint(packetNumber-1, 36)) == 0 && packetNumber != 0 {

			before = f.Name()
		} else if strings.Compare(strings.Split(f.Name(), "-")[0],
			strconv.FormatUint(packetNumber+1, 36)) == 0 && packetNumber != ^uint64(0) {

			after = f.Name()
		}
		if before != "" && after != "" {
			break
		}
	}
	return before, after
}
