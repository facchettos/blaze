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

func TcpClient(privateKey *rsa.PrivateKey) {
	conn, err := EstablishConnection("localhost:8080")
	if err != nil {
		fmt.Println("impossible to connect to remote host")
		return
	}
	aesKey := doChallenge(privateKey, conn)
	fmt.Println("the key is: " + hex.EncodeToString(aesKey))
}

func doChallenge(privateKey *rsa.PrivateKey, conn net.Conn) []byte {

	buffForSize := make([]byte, 2)
	n, err := conn.Read(buffForSize)
	if err != nil || n != 2 {
		return nil
	}
	numberToRead := binary.LittleEndian.Uint16(buffForSize)
	conn.SetReadDeadline(time.Now().Add(time.Second * 2))
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

	conn.SetReadDeadline(time.Time{})

	if err != nil || uint16(n) != numberToRead {
		return nil
	}

	aeskey := security.DoChallenge(buff[:n], privateKey)
	fmt.Println("decryption gave: " + hex.EncodeToString(aeskey))
	aeshash := security.ComputeHash(aeskey)
	fmt.Println("hash from client is: " + hex.EncodeToString(aeshash))
	conn.Write(aeshash)
	return aeskey
}
