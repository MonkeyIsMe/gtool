package string

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log"
	"math"
	"math/big"
	"reflect"
	"strings"
	"time"
	"unsafe"

	"github.com/MonkeyIsMe/gtool/constant"
	"github.com/antlabs/strsim"

	_ "github.com/go-sql-driver/mysql"
)

// NullStringToString nullstring转换成string
func NullStringToString(nullStrings []sql.NullString) []string {
	var strings []string
	for _, nullString := range nullStrings {
		if nullString.Valid {
			strings = append(strings, nullString.String)
		}
	}
	return strings
}

// SplitString 返回以partition分割的字符串数组
func SplitString(str, partition string) []string {
	splitStr := strings.Split(str, partition)
	return splitStr
}

// CaclMD5 返回str内容对应md5值(16进制表示)
func CaclMD5(str string) string {
	h := md5.New()
	h.Write([]byte(str))
	return hex.EncodeToString(h.Sum(nil))
}

// FilterEmptyString 过滤空字符串
func FilterEmptyString(src []string) []string {
	dest := make([]string, 0, len(src))
	for _, s := range src {
		if s == "" {
			continue
		}

		dest = append(dest, s)
	}
	return dest
}

// Base64Encode 对数据进行 base64 编码
func Base64Encode(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

// Base64Decode 对数据进行 base64 解码
func Base64Decode(s string) (string, error) {
	rs, err := base64.StdEncoding.DecodeString(s)
	return string(rs), err
}

// GetUniqueID 生成UniqueID方法
func GetUniqueID() string {
	// 当前毫秒时间戳
	timestamp := time.Now().UnixNano() / 1000000
	s1, _ := rand.Int(rand.Reader, big.NewInt(10))
	s2, _ := rand.Int(rand.Reader, big.NewInt(10))

	seqNo := timestamp * (s1.Int64() + 1) * (s2.Int64() + 1)
	uniqueID := fmt.Sprintf("%d", seqNo)
	return uniqueID
}

// IsStringSliceEqual 返回两个字符串列表是否相等。都为nil，返回true。其中之一为nil，返回false。
// 都不是nil，则长度内容顺序都一致才返回true。
func IsStringSliceEqual(lh, rh []string) bool {
	if lh == nil && rh == nil {
		return true
	}

	if lh == nil || rh == nil {
		return false
	}

	if len(lh) != len(rh) {
		return false
	}

	for k, v := range lh {
		if v != rh[k] {
			return false
		}
	}
	return true
}

// SplitSlice 将slice按长度batchSize分成多段
// batchHandler 返回false，表示退出执行
func SplitSlice(slice interface{}, batchSize int, batchHandler func(batch interface{}) bool) {
	if batchHandler == nil {
		return
	}

	rv := reflect.ValueOf(slice)
	if rv.Kind() != reflect.Slice {
		panic("argument not a slice")
	}

	blocks := int(math.Ceil(float64(rv.Len()) / float64(batchSize)))
	for i := 0; i < blocks; i++ {
		begin := i * batchSize
		end := begin + batchSize
		if end > rv.Len() {
			end = rv.Len()
		}

		batch := rv.Slice(begin, end)
		isContinue := batchHandler(batch.Interface())
		if !isContinue {
			break
		}
	}
}

// BytesToString copy-free的[]byte转string，但注意使用场景限制，不可滥用。
func BytesToString(bytes []byte) string {
	var s string
	sliceHeader := (*reflect.SliceHeader)(unsafe.Pointer(&bytes))
	stringHeader := (*reflect.StringHeader)(unsafe.Pointer(&s))
	stringHeader.Data = sliceHeader.Data
	stringHeader.Len = sliceHeader.Len
	return s
}

// NullStr 判断是否是空字符串
func NullStr(str string) bool {
	return len([]rune(str)) == 0
}

// NullStrWithDefault 空字符串赋予默认值
func NullStrWithDefault(str string, defaultStr string) (retStr string) {
	if len([]rune(str)) == 0 {
		return defaultStr
	}
	return str
}

// Similarity 计算两个字符串的相似度
func Similarity(s1 string, s2 string) float64 {
	sim := strsim.Compare(s1, s2)
	return sim
}

// StringFilter 字符串过滤，将1,2,3,4,5 -> '1','2','3','4','5'
func StringFilter(ids string) string {
	splitString := strings.Split(ids, constant.Comma)
	videos := fmt.Sprintf("'%s'", splitString[0])
	for i := 1; i < len(splitString); i++ {
		videos = fmt.Sprintf("%s,'%s'", videos, splitString[i])
	}
	return videos
}

func AesEncrypt(orig string, key string) string {
	// 转成字节数组
	origData := []byte(orig)
	k := []byte(key)

	// 分组秘钥
	block, err := aes.NewCipher(k)
	if err != nil {
		panic(fmt.Sprintf("key 长度必须 16/24/32长度: %s", err.Error()))
	}
	// 获取秘钥块的长度
	blockSize := block.BlockSize()
	// 补全码
	origData = PKCS7Padding(origData, blockSize)
	// 加密模式
	blockMode := cipher.NewCBCEncrypter(block, k[:blockSize])
	// 创建数组
	cryted := make([]byte, len(origData))
	// 加密
	blockMode.CryptBlocks(cryted, origData)
	//使用RawURLEncoding 不要使用StdEncoding
	//不要使用StdEncoding  放在url参数中回导致错误
	return base64.RawURLEncoding.EncodeToString(cryted)

}

func AesDecrypt(cryted string, key string) string {
	//使用RawURLEncoding 不要使用StdEncoding
	//不要使用StdEncoding  放在url参数中回导致错误
	crytedByte, _ := base64.RawURLEncoding.DecodeString(cryted)
	k := []byte(key)

	// 分组秘钥
	block, err := aes.NewCipher(k)
	if err != nil {
		panic(fmt.Sprintf("key 长度必须 16/24/32长度: %s", err.Error()))
	}
	// 获取秘钥块的长度
	blockSize := block.BlockSize()
	// 加密模式
	blockMode := cipher.NewCBCDecrypter(block, k[:blockSize])
	// 创建数组
	orig := make([]byte, len(crytedByte))
	// 解密
	blockMode.CryptBlocks(orig, crytedByte)
	// 去补全码
	orig = PKCS7UnPadding(orig)
	return string(orig)
}

// PKCS7Padding 补码
func PKCS7Padding(ciphertext []byte, blocksize int) []byte {
	padding := blocksize - len(ciphertext)%blocksize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

// PKCS7UnPadding 去码
func PKCS7UnPadding(origData []byte) []byte {
	length := len(origData)
	unpadding := int(origData[length-1])
	return origData[:(length - unpadding)]
}

// StringHandler 处理从数据库里查出来的字符串的信息
func StringHandler(results *sql.Rows) []string {
	characters := make([]string, 0)
	for results.Next() {
		character := ""
		err := results.Scan(&character)
		if err != nil {
			log.Printf("Scan string err: [%+v]", err)
			continue
		}

		characters = append(characters, character)
	}

	return characters
}

// NullStringHandler 处理从数据库里查出来的字符串的信息
func NullStringHandler(results *sql.Rows) []sql.NullString {
	characters := make([]sql.NullString, 0)
	for results.Next() {
		character := sql.NullString{}
		err := results.Scan(&character.String)
		if err != nil {
			log.Printf("Scan string err: [%+v]", err)
			continue
		}

		characters = append(characters, character)
	}

	return characters
}

// NumberHandler 处理从数据库里查出来的数字的信息
func NumberHandler(results *sql.Rows) []int {
	numbers := make([]int, 0)
	for results.Next() {
		number := 0
		err := results.Scan(&number)
		if err != nil {
			log.Printf("Scan number err: [%+v]", err)
			continue
		}

		numbers = append(numbers, number)
	}
	return numbers
}

// IsValidString 判断sql里面的数据是否合法
func IsValidString(nullString sql.NullString) string {
	if nullString.Valid {
		return nullString.String
	}

	return ""
}
