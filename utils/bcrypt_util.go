package utils

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"

	"golang.org/x/crypto/bcrypt"
)

var AesKey = "wh&navfirst&2020"

// BcryptHash 使用 bcrypt 对密码进行加密
func BcryptHash(password string) string {
	fromPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(fromPassword)
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
	blockMode := cipher.NewCBCDecrypter(block, []byte(key)[:block.BlockSize()])
	origData := make([]byte, len(crypted))
	blockMode.CryptBlocks(origData, crypted)
	origData = pkcs7UnPadding(origData)
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
	length := len(origData)
	unPadding := int(origData[length-1])
	return origData[:(length - unPadding)]
}
