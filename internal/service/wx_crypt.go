package service

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
)

type WxCrypt struct {
	Token          string
	EncodingAESKey string
	AppID          string
	aesKey         []byte
}

type EncryptedXMLRequest struct {
	XMLName    xml.Name `xml:"xml"`
	ToUserName string   `xml:"ToUserName"`
	Encrypt    string   `xml:"Encrypt"`
}

type EncryptedXMLResponse struct {
	XMLName      xml.Name `xml:"xml"`
	Encrypt      string   `xml:"Encrypt"`
	MsgSignature string   `xml:"MsgSignature"`
	TimeStamp    string   `xml:"TimeStamp"`
	Nonce        string   `xml:"Nonce"`
}

func NewWxCrypt(token, encodingAESKey, appID string) (*WxCrypt, error) {
	if len(encodingAESKey) != 43 {
		return nil, errors.New("invalid encodingAESKey length")
	}
	aesKey, err := base64.StdEncoding.DecodeString(encodingAESKey + "=")
	if err != nil {
		return nil, fmt.Errorf("decode encodingAESKey error: %w", err)
	}

	return &WxCrypt{
		Token:          token,
		EncodingAESKey: encodingAESKey,
		AppID:          appID,
		aesKey:         aesKey,
	}, nil
}

// CheckSignature 签名校验
func (w *WxCrypt) VerifySignature(msgSignature, timestamp, nonce, encrypt string) bool {
	strs := []string{w.Token, timestamp, nonce, encrypt}
	sort.Strings(strs)
	h := sha1.New()
	h.Write([]byte(strings.Join(strs, "")))
	calculated := hex.EncodeToString(h.Sum(nil))
	return calculated == msgSignature
}

// DecryptMsg 解密微信密文并返回明文 XML
func (w *WxCrypt) DecryptMsg(msgSignature, timestamp, nonce string, rawBody []byte) ([]byte, error) {
	var req EncryptedXMLRequest
	if err := xml.Unmarshal(rawBody, &req); err != nil {
		return nil, fmt.Errorf("unmarshal encrypted xml error: %w", err)
	}

	if req.Encrypt == "" {
		// 如果未加密，直接返回原始报文
		return rawBody, nil
	}

	// 校验签名
	if msgSignature != "" && !w.VerifySignature(msgSignature, timestamp, nonce, req.Encrypt) {
		return nil, errors.New("msg_signature verification failed")
	}

	cipherData, err := base64.StdEncoding.DecodeString(req.Encrypt)
	if err != nil {
		return nil, fmt.Errorf("decode base64 error: %w", err)
	}

	block, err := aes.NewCipher(w.aesKey)
	if err != nil {
		return nil, err
	}

	if len(cipherData) < aes.BlockSize || len(cipherData)%aes.BlockSize != 0 {
		return nil, errors.New("invalid cipher data size")
	}

	// IV 取 AESKey 的前 16 字节
	iv := w.aesKey[:16]
	mode := cipher.NewCBCDecrypter(block, iv)
	plainPad := make([]byte, len(cipherData))
	mode.CryptBlocks(plainPad, cipherData)

	plain := pkcs7Unpad(plainPad)
	if len(plain) < 20 {
		return nil, errors.New("plain text too short")
	}

	// 前 16 字节为随机串
	// 4 字节为明文长度（网络字节序）
	xmlLen := binary.BigEndian.Uint32(plain[16:20])
	end := 20 + xmlLen
	if uint32(len(plain)) < end {
		return nil, errors.New("xml length overflow")
	}

	xmlBytes := plain[20:end]
	return xmlBytes, nil
}

// EncryptMsg 将明文 XML 加密为微信密文 XML 响应
func (w *WxCrypt) EncryptMsg(plainXml []byte, timestamp, nonce string) ([]byte, error) {
	if timestamp == "" {
		timestamp = fmt.Sprintf("%d", time.Now().Unix())
	}
	if nonce == "" {
		nonce = fmt.Sprintf("%d", time.Now().UnixNano())
	}

	// 16 字节随机串
	randomBytes := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, randomBytes); err != nil {
		return nil, err
	}

	// 4 字节明文长度 (网络字节序)
	bufLen := make([]byte, 4)
	binary.BigEndian.PutUint32(bufLen, uint32(len(plainXml)))

	// 拼装待加密内容: 16B 随机串 + 4B 长度 + 明文 XML + AppID
	raw := bytes.NewBuffer(randomBytes)
	raw.Write(bufLen)
	raw.Write(plainXml)
	raw.WriteString(w.AppID)

	padded := pkcs7Pad(raw.Bytes(), 32)

	block, err := aes.NewCipher(w.aesKey)
	if err != nil {
		return nil, err
	}

	iv := w.aesKey[:16]
	mode := cipher.NewCBCEncrypter(block, iv)
	cipherData := make([]byte, len(padded))
	mode.CryptBlocks(cipherData, padded)

	encryptStr := base64.StdEncoding.EncodeToString(cipherData)

	// 计算签名
	strs := []string{w.Token, timestamp, nonce, encryptStr}
	sort.Strings(strs)
	h := sha1.New()
	h.Write([]byte(strings.Join(strs, "")))
	signature := hex.EncodeToString(h.Sum(nil))

	resp := EncryptedXMLResponse{
		Encrypt:      encryptStr,
		MsgSignature: signature,
		TimeStamp:    timestamp,
		Nonce:        nonce,
	}

	return xml.Marshal(resp)
}

func pkcs7Pad(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(data, padText...)
}

func pkcs7Unpad(data []byte) []byte {
	length := len(data)
	if length == 0 {
		return data
	}
	unpadding := int(data[length-1])
	if length < unpadding || unpadding > 32 {
		return data
	}
	return data[:(length - unpadding)]
}
