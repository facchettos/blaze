package security

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"log"
)

//CreateKeyPairRSA create a new pair of key for RSA
func CreateKeyPairRSA() *rsa.PrivateKey {

	rand := rand.Reader
	bitSize := 2048

	key, err := rsa.GenerateKey(rand, bitSize)
	if err != nil {
		log.Fatal(err)
	} else {
		// fmt.Println(key.D)
		// fmt.Println(key.N)
		fmt.Println("Created a new key")
	}
	return key
}

func ComputeHash(message []byte) []byte {
	hash := sha256.Sum256(message)
	return hash[:32]
}

func DoChallenge(challenge []byte, privateKey *rsa.PrivateKey) []byte {
	hash := sha512.New()
	plaintext, err := rsa.DecryptOAEP(hash, rand.Reader, privateKey, challenge, nil)
	if err != nil {
		return nil
	}
	return plaintext
}

func EncryptWithPublicKey(msg []byte, pub *rsa.PublicKey) []byte {
	hash := sha512.New()
	ciphertext, err := rsa.EncryptOAEP(hash, rand.Reader, pub, msg, nil)
	if err != nil {
		return nil
	}
	return ciphertext
}
