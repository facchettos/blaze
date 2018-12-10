package streamsutils

import (
	"blaze/security"
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

// FilePiper reads from a file and sends it to a pipe
func FilePiper(filename string, pipeWriter *io.PipeWriter) {
	file, err := os.Open(filename)
	if err != nil {
		fmt.Println("error while opening file" + filename)
		return
	}
	io.Copy(pipeWriter, file)
	pipeWriter.Close()
}

// PacketGenerator reads from a pipe and splits the file into small packets
//  and adds a packet number to it
func PacketGenerator(pipeIn *io.PipeReader, pipeOut *io.PipeWriter, packetSize int) {
	buffin := make([]byte, packetSize-8)
	packetNumberAsBytes := make([]byte, 8)
	var packetNumber uint64
	packetNumber = 0
	// fmt.Println("entering packetgen loop")
	// fmt.Println(n)
	// fmt.Println(err)

	for {
		n, err := io.ReadFull(pipeIn, buffin)
		fmt.Println(packetNumber)
		binary.LittleEndian.PutUint64(packetNumberAsBytes, packetNumber)
		packetNumber++
		buffout := append(buffin[:n], packetNumberAsBytes...)
		// fmt.Println("buffout length= ", len(buffout))
		// fmt.Println(n)

		pipeOut.Write(buffout[:n+8])
		if err != nil {
			fmt.Println("err: ", err)
			fmt.Println(n)
			break
		}
		// fmt.Println("buffin length:")
		// fmt.Println(len(buffin))
	}
	pipeOut.Close()
}

// PacketEncryptor is a wrapper for the different encryption modes
func PacketEncryptor(pipeIn *io.PipeReader, pipeOut *io.PipeWriter, key []byte) {
	security.StreamWriter(pipeIn, key, pipeOut)
}

// FileShaper reads from a stream and removes the packet informations
//  then writes the file to disk
func FileShaper(pipein *io.PipeReader, filename string, buffsize int) {
	buff := make([]byte, buffsize)
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	fmt.Println(err)
	defer file.Close()
	for {
		n, err := pipein.Read(buff)
		if err != nil || n < 9 {
			return
		}
		file.Write(buff[:n-8])
		fmt.Print("bytes read : ")
		// fmt.Println(n - 8)
		if n < buffsize {
			return
		}
	}
}
