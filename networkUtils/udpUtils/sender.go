package udpUtils

import (
	"net"
	"time"
)

// SendLoop get bytes from the channel, and send it via conn
func SendLoop(channel chan []byte, conn *net.UDPConn, interPktTime time.Duration, treshold time.Duration) {
	interPacket := interPktTime
	numberOfTries := 1
	for toSend := range channel {
		if numberOfTries > 2 {
			interPacket += treshold
		} else if numberOfTries == 1 {
			interPacket -= treshold
		}
		n, _ := conn.Write(toSend)
		time.Sleep(interPacket)
		for n == 0 {
			n, _ = conn.Write(toSend)
			numberOfTries++
			time.Sleep(interPacket)
		}
	}
}
