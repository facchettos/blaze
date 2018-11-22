package networkUtils

import (
	"blaze/networkUtils/networkproto"
	"blaze/security"
	"crypto/rsa"
	"errors"
	"fmt"
	"net"

	"github.com/golang/protobuf/proto"
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

//GetChallenge sends the private key name and gets back the challenge
func GetChallenge(conn *net.TCPConn, keyName string) []byte {
	request := &networkproto.ACKNACK{
		MessageType: "connrequest",
		KeyToUse:    keyName,
	}
	marshalled, err := proto.Marshal(request)
	if err != nil {
		fmt.Println("Error while marshalling the getchallenge request")
		return nil
	}
	WriteProto(conn, marshalled)

	return ReadProto(conn).GetToDecrypt()
}

// SendChallengeAnswer decrypt the key, and sends back the hash of it
func SendChallengeAnswer(challenge []byte, privateKey *rsa.PrivateKey, conn *net.TCPConn) {
	decryptedkey := security.DoChallenge(challenge, privateKey)
	answer := &networkproto.ACKNACK{
		MessageType: "challengeanswer",
		Hash:        security.ComputeHash(decryptedkey),
	}
	marshalled, err := proto.Marshal(answer)
	if err != nil {
		fmt.Println("Problem during protobuf marshalling")
		return
	}
	WriteProto(conn, marshalled)
}
