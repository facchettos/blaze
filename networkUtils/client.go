package networkUtils

import (
	"blaze/security"
	"crypto/rsa"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
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
func TCPClient(privateKey *rsa.PrivateKey) {
	conn, err := EstablishConnection("localhost:8080")
	if err != nil {
		fmt.Println("impossible to connect to remote host")
		return
	}
	aesKey := doChallenge(privateKey, conn)
	fmt.Println("the key is: " + hex.EncodeToString(aesKey))
	buff := make([]byte, 2)
	_, err = conn.Read(buff)
	if err != nil {
		fmt.Println("reading from the connection to get the udp port failed")
	}
	udpPort := binary.LittleEndian.Uint16(buff)
	fmt.Println(udpPort)
	fileChannel := make(chan []byte, 5000)
	fmt.Println(len(fileChannel))
	//TODO open udp dial with the port
	// add channel, and write files to channel
	// readFrom file and encrypt it
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
	fmt.Print("number to read is: ")
	fmt.Println(numberToRead)
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
