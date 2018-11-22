package main

import (
	"blaze/security"
	"crypto/aes"
	"fmt"
	"io"
	"os"
	"time"
)

func main() {
	// key := security.CreateKeyPairRSA()
	// cipher := security.EncryptWithPublicKey([]byte("toto"), &key.PublicKey)
	// plain := security.DoChallenge(cipher, key)
	// fmt.Println(string(plain))
	// buff := make([]byte, 18)
	// buff2 := make([]byte, 10)
	// for i := 0; i < len(buff2); i++ {
	// 	buff2[i] = byte('a')
	// }
	// binary.LittleEndian.PutUint64(buff, ^uint64(0)-uint64(255))
	// copy(buff[8:], buff2)
	// fmt.Println(buff)
	pr, pw := io.Pipe()
	pr2, pw2 := io.Pipe()
	pr3, pw3 := io.Pipe()
	file, _ := os.Open("flowDiagram")
	go func() {
		io.Copy(pw, file)
		pw.Close()
	}()
	go func() {
		buf := make([]byte, 1000)
		for {
			int, err := pr2.Read(buf)
			fmt.Println(int)
			pw3.Write(buf[:int])
			if int < len(buf) {
				break
			}
			fmt.Println(string(buf))
			fmt.Println(int, err)
		}
	}()
	var key [aes.BlockSize]byte
	go func() {
		security.StreamReaderDecrypt(pr3, "flow2", key[:])
	}()
	security.StreamWriter(pr, key[:], pw2)

	time.Sleep(time.Second * 2)
}
