package main

import (
	"blaze/security"
	"fmt"
	"reflect"
)

func main() {
	// pr, pw := io.Pipe()
	// pr2, pw2 := io.Pipe()
	// pr3, pw3 := io.Pipe()
	// pr4, pw4 := io.Pipe()
	// go streams.FilePiper("test", pw)
	// go streams.PacketGenerator(pr, pw2, 50)
	// var key [aes.BlockSize]byte
	// go streams.PacketEncryptor(pr2, pw3, key[:])
	// go security.StreamReaderDecrypt(pr3, pw4, key[:])
	// streams.FileShaper(pr4, "testfileshaper", 50)
	// rsakey := security.CreateKeyPairRSA()
	// go networkUtils.ListenRequest(8080, &rsakey.PublicKey)
	// time.Sleep(time.Second)
	// networkUtils.TcpClient(rsakey)
	// time.Sleep(time.Second)
	// networkUtils.WriteToDisk([]byte("packet\n"), uint64(3), "", "")
	// intslice := make([]byte, 8)
	// binary.LittleEndian.PutUint64(intslice, ^uint64(0))
	// fmt.Println(networkUtils.GetPacketNumber(intslice))
	// fmt.Println(^uint64(0))
	// listen, _ := net.Listen("tcp", ":8080")
	// for {
	// 	conn, _ := listen.Accept()
	// 	buff := make([]byte, 10)
	// 	for {
	// 		conn.SetDeadline(time.Now().Add(time.Second * 5))
	// 		_, err := conn.Read(buff)
	// 		fmt.Println(err)
	// 		fmt.Println(buff)
	// 	}
	// }

	// packetChanSender := make(chan []byte)

	// var key [aes.BlockSize]byte

	// // go networkUtils.UdpListener(8080, "127.0.0.1", 40, 10, "testtototo", key[:])
	// // time.Sleep(time.Second)
	// // conn, _ := net.Dial("udp", "127.0.0.1:8080")
	// go func() {
	// 	result, _ := os.OpenFile("testtototo", os.O_WRONLY|os.O_CREATE, 0600)
	// 	security.StreamReaderDecrypt(pr, result, key[:])

	// }()

	// go networkUtils.SendChunksToChannel("flowDiagram", packetChanSender, 32, key)
	// go networkUtils.OpenUdp(1280, "8080", 485, "testtototo.jpg", key[:], "127.0.0.1", 200, 10, nil)
	// time.Sleep(time.Second)
	// add, _ := net.ResolveUDPAddr("udp", "127.0.0.1:8080")
	// udpconn, err := net.DialUDP("udp", nil, add)
	// if err != nil {
	// 	fmt.Println(err)
	// }
	// tcpaddress, _ := net.ResolveTCPAddr("tcp", ":8081")
	// tcpListener, _ := net.ListenTCP("tcp", tcpaddress)
	// tcpconnWrite, _ := net.DialTCP("tcp", nil, tcpaddress)
	// tcpConnRead, _ := tcpListener.Accept()
	// go func() {
	// 	time.Sleep(time.Second * 4)
	// 	ack := &networkproto.ACKNACK{
	// 		MessageType: "done",
	// 	}
	// 	toWrite, _ := proto.Marshal(ack)
	// 	buff := make([]byte, 8)
	// 	binary.LittleEndian.PutUint64(buff, uint64(len(toWrite)))
	// 	tcpconnWrite.Write(buff)
	// 	tcpconnWrite.Write(toWrite)
	// }()

	// networkUtils.SendFile("lock-screen.jpg", key, 200, udpconn, 1280, tcpConnRead, 20000)

	// go func() {
	// 	time.Sleep(time.Second)
	// 	conn, _ := net.Dial("udp", "127.0.0.1:8080")
	// 	for packet := range packetChanSender {
	// 		conn.Write(packet)
	// 	}
	// }()

	// networkUtils.ReceiveUDP(40, pc, packetChan, done)

	// time.Sleep(time.Second)
	mykey, _ := security.ParseRsaPublicKeyFromPubFile("rsa.pub")
	fmt.Println(mykey)
	fmt.Println(reflect.TypeOf(mykey))

	res := security.EncryptWithPublicKey([]byte("salut toi"), mykey)
	fmt.Println(res)
	fmt.Println(string(security.DoChallenge(res, security.ReadPrivateKeyFromFile("rsa.pem"))))
}
