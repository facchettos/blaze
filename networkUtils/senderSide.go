package networkUtils

import (
	streams "blaze/streamsutils"
	"container/list"
	"crypto/aes"
	"fmt"
	"io"
)

type order struct {
	orderType    uint16
	packetNumber []uint64 // corresponds to a NACK
	from         uint64   // corresponds to an ACK
	to           uint64   // corresponds to an ACK
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

func SendChunksToChannel(filename string, channel chan []byte, buffsize int, key [aes.BlockSize]byte) {
	// make buffsize to be a multiple of 16, to ensure
	//  that encrypted packet size is the same than the buffer's
	buffsize16 := (buffsize - (buffsize % 16)) + 8
	pr, pw := io.Pipe()
	pr2, pw2 := io.Pipe()
	pr3, pw3 := io.Pipe()
	go streams.FilePiper(filename, pw)
	go streams.PacketEncryptor(pr, pw2, key[:])
	go streams.PacketGenerator(pr2, pw3, buffsize16)
	// i := 0
	for {
		// fmt.Println(i)
		buff := make([]byte, buffsize16)
		n, err := io.ReadFull(pr3, buff)

		channel <- buff[:n]
		// i++
		if err != nil || n < len(buff) {
			close(channel)
			return
		}

	}
}

//CHAN OUT (to packetsender) HAS TO BE UNBUFFERED
func packetBuffHandler(
	orders chan order,
	packets chan []byte,
	chanOut chan []byte,
	numbOfPackets uint64,
	maxBuff, blockSize uint64) {

	packetsList := list.New()

Loop:
	for {
		select {
		case orderFromChan := <-orders:
			if orderFromChan.orderType != done {
				packetsList = handleOrderList(orderFromChan, packetsList, chanOut, blockSize)
			} else {
				break Loop
			}
		default:
			if packet := <-packets; uint64(packetsList.Len()) < maxBuff && packet != nil {

				chanOut <- packet
				packetsList.PushBack(packetStruct{getPacketNumber(packet), packet})
			}
		}
	}
}

type packetStruct struct {
	Number  uint64
	Payload []byte
}

func handleOrderList(order order, packets *list.List, chanOut chan []byte, blockSize uint64) *list.List {
	if order.orderType == send {
		firstPacket := order.packetNumber[0]
		firstPOfblock := firstPacket - (firstPacket % blockSize)
		fmt.Println(firstPOfblock)
		//NACK
		fmt.Println("received nack")
		i := 0

		for e := packets.Front(); e != nil; {
			fmt.Println(i)
			// do something with e.Value
			next := e.Next()
			if e.Value.(packetStruct).Number == order.packetNumber[i] {
				chanOut <- e.Value.(packetStruct).Payload
				if i < len(order.packetNumber)-1 {
					i++
				}
			} else if e.Value.(packetStruct).Number >= firstPOfblock &&
				e.Value.(packetStruct).Number < firstPOfblock+blockSize {

				packets.Remove(e)
			} else if e.Value.(packetStruct).Number >= firstPOfblock+blockSize {
				break
			}
			e = next
		}
	} else if order.orderType == remove {
		//ACK
		fmt.Println("received ack")
		fmt.Println("from: ", order.from)
		fmt.Println("to: ", order.to)
		for e := packets.Front(); e != nil; {
			next := e.Next()
			if e.Value.(packetStruct).Number >= order.from {
				if e.Value.(packetStruct).Number > order.to {
					break
				}
				packets.Remove(e)
			}
			e = next
		}
	}
	return packets
}

func TestList() {
	mylist := list.New()
	myarray := make([]uint64, 3)
	myarray[0] = 1
	myarray[1] = 2
	myarray[2] = 3
	myorder := order{2, nil, 2, 9}
	mylist.PushBack(packetStruct{0, nil})
	mylist.PushBack(packetStruct{1, nil})
	mylist.PushBack(packetStruct{2, nil})
	mylist.PushBack(packetStruct{3, nil})
	mylist.PushBack(packetStruct{4, nil})
	mylist.PushBack(packetStruct{5, nil})
	mylist.PushBack(packetStruct{6, nil})
	mylist.PushBack(packetStruct{7, nil})
	mylist.PushBack(packetStruct{8, nil})
	mylist.PushBack(packetStruct{9, nil})
	mylist.PushBack(packetStruct{10, nil})
	mylist.PushBack(packetStruct{11, nil})
	for e := mylist.Front(); e != nil; e = e.Next() {
		fmt.Println(e.Value)
	}
	fmt.Println()
	mylist = handleOrderList(myorder, mylist, nil, 10)
	for e := mylist.Front(); e != nil; e = e.Next() {
		fmt.Println(e.Value)
	}

}
func handlerOrder(order order, packetBuff [][]byte, buffedPackets uint64,
	firstPacketIndex uint64, chanOut chan []byte) (uint64, uint64, [][]byte) {
	if order.orderType == send {
		//NACK
		for _, n := range order.packetNumber {
			chanOut <- packetBuff[n]
		}
	} else if order.orderType == remove {
		//ACK
		for i := order.from; i <= order.to; i++ {
			packetBuff[i] = nil
			buffedPackets--
		}
		if order.from == firstPacketIndex {
			for packetBuff[firstPacketIndex] == nil {
				firstPacketIndex++
			}
		}
	}
	return buffedPackets, firstPacketIndex, packetBuff
}

func continueSending(
	buffedPackets,
	maxBuff,
	packetIndex uint64,
	packets chan []byte,
	chanOut chan []byte,
	packetBuff [][]byte) (uint64, uint64) {

	if buffedPackets < maxBuff && packetIndex < uint64(len(packetBuff)) {
		packet := <-packets
		packetBuff[packetIndex] = packet
		packetIndex++
		buffedPackets++
		chanOut <- packet
	}
	return buffedPackets, packetIndex
}
