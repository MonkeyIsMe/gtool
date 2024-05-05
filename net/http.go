package net

import (
	"crypto/tls"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Cors 解决跨域问题
func Cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		origin := c.Request.Header.Get("Origin")
		if origin != "" {
			c.Header("Access-Control-Allow-Origin", "*")
			c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
			c.Header("Access-Control-Allow-Headers", "Content-Type,AccessToken,X-CSRF-Token, Authorization")
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Set("content-type", "application/json")
		}
		//放行所有OPTIONS方法
		if method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
		}
		c.Next()
	}
}

// DoGet get请求
func DoGet(req *http.Request) (map[string]string, error) {
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	rsp, err := client.Do(req)
	if err != nil {
		log.Printf("getToken client.Do err: [%+v]", err)
		return nil, err
	}
	defer rsp.Body.Close()

	body, _ := ioutil.ReadAll(rsp.Body)
	rspMap := make(map[string]string)
	err = json.Unmarshal(body, &rspMap)
	if err != nil {
		log.Printf("Unmarshal rsp body err:[%+v]", err)
		return nil, nil
	}

	return rspMap, nil
}
