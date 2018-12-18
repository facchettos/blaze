package networkUtils

import (
	"blaze/networkUtils/networkproto"
	"blaze/security"
	"encoding/binary"
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

func OpenUdp(packetSize int, port string, fileSize uint64, fileName string,
	key []byte, authIP string, blockSize, treshold uint64, TCPconn net.Conn) {

	packetChan := make(chan []byte, 10)
	done := make(chan bool)
	address, err := net.ResolveUDPAddr("udp", ":"+port)

	if err != nil {
		fmt.Println(err)
		return
	}
	pr, pw := io.Pipe()
	pc, err := net.ListenUDP("udp", address)
	if err != nil {
		fmt.Println(err)
		return
	}
	go writer(packetChan, done, fileSize, pw, blockSize, treshold, TCPconn)

	go func() {
		result, _ := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE, 0600)
		defer result.Close()
		security.StreamReaderDecrypt(pr, result, key[:])

	}()

	receiveUDP(packetSize, *pc, packetChan, done, authIP)

	TCPconn.Close()
	// time.Sleep(time.Second)
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

func receiveUDP(packetSize int, conn net.UDPConn, chanOut chan []byte, doneChan chan bool, incIP string) {
	for {
		buff := make([]byte, packetSize)
		err := conn.SetReadDeadline(time.Now().Add(time.Millisecond * 10))
		if err != nil {
			fmt.Println(err)
		}
		n, address, err := conn.ReadFrom(buff)
		// fmt.Println(address)
		if address != nil {
			//if incoming ip is different than authorized , read next packet
			ip := strings.Split(address.String(), ":")
			if ip[0] != incIP {
				continue
			}
		}
		if err == nil {
			chanOut <- buff[:n]
		} else if err, ok := err.(net.Error); ok && err.Timeout() {
			//if the error is a timeout, check if we have all chunk and break if we do
			select {
			case <-doneChan:
				fmt.Println("done receiving")
				return
			default:
			}
		} else {
			fmt.Println("something went wrong with reading the packet")
			close(chanOut)
			break
		}
		//if receiving routing has all the chunks, returns, else continue loop
	}
}

func getPacketNumber(packet []byte) uint64 {
	return binary.LittleEndian.Uint64(packet[len(packet)-8:])
}

func checkFullBlock(received []bool, startingPoint, blockSize uint64) (bool, []uint64) {
	toResend := make([]uint64, 0)
	for i := startingPoint; i < startingPoint+blockSize; i++ {
		if !received[i] {
			toResend = append(toResend, i)
		}
	}
	if len(toResend) == 0 {
		return true, nil
	}
	return false, toResend

}

func sendACKORNACK(received []bool, startingPoint, blockSize uint64, conn net.Conn) bool {
	sendAck, nacks := checkFullBlock(received, startingPoint, blockSize)
	length := make([]byte, 8)
	var toSend []byte
	if sendAck {
		fmt.Println("sending ack")
		fmt.Println(startingPoint / blockSize)
		toSend, err := createACK((startingPoint / blockSize) * blockSize)
		if err != nil {
			fmt.Println("error while creating ack")
		}
		binary.LittleEndian.PutUint64(length, uint64(len(toSend)))
		if conn != nil {
			conn.Write(length)
			conn.Write(toSend)
		}
		return true
	} else {
		fmt.Println("sending nack")
		toSend, _ = createNACK(nacks)
		binary.LittleEndian.PutUint64(length, uint64(len(toSend)))
		if conn != nil {
			conn.Write(length)
			conn.Write(toSend)
		}
		return false
	}
}

func firstNotAcked(acked []bool, start, end uint64) []uint64 {
	res := make([]uint64, 0)
	for i := start; i < end; i++ {
		if acked[i] == false {
			res = append(res, uint64(i))
		}
	}
	return res
}

func writer(chanIn chan []byte, chanOut chan bool,
	fileSize uint64, pipeOut *io.PipeWriter, blockSize, treshold uint64, conn net.Conn) {

	buff := make([]bool, fileSize)
	blockACKED := make([]bool, fileSize/blockSize)
	var lastWrite uint64
	//last write takes the max value of uint64 so uint+1 == 0
	lastWrite = ^uint64(0)
	nextCheck := blockSize
	var start uint64

	for packet := range chanIn {
		packetNumber := getPacketNumber(packet)
		fmt.Println("packet number is :", packetNumber)
		// fmt.Println("packetnumber: ", packetNumber)
		// fmt.Println("udp packet received has a size of ", len(packet))
		//to prevent a small packet reording from sending nack, a threshold is introduced
		// only one ACK/NACK per block is sent
		// fmt.Println("packet number is :", packetNumber)

		if buff[packetNumber] == false {
			if (packetNumber%blockSize >= treshold && packetNumber >= nextCheck) ||
				packetNumber == fileSize-1 {

				nextCheck += blockSize

				shouldIncrement := true
				for i := start; i < packetNumber/blockSize; i++ {
					if blockACKED[i] == false {
						blockACKED[i] = sendACKORNACK(buff, i*blockSize, blockSize, conn)
						if packetNumber/blockSize < uint64(len(blockACKED)) {
							if blockACKED[packetNumber/blockSize] == false && shouldIncrement {
								shouldIncrement = false
								start = i
							}
						}
					}
				}
			}

			before, after := findFileBeforeAfter(packetNumber)
			if packetNumber != lastWrite+1 {
				fmt.Println("write to disk")
				writeToDisk(packet, packetNumber, before, after)
			} else {
				fmt.Println("write to pipe")
				lastWrite = writeToPipe(pipeOut, packet, after)
			}
			buff[packetNumber] = true
			if lastWrite == fileSize-1 {
				pipeOut.Close()
				// fmt.Println(lastWrite)
				//chanout notifies higher level routine to shutdown connection and channels
				conn.Close()
				chanOut <- true
			}
		}
	}
}

func writeToPipe(pipeOut *io.PipeWriter, packet []byte, after string) uint64 {
	afterArray := strings.Split(after, "/")
	after = afterArray[len(afterArray)-1]
	fmt.Println("writing to pipe from ", getPacketNumber(packet), strings.Split(after, "-"))
	fmt.Println("after is ", after)
	n, err := pipeOut.Write(packet[:len(packet)-8])
	fmt.Println(n, err)
	var lastWrite uint64
	lastWrite = getPacketNumber(packet)
	if after != "" {
		fmt.Println("now sending the file after ", getPacketNumber(packet))
		afterFile, _ := ioutil.ReadFile(after)
		fmt.Print("trying to write to pipe this much bytes ", len(afterFile), getPacketNumber(packet), " ")
		n, err = pipeOut.Write(afterFile)

		if err != nil {
			fmt.Print("ERROR")
		}
		fmt.Println(err, n)

		// fmt.Println("done writing")

		lastWrite, _ = strconv.ParseUint(strings.Split(after, "-")[1], 36, 64)
		fmt.Println("new last write is : ", lastWrite)
		os.Remove(after)
	}
	// fmt.Println("last write:", lastWrite)
	return lastWrite
}

func writeToDisk(packet []byte, packetNumber uint64, before string, after string) {
	afterArray := strings.Split(after, "-")
	fmt.Println(strconv.FormatUint(packetNumber, 36))
	fmt.Println("before:", before, "after:", after)
	suffix := afterArray[len(afterArray)-1]
	if after == "" {
		suffix = strconv.FormatUint(packetNumber, 36)
	}
	beforeArray := strings.Split(before, "-")
	prefix := beforeArray[0]
	if before == "" {
		prefix = strconv.FormatUint(packetNumber, 36)
	}
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
	_, err = f.Write(packet[:len(packet)-8])
	if err != nil {
		fmt.Println("error while writing to file")
	}
	if after != "" {
		fAfter, _ := ioutil.ReadFile(after)

		f.Write(fAfter)
		if err != nil {
			fmt.Println(err)
		}
		os.Remove(after)
		err = f.Close()
		if err != nil {
			fmt.Println("error closing file " + f.Name())
		}
	}
	fmt.Println("renaming file to " + prefix + "-" + suffix)

	os.Rename(before, prefix+"-"+suffix)

	stats, err := os.Stat(prefix + "-" + suffix)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(stats.Size())

	fmt.Println("filename is : ", prefix, "-", suffix)
}

func findFileBeforeAfter(packetNumber uint64) (before, after string) {
	files, err := ioutil.ReadDir("./")
	before, after = "", ""
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("and ", strconv.FormatUint(packetNumber, 36))
	for _, f := range files {
		if len(strings.Split(f.Name(), "-")) == 2 {
			if strings.Compare(strings.Split(f.Name(), "-")[1],
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
	}
	fmt.Println("for file ", packetNumber, " ", strconv.FormatUint(packetNumber, 36), " before and after are ", before, " ", after)
	return before, after
}
