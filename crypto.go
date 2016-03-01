package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	b64 "encoding/base64"
	"errors"
	"log"
)

func encrypt(key_str, iv_str, text string) (string, error) {

	key, err1 := b64.RawURLEncoding.DecodeString(key_str)
	iv, err2 := b64.RawURLEncoding.DecodeString(iv_str)

	if err1 != nil || err2 != nil {
		log.Println("Error decoding key/iv")
		return "", errors.New("Error decoding key/iv")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	b := []byte(text)
	b = PKCS5Padding(b, aes.BlockSize, len(text))
	ciphertext := make([]byte, len(b))

	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext, b)

	return b64.RawURLEncoding.EncodeToString(ciphertext), nil
}

func decrypt(key_str, iv_str, ciphertext string) (string, error) {

	key, err1 := b64.RawURLEncoding.DecodeString(key_str)
	iv, err2 := b64.RawURLEncoding.DecodeString(iv_str)
	text, err3 := b64.RawURLEncoding.DecodeString(ciphertext)

	if err1 != nil || err2 != nil || err3 != nil {
		log.Println("Error decoding key/iv/data")
		return "", errors.New("Error decoding key/iv")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	if len(text) < aes.BlockSize {
		return "", errors.New("ciphertext too short")
	}

	decrypted := make([]byte, len(text))
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(decrypted, text)

	return string(PKCS5UnPadding(decrypted)), nil
}

func PKCS5Padding(ciphertext []byte, blockSize int, after int) []byte {
	padding := (blockSize - len(ciphertext)%blockSize)
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

func PKCS5UnPadding(src []byte) []byte {
	length := len(src)
	unpadding := int(src[length-1])
	return src[:(length - unpadding)]
}
