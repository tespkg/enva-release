package kvs

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"

	"golang.org/x/crypto/pbkdf2"
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
	password []byte
	aesgcm   cipher.AEAD
}

func NewCreds() (*Creds, error) {
	secret, err := getSecretCred()
	if err != nil {
		return nil, err
	}
	return newCreds(secret)
}

func newCreds(password string) (*Creds, error) {
	hasher := sha256.New()
	hasher.Write([]byte(password))
	key32Bytes := hasher.Sum(nil)

	block, err := aes.NewCipher(key32Bytes)
	if err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Creds{password: key32Bytes, aesgcm: aesgcm}, nil
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

type PbkdfCreds struct {
	saltSize  int
	nonceSize int

	keySize    int
	iterations int
}

var stdPkdfCreds = newPbkdfCreds(32, 12, 32, 100000)

func newPbkdfCreds(saltSize, nonceSize, keySize, iterations int) *PbkdfCreds {
	return &PbkdfCreds{
		saltSize:   saltSize,
		nonceSize:  nonceSize,
		keySize:    keySize,
		iterations: iterations,
	}
}

func (c *PbkdfCreds) Encrypt(plaintext, password string) (string, error) {
	// Generate a random salt
	salt := make([]byte, c.saltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return "", err
	}

	// Derive the key using PBKDF2
	key := pbkdf2.Key([]byte(password), salt, c.iterations, c.keySize, sha256.New)

	// Create AES block cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	// Generate a random nonce
	nonce := make([]byte, c.nonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	// Create GCM (Galois/Counter Mode) cipher
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// Encrypt the data
	ciphertext := aesgcm.Seal(nil, nonce, []byte(plaintext), nil)

	// Concatenate salt, nonce, and ciphertext for easier transport
	finalData := append(salt, nonce...)
	finalData = append(finalData, ciphertext...)

	// Encode to base64 for a compact string representation
	encData := base64.StdEncoding.EncodeToString(finalData)

	return encData, nil
}

func (c *PbkdfCreds) Decrypt(encData, password string) (string, error) {
	// Decode the base64-encoded data
	data, err := base64.StdEncoding.DecodeString(encData)
	if err != nil {
		return "", err
	}

	// Extract the salt, nonce, and ciphertext
	if len(data) < c.saltSize+c.nonceSize {
		return "", errors.New("invalid encrypted data")
	}
	salt := data[:c.saltSize]
	nonce := data[c.saltSize : c.saltSize+c.nonceSize]
	ciphertext := data[c.saltSize+c.nonceSize:]

	// Derive the key using PBKDF2
	key := pbkdf2.Key([]byte(password), salt, c.iterations, c.keySize, sha256.New)

	// Create AES block cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	// Create GCM cipher
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// Decrypt the data
	plaintext, err := aesgcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}
