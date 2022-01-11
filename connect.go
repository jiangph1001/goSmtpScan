package main

import (
	"fmt"
	"log"
	"net"
	"regexp"
	"strings"
)

func getResponse(conn net.Conn) string {
	buf := make([]byte, 1024)
	cnt, err := conn.Read(buf)
	if err != nil {
		logger.Printf("get response error: %s\n", err)
		return ""
	}
	return string(buf[0:cnt])
}

// SendCommand 发送命令
// smtpLogger 每一个扫描的host有一个单独的logger
func SendCommand(conn net.Conn, command string,smtpLogger *log.Logger) string {
	writeLog(smtpLogger, ">> "+command, config.SendLogScreen)
	fmt.Fprintf(conn, command)
	response := getResponse(conn)
	writeLog(smtpLogger, "<< "+response, config.ReceiveLogScreen)
	return response
}


// 解析服务器发过来的第一条响应
// 220 mail.imapmax.xyz smtp4dev ready
// 匹配结果 (220 mail.imapmax.xyz smtp4dev ready) (220) (mail.imapmax.xyz) (.xyz) (smtp4dev ready)
// 返回 mail.imapmax.xyz smtp4dev
func parseFirstResponse(response string) (string,string){
	// 部分服务器的格式是220-asp-relay.et.namecheap.tech ESMTP Postfix
	if len(response)>3 && response[3] == '-' {
		byteResponse := []byte(response)
		byteResponse[3] = ' '
		response = string(byteResponse)
	}
	regAll := regexp.MustCompile("^(\\d+) ([a-zA-Z0-9][-a-zA-Z0-9]{0,62}(\\.[a-zA-Z0-9][-a-zA-Z0-9]{0,62})+\\.?) (.*)")
	matched := regAll.FindStringSubmatch(response)
	var smtpHost, smtpServer string
	if len(matched) == 5 {
		smtpHost = matched[2]
		smtpServer = strings.TrimSpace(matched[4])
	} else {
		// 无客户端的情况 ，例如 220 mail.imapmax.xyz
		// 匹配结果 (220 mail.imapmax.xyz smtp4dev ready) (220) (mail.imapmax.xyz) (.xyz)
		regExcludeServer := regexp.MustCompile("^(\\d+) ([a-zA-Z0-9][-a-zA-Z0-9]{0,62}(\\.[a-zA-Z0-9][-a-zA-Z0-9]{0,62})+\\.?)")
		matched = regExcludeServer.FindStringSubmatch(response)
		if len(matched) == 4 {
			smtpHost = matched[2]
			smtpServer = "None"
		} else {
			logger.Println("parse 1st response failed:[",response,"]")
			smtpHost = ""
			smtpServer = "None"
		}
	}
	return smtpHost, smtpServer
}



//  250 New message started
// 	500 Command unrecognised
func parseResponse(response string) (string,string) {
	regAll := regexp.MustCompile("^(\\d+) (.*)")
	matched := regAll.FindStringSubmatch(response)
	if len(matched) == 3 {
		return matched[1],matched[2]
	}
	logger.Println("parse response failed:[",response,"]")
	return "000","error"
}
