package utils

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"golang.org/x/crypto/ssh"
)

func GenerateKey(bits int) (*rsa.PrivateKey, error) {
	private, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil,err
	}
	return private, nil
}

func MakeSSHKeyPair() (privateKey string,pubKey string,err error) {
	var rsaKey * rsa.PrivateKey
	if rsaKey,  err = GenerateKey(2048);err != nil {
         return
	}
	if sshPub,fail := ssh.NewPublicKey(&rsaKey.PublicKey);fail != nil {
		err = fail
		return
	}else{
		pubKey=string(ssh.MarshalAuthorizedKey(sshPub))
	}
	privateKey = string(pem.EncodeToMemory(&pem.Block{
		Bytes: x509.MarshalPKCS1PrivateKey(rsaKey),
		Type:  "RSA PRIVATE KEY",
	}))
	return
}

