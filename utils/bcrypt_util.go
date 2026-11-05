package utils

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"errors"

	"golang.org/x/crypto/bcrypt"
)

var AesKey = "wh&navfirst&2020"

// BcryptHash 使用 bcrypt 对密码进行加密
func BcryptHash(password string) string {
	hash, _ := BcryptHashE(password)
	return hash
}

// BcryptHashE returns bcrypt validation errors to callers that persist passwords.
// BcryptHash is retained for source compatibility; new code should use BcryptHashE.
func BcryptHashE(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash), err
}

// BcryptCheck 对比明文密码和数据库的哈希值
func BcryptCheck(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// MD5V md5加密
func MD5V(str []byte, b ...byte) string {
	h := md5.New()
	h.Write(str)
	return hex.EncodeToString(h.Sum(b))
}

// AesEncrypt AES加密
func AesEncrypt(plainText string) (string, error) {
	return AesEncryptWithKey(plainText, AesKey)
}

// AesDecrypt AES解密
func AesDecrypt(cipherText string) (string, error) {
	return AesDecryptWithKey(cipherText, AesKey)
}

// AesEncryptWithKey AES加密
func AesEncryptWithKey(plainText, key string) (string, error) {
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}
	origData := pkcs7Padding([]byte(plainText), block.BlockSize())
	blockMode := cipher.NewCBCEncrypter(block, []byte(key)[:block.BlockSize()])
	crypted := make([]byte, len(origData))
	blockMode.CryptBlocks(crypted, origData)
	return base64.StdEncoding.EncodeToString(crypted), nil
}

// AesDecryptWithKey AES解密
func AesDecryptWithKey(cipherText, key string) (string, error) {
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}
	crypted, err := base64.StdEncoding.DecodeString(cipherText)
	if err != nil {
		return "", err
	}
	if len(crypted) == 0 || len(crypted)%block.BlockSize() != 0 {
		return "", errors.New("invalid AES ciphertext length")
	}
	blockMode := cipher.NewCBCDecrypter(block, []byte(key)[:block.BlockSize()])
	origData := make([]byte, len(crypted))
	blockMode.CryptBlocks(origData, crypted)
	origData, err = unpadPKCS7(origData, block.BlockSize())
	if err != nil {
		return "", err
	}
	return string(origData), nil
}

// PKCS7填充
func pkcs7Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padText...)
}

// PKCS7去填充
func pkcs7UnPadding(origData []byte) []byte {
	data, _ := unpadPKCS7(origData, aes.BlockSize)
	return data
}

func unpadPKCS7(data []byte, blockSize int) ([]byte, error) {
	if len(data) == 0 || len(data)%blockSize != 0 {
		return nil, errors.New("invalid PKCS7 padding")
	}
	n := int(data[len(data)-1])
	if n == 0 || n > blockSize || n > len(data) {
		return nil, errors.New("invalid PKCS7 padding")
	}
	for _, b := range data[len(data)-n:] {
		if int(b) != n {
			return nil, errors.New("invalid PKCS7 padding")
		}
	}
	return data[:len(data)-n], nil
}
