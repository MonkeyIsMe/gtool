package validator

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"net"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// VerifyEmailFormat 验证邮箱合法性
func VerifyEmailFormat(email string) bool {
	pattern := `^[0-9a-z][_.0-9a-z-]{0,31}@([0-9a-z][0-9a-z-]{0,30}[0-9a-z]\.){1,4}[a-z]{2,4}$`
	reg := regexp.MustCompile(pattern)
	return reg.MatchString(email)
}

// VerifyMobileFormat 验证手机号合法性
func VerifyMobileFormat(mobileNumber string) bool {
	regular := "^((13[0-9])|(14[5,7])|(15[0-3,5-9])|(17[0,3,5-8])|(18[0-9])|166|198|199|(147))\\d{8}$"
	reg := regexp.MustCompile(regular)
	return reg.MatchString(mobileNumber)
}

// SignIDCard 身份证号码处理
func SignIDCard(idcard string) string {
	cp := idcard
	leth := len(cp)
	return cp[0:4] + " **** **** " + cp[leth-4:]
}

// SignMobilePhone 手机号码处理
func SignMobilePhone(mobile string) string {
	cp := mobile
	leth := len(cp)
	return cp[0:3] + " **** " + cp[leth-4:]
}

// GetUniqueId 生成UniqueId方法
func GetUniqueId() string {
	// 当前毫秒时间戳
	timestamp := time.Now().UnixNano() / 1000000
	s1, _ := rand.Int(rand.Reader, big.NewInt(10))
	s2, _ := rand.Int(rand.Reader, big.NewInt(10))

	seqNo := timestamp * (s1.Int64() + 1) * (s2.Int64() + 1)
	uniqueId := fmt.Sprintf("%d", seqNo)
	return uniqueId
}

// VerifyTimeDate 判断时间格式是否正确 - xxxx-xx-xx 00:00:00
func VerifyTimeDate(content string) bool {
	reg := regexp.MustCompile(`^[1-9]\d{3}-(0[1-9]|1[0-2])-(0[1-9]|[1-2][0-9]|3[0-1])\s+(20|21|22|23|[0-1]\d):[0-5]\d:[0-5]\d$`)
	return reg.Match([]byte(content))
}

// VerifyPrice 验证价格
func VerifyPrice(content string) bool {
	reg := regexp.MustCompile(`(^[1-9]([0-9]+)?(\.[0-9]{1,2})?$)|(^(0){1}$)|(^[0-9]\.[0-9]([0-9])?$)`)
	return reg.Match([]byte(content))
}

// VerifyUrl 验证网络地址
func VerifyUrl(content string) bool {
	reg := regexp.MustCompile(`http(s)?:\/\/([\w-]+\.)+[\w-]+(\/[\w- .\/?%&=]*)?`)
	return reg.Match([]byte(content))
}

// VerifyBankCode 验证银行卡
func VerifyBankCode(content string) bool {
	reg := regexp.MustCompile(`^\d{16,21}$`)
	return reg.Match([]byte(content))
}

// VerifyInteger 验证整数
func VerifyInteger(content string) bool {
	reg := regexp.MustCompile(`^-?\\d+$`)
	return reg.Match([]byte(content))
}

// VerifyIdcard 身份证号正确性检查
func VerifyIdcard(idCard string) bool {
	idCard = strings.ToUpper(idCard)

	reg := regexp.MustCompile(`^[0-9]{17}[0-9X]$`)
	if reg.MatchString(idCard) == false {
		return false
	}
	return checkIdCardCode(idCard)
}

// 身份证校验码的计算方法：
//
//	1、将身份证号码前面的17位数分别乘以不同的加权因子，见： weights
//	2、将这17位数字和加权因子相乘的结果相加，得到的结果再除以11，得到余数 m
//	3、余数m作为位置值，在校验码数组 codes 中找到对应的值，就是身份证号码的第18位校验码
func checkIdCardCode(id string) bool {
	var weights []int = []int{7, 9, 10, 5, 8, 4, 2, 1, 6, 3, 7, 9, 10, 5, 8, 4, 2}
	var codes []string = []string{"1", "0", "X", "9", "8", "7", "6", "5", "4", "3", "2"}

	var sum int = 0
	for i := 0; i < 17; i++ {
		n, _ := strconv.Atoi(string(id[i]))
		sum += n * weights[i]
	}

	m := sum % 11

	return codes[m] == id[17:]
}

// IsAllUpper 是否全是大写
func IsAllUpper(str string) bool {
	for _, r := range str {
		if !unicode.IsUpper(r) {
			return false
		}
	}

	return str != ""
}

// IsAllLower 是否全是小写
func IsAllLower(str string) bool {
	for _, r := range str {
		if !unicode.IsLower(r) {
			return false
		}
	}

	return str != ""
}

var isAlphaRegexMatcher *regexp.Regexp = regexp.MustCompile(`^[a-zA-Z]+$`)

// IsAlpha 是否是字符
func IsAlpha(str string) bool {
	return isAlphaRegexMatcher.MatchString(str)
}

var containLetterRegexMatcher *regexp.Regexp = regexp.MustCompile(`[a-zA-Z]`)

// ContainLetter 判断是否包含字母
func ContainLetter(str string) bool {
	return containLetterRegexMatcher.MatchString(str)
}

// IsJSON 判断是不是json格式
func IsJSON(str string) bool {
	var js json.RawMessage
	return json.Unmarshal([]byte(str), &js) == nil
}

// IsNumberStr 判断字符串是否可以转成数字
func IsNumberStr(s string) bool {
	return IsIntStr(s) || IsFloatStr(s)
}

// IsFloatStr 判断字符串是否可以转成float
func IsFloatStr(str string) bool {
	_, e := strconv.ParseFloat(str, 64)
	return e == nil
}

var isIntStrRegexMatcher *regexp.Regexp = regexp.MustCompile(`^[\+-]?\d+$`)

// IsIntStr 判断字符串是否可以转成整数
func IsIntStr(str string) bool {
	return isIntStrRegexMatcher.MatchString(str)
}

// IsIP 判断字符串是否是ip
func IsIP(ipstr string) bool {
	ip := net.ParseIP(ipstr)
	return ip != nil
}

// IsIPV4 判断字符串是否是ipv4
func IsIPV4(ipstr string) bool {
	ip := net.ParseIP(ipstr)
	if ip == nil {
		return false
	}
	return strings.Contains(ipstr, ".")
}

// IsIPV6 判断字符串是否是ipv6
func IsIPV6(ipstr string) bool {
	ip := net.ParseIP(ipstr)
	if ip == nil {
		return false
	}
	return strings.Contains(ipstr, ":")
}

// IsPort 判断是不是个端口
func IsPort(str string) bool {
	if i, err := strconv.ParseInt(str, 10, 64); err == nil && i > 0 && i < 65536 {
		return true
	}
	return false
}

var isUrlRegexMatcher *regexp.Regexp = regexp.MustCompile(`^((ftp|http|https?):\/\/)?(\S+(:\S*)?@)?((([1-9]\d?|1\d\d|2[01]\d|22[0-3])(\.(1?\d{1,2}|2[0-4]\d|25[0-5])){2}(?:\.([0-9]\d?|1\d\d|2[0-4]\d|25[0-4]))|(([a-zA-Z0-9]+([-\.][a-zA-Z0-9]+)*)|((www\.)?))?(([a-z\x{00a1}-\x{ffff}0-9]+-?-?)*[a-z\x{00a1}-\x{ffff}0-9]+)(?:\.([a-z\x{00a1}-\x{ffff}]{2,}))?))(:(\d{1,5}))?((\/|\?|#)[^\s]*)?$`)

// IsURL 判断是否是个URL
func IsURL(str string) bool {
	if str == "" || len(str) >= 2083 || len(str) <= 3 || strings.HasPrefix(str, ".") {
		return false
	}
	u, err := url.Parse(str)
	if err != nil {
		return false
	}
	if strings.HasPrefix(u.Host, ".") {
		return false
	}
	if u.Host == "" && (u.Path != "" && !strings.Contains(u.Path, ".")) {
		return false
	}

	return isUrlRegexMatcher.MatchString(str)
}

var isDnsRegexMatcher *regexp.Regexp = regexp.MustCompile(`^[a-zA-Z]([a-zA-Z0-9\-]+[\.]?)*[a-zA-Z0-9]$`)

// IsDNS 判断字符串是否是DNS
func IsDNS(dns string) bool {
	return isDnsRegexMatcher.MatchString(dns)
}

var isEmailRegexMatcher *regexp.Regexp = regexp.MustCompile(`\w+([-+.]\w+)*@\w+([-.]\w+)*\.\w+([-.]\w+)*`)

// IsEmail 判断字符串是否是邮箱
func IsEmail(email string) bool {
	return isEmailRegexMatcher.MatchString(email)
}

var isChineseMobileRegexMatcher *regexp.Regexp = regexp.MustCompile("^((13[0-9])|(14[5,7])|(15[0-3,5-9])|(17[0,3,5-8])|(18[0-9])|166|198|199|(147))\\d{8}$")

// IsChineseMobile 判断字符串是否是手机号
func IsChineseMobile(mobileNum string) bool {
	return isChineseMobileRegexMatcher.MatchString(mobileNum)
}

var isChineseIdNumRegexMatcher *regexp.Regexp = regexp.MustCompile(`^[1-9]\d{5}(18|19|20|21|22)\d{2}((0[1-9])|(1[0-2]))(([0-2][1-9])|10|20|30|31)\d{3}[0-9Xx]$`)

// IsChineseIdNum 判断字符串是否是身份证号
func IsChineseIdNum(id string) bool {
	return isChineseIdNumRegexMatcher.MatchString(id)
}

var containChineseRegexMatcher *regexp.Regexp = regexp.MustCompile("[\u4e00-\u9fa5]")

// ContainChinese 判断字符串中是否含有中文字符
func ContainChinese(s string) bool {
	return containChineseRegexMatcher.MatchString(s)
}

var isChinesePhoneRegexMatcher *regexp.Regexp = regexp.MustCompile(`\d{3}-\d{8}|\d{4}-\d{7}`)

// IsChinesePhone 断字符串是否是座机电话号码
func IsChinesePhone(phone string) bool {
	return isChinesePhoneRegexMatcher.MatchString(phone)
}

var isCreditCardRegexMatcher *regexp.Regexp = regexp.MustCompile(`^(?:4[0-9]{12}(?:[0-9]{3})?|5[1-5][0-9]{14}|(222[1-9]|22[3-9][0-9]|2[3-6][0-9]{2}|27[01][0-9]|2720)[0-9]{12}|6(?:011|5[0-9][0-9])[0-9]{12}|3[47][0-9]{13}|3(?:0[0-5]|[68][0-9])[0-9]{11}|(?:2131|1800|35\\d{3})\\d{11}|6[27][0-9]{14})$`)

// IsCreditCard 判断字符串是否是信用卡
func IsCreditCard(creditCart string) bool {
	return isCreditCardRegexMatcher.MatchString(creditCart)
}

var isBase64RegexMatcher *regexp.Regexp = regexp.MustCompile(`^(?:[A-Za-z0-9+\\/]{4})*(?:[A-Za-z0-9+\\/]{2}==|[A-Za-z0-9+\\/]{3}=|[A-Za-z0-9+\\/]{4})$`)

// IsBase64 c判断字符串是base64
func IsBase64(base64 string) bool {
	return isBase64RegexMatcher.MatchString(base64)
}

// IsEmptyString 判断字符串是否为空
func IsEmptyString(str string) bool {
	return len(str) == 0
}

// IsRegexMatch 判断字符串是否match正则表达式
func IsRegexMatch(str, regex string) bool {
	reg := regexp.MustCompile(regex)
	return reg.MatchString(str)
}

// GetSecretKey 获取签名
func GetSecretKey(params map[string]interface{}, signature, nonce, appID,
	timestamp string) string {
	data := JoinStringsInASCII(params, "&", false, false, "")
	data = fmt.Sprintf("%s&key=%s&nonce=%s&appID=%s&timestamp=%s", data,
		signature, nonce, appID, timestamp)
	return fmt.Sprintf("%x", md5.Sum([]byte(data)))
}

// JoinStringsInASCII URL参数拼接
func JoinStringsInASCII(data map[string]interface{}, sep string, onlyValues,
	includeEmpty bool, exceptKeys ...string) string {
	var list []string
	var keyList []string
	m := make(map[string]int)
	if len(exceptKeys) > 0 {
		for _, except := range exceptKeys {
			m[except] = 1
		}
	}
	for k := range data {
		if _, ok := m[k]; ok {
			continue
		}
		value := data[k]
		if !includeEmpty && value == "" {
			continue
		}
		if onlyValues {
			keyList = append(keyList, k)
		} else {
			_, ok := value.(string)
			if ok {
				list = append(list, fmt.Sprintf("%s=%s", k, value))
			} else {
				list = append(list, fmt.Sprintf("%s=%d", k, value))
			}
		}
	}
	if onlyValues {
		sort.Strings(keyList)
		for _, v := range keyList {
			list = append(list, data[v].(string))
		}
	} else {
		sort.Strings(list)
	}
	return strings.Join(list, sep)
}
