package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

//FileToPipe reads a file and prepend packet number and send it to a pipe
// to be encrypted
func FileToPipe(outPipe *io.PipeWriter, filename string, bufferSize int) {
	defer outPipe.Close()
	file, err := os.Open(filename)
	if err != nil {
		return
	}
	defer file.Close()
	buff := make([]byte, bufferSize)
	var packetNumber uint64
	for {
		_, err := file.Read(buff)
		writePacketToPipe(outPipe, packetNumber, buff)
		packetNumber++
		if err != nil {
			if err == io.EOF {
				fmt.Println("end of file")
			}
			break
		}
	}
}

func writePacketToPipe(pipe *io.PipeWriter, packetNumber uint64, buff []byte) {
	finalBuffer := make([]byte, 8, 8+len(buff))
	binary.LittleEndian.PutUint64(finalBuffer, packetNumber)
	copy(finalBuffer[8:], buff)
	pipe.Write(finalBuffer)
}

// pipetochannel reads data from a pipe (packetsize at a time), and sends it to
// a channel. it can send smaller packet than packetsize if the writing end of the
//pipe is done writing
func pipeToChannel(pipe *io.PipeReader, channel chan []byte, packetSize uint64) {
	defer pipe.Close()
	for {
		buff := make([]byte, packetSize)
		n, err := pipe.Read(buff)
		channel <- buff[:n]
		if err != nil {
			break
		}
	}
	close(channel)
}

//StreamWriter takes a filename, a key, and a pipe to write to, and encrypt
//the file with the key to the pipe
//This function encrypts the udp flow
func StreamWriter(inPipe *io.PipeReader, key []byte, pipeWriter *io.PipeWriter) {

	defer inPipe.Close()

	fmt.Println("encrypting with key : ", hex.EncodeToString(key))

	block, err := aes.NewCipher(key)

	if err != nil {
		panic(err)
	}

	var iv [aes.BlockSize]byte

	stream := cipher.NewCTR(block, iv[:])

	if err != nil {
		panic(err)
	}

	writer := &cipher.StreamWriter{S: stream, W: pipeWriter}

	// Copy the input file to the output file, encrypting as we go.
	if _, err := io.Copy(writer, inPipe); err != nil {
		if err == io.EOF {
			fmt.Println("everything alright")
			pipeWriter.Close()
		} else {
			fmt.Println("not everything alright")
		}
	}
	pipeWriter.Close()
}

//StreamReaderDecrypt decrypt the stream coming from the pipe using the key,
// and writes it to the destination file
func StreamReaderDecrypt(readPipe *io.PipeReader, outpipe io.Writer, key []byte) {

	block, err := aes.NewCipher(key)
	fmt.Println("decrypting with key: ", hex.EncodeToString(key))
	if err != nil {
		panic(err)
	}

	var iv [aes.BlockSize]byte

	stream := cipher.NewCTR(block, iv[:])

	if err != nil {
		panic(err)
	}

	reader := &cipher.StreamReader{S: stream, R: readPipe}

	if _, err := io.Copy(outpipe, reader); err != nil {
		panic(err)
	}

	// fmt.Println(os.Rename(destinationFile+".tmp", destinationFile))
}

//CreateAESKey creates a new random key
func CreateAESKey() []byte {
	randG := rand.Reader
	keyBuffer := make([]byte, 16)
	bytesCreated, err := randG.Read(keyBuffer)

	if bytesCreated != 16 || err != nil {
		return nil
	}
	return keyBuffer
}
