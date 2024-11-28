package quark

import (
	"crypto/md5"
	"crypto/sha1"
	"encoding/hex"
	"io"
	"math/rand"
	"mime"
	"os"
	"path"
	"time"
)

var extraMimeTypes = map[string]string{
	".apk": "application/vnd.android.package-archive",
}

func getMimeType(name string) string {
	ext := path.Ext(name)
	if m, ok := extraMimeTypes[ext]; ok {
		return m
	}
	m := mime.TypeByExtension(ext)
	if m != "" {
		return m
	}
	return "application/octet-stream"
}

// getFileMd5 计算本地文件的MD5值
func getFileMd5(filename string) (string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hasher := md5.New()
	buffer := make([]byte, 1024*1024) // 1MB buffer
	_, err = io.CopyBuffer(hasher, file, buffer)
	if err != nil {
		return "", err
	}
	md5Bytes := hasher.Sum(nil)
	md5Str := hex.EncodeToString(md5Bytes)
	return md5Str, nil
}

// getFileSha1 计算本地文件的SHA1值
func getFileSha1(filename string) (string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hasher := sha1.New()
	buffer := make([]byte, 1024*1024) // 1MB buffer
	_, err = io.CopyBuffer(hasher, file, buffer)
	if err != nil {
		return "", err
	}
	sha1Bytes := hasher.Sum(nil)
	sha1Str := hex.EncodeToString(sha1Bytes)
	return sha1Str, nil
}

// genRandomWord 生成一个4位随机字谜
func genRandomWord() string {
	const letters = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	b := make([]byte, 4)
	rand.Seed(time.Now().UnixNano())

	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func md5Hash(key string) string {
	// 计算JSON数据的MD5
	hash := md5.Sum([]byte(key))
	return hex.EncodeToString(hash[:])
}
