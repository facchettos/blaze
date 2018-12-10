package networkUtils

import (
	streams "blaze/streamsutils"
	"crypto/aes"
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
	i := 0
	for {
		buff := make([]byte, buffsize16)
		n, err := io.ReadFull(pr3, buff)
		// fmt.Println(string(buff[:n]))
		channel <- buff[:n]
		i++
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
	maxBuff uint64) {

	packetBuff := make([][]byte, numbOfPackets)
	var packetIndex uint64
	var buffedPackets uint64
	var firstPacketIndex uint64
	packetIndex = 0
	buffedPackets = 0
	firstPacketIndex = 0

Loop:
	for {
		select {
		case orderFromChan := <-orders:
			if orderFromChan.orderType != done {
				buffedPackets, firstPacketIndex, packetBuff =
					handlerOrder(orderFromChan,
						packetBuff,
						buffedPackets,
						firstPacketIndex,
						chanOut)
			} else {
				break Loop
			}
		default:
			buffedPackets, packetIndex = continueSending(buffedPackets,
				maxBuff,
				packetIndex,
				packets,
				chanOut,
				packetBuff)
		}
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
