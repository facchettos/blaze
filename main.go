package main

import (
	"blaze/security"
	streams "blaze/streamsutils"
	"crypto/aes"
	"io"
)

func main() {
	pr, pw := io.Pipe()
	pr2, pw2 := io.Pipe()
	pr3, pw3 := io.Pipe()
	pr4, pw4 := io.Pipe()
	go streams.FilePiper("flowDiagram", pw)
	go streams.PacketGenerator(pr, pw2, 50)
	var key [aes.BlockSize]byte
	go streams.PacketEncryptor(pr2, pw3, key[:])
	go security.StreamReaderDecrypt(pr3, pw4, key[:])
	streams.FileShaper(pr4, "testfileshaper", 50)
}
