package kvs

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"os"
)

const (
	// default secret credential, base64 encoded, that used as the credential to encrypt and decrypt of the envk var
	defaultSecretCred = "ajghUERAM3VNcSF6SyZWcA=="
)

func getSecretCred() (string, error) {
	sc := defaultSecretCred
	if s := os.Getenv("SECRET_CREDENTIAL"); s != "" {
		sc = s
	}
	out, err := base64.StdEncoding.DecodeString(sc)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

type Creds struct {
	key    []byte
	aesgcm cipher.AEAD
}

func NewCreds() (*Creds, error) {
	secret, err := getSecretCred()
	if err != nil {
		return nil, err
	}
	return newCreds(secret)
}

func newCreds(key string) (*Creds, error) {
	hasher := sha256.New()
	hasher.Write([]byte(key))
	key32Bytes := hasher.Sum(nil)

	block, err := aes.NewCipher(key32Bytes)
	if err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Creds{key: key32Bytes, aesgcm: aesgcm}, nil
}

func (c *Creds) Encrypt(plaintext string) (string, error) {
	ciphertext, nonce, err := c.encrypt([]byte(plaintext))
	if err != nil {
		return "", err
	}
	combined := append(nonce, ciphertext...)
	out := base64.StdEncoding.EncodeToString(combined)
	return out, nil
}

func (c *Creds) Decrypt(ciphertext string) (string, error) {
	bytes, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}
	nonceSize := c.aesgcm.NonceSize()
	if len(bytes) < nonceSize {
		return "", fmt.Errorf("invalid ciphertext")
	}
	nonce, realcipher := bytes[:nonceSize], bytes[nonceSize:]
	plaintext, err := c.decrypt(nonce, realcipher)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

func (c *Creds) encrypt(plaintext []byte) ([]byte, []byte, error) {
	nonce := make([]byte, c.aesgcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, err
	}

	ciphertext := c.aesgcm.Seal(nil, nonce, plaintext, nil)
	return ciphertext, nonce, nil
}

func (c *Creds) decrypt(nonce, ciphertext []byte) ([]byte, error) {
	plaintext, err := c.aesgcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}
