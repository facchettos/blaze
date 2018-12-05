package networkUtils

import (
	"blaze/networkUtils/networkproto"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/golang/protobuf/proto"
)

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
		//and create file
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
	}

	if before != "" && after != "" {
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
