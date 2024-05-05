package net

import (
	"encoding/binary"
	"github.com/gin-gonic/gin"
	"net"
	"net/http"
	"strings"
)

const (
	XForwardedFor = "X-Forwarded-For"
	XRealIP       = "X-Real-IP"
)

const localhost = "127.0.0.1"

// RemoteIp 返回远程客户端的 IP，如 192.168.1.1
func RemoteIp(req *http.Request) string {
	remoteAddr := req.RemoteAddr
	if ip := req.Header.Get(XRealIP); ip != "" {
		remoteAddr = ip
	} else if ip = req.Header.Get(XForwardedFor); ip != "" {
		remoteAddr = ip
	} else {
		remoteAddr, _, _ = net.SplitHostPort(remoteAddr)
	}

	if remoteAddr == "::1" {
		remoteAddr = localhost
	}

	return remoteAddr
}

// Ip2long 将 IPv4 字符串形式转为 uint32
func Ip2long(ipstr string) uint32 {
	ip := net.ParseIP(ipstr)
	if ip == nil {
		return 0
	}
	ip = ip.To4()
	return binary.BigEndian.Uint32(ip)
}

// GenRequestIP 获取请求的IP
func GenRequestIP(ctx *gin.Context) string {
	ip := ctx.Request.RemoteAddr
	if len(ip) > 0 {
		return formatIP(ip)
	}

	ip = ctx.Request.Header.Get("X-Forwarded-For")
	if len(ip) > 0 {
		return formatIP(ip)
	}
	ip = ctx.Request.Header.Get("X-Real-IP")

	return formatIP(ip)
}

// formatIP 格式化IP
func formatIP(ip string) string {
	if strings.Contains(ip, "[::1]") {
		return localhost
	}

	return strings.Split(ip, ":")[0]
}
