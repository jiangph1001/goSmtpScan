package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
)
var relayConfig RelayConfig

func SingleTest(conn net.Conn,mailFrom string,rcptTo string) int {
	var commandList []string
	var responseList []string
	mailFromCommand := fmt.Sprintf("MAIL FROM:<%s>\r\n", mailFrom)
	rcptToCommand := fmt.Sprintf("RCPT TO:<%s>\r\n", rcptTo)
	commandList = append(commandList, mailFromCommand)
	commandList = append(commandList, rcptToCommand)
	commandList = append(commandList, "RESET")
	for _, command := range commandList {
		response := SendCommand(conn, command)
		responseList = append(responseList,response)
	}
	for i,response := range responseList {
		//statusCode,statusInfo := parseResponse(response)
		statusCode,_ := parseResponse(response)

	}

	return 0
}

func GetCommand() []string {
	var commandList []string
	helloCommand := fmt.Sprintf("EHLO %s\r\n", "greynius")
	commandList = append(commandList, helloCommand)

	commandList = append(commandList, "QUIT\r\n")
	return commandList
}
func getResCode(response string) int {
	resCode, _ := strconv.ParseInt(response[:3], 0, 32)
	return int(resCode)
}

func ScanHost(host string,port int) {
	address := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.Dial("tcp", address)
	if err != nil {
		logger.Fatal("connect", address, "fail: ", err)
	}
	defer conn.Close()
	logger.Printf("connect %s success: ", address)
	connIpv4 := conn.RemoteAddr().(*net.TCPAddr).IP.String()
	// 获取服务器的身份响应
	firstResponse := getResponse(conn)
	smtpHost, smtpServer := parseFirstResponse(firstResponse)
	envelopes := []mailEnvelope {
		mailEnvelope{
			"",
			fmt.Sprintf("%s@%s",relayConfig.RelayUser,relayConfig.RelayDomain)},
		mailEnvelope{
			fmt.Sprintf("%s@%s",relayConfig.FromUser,relayConfig.RelayDomain),
			fmt.Sprintf("%s@%s",relayConfig.RelayUser,relayConfig.RelayDomain)},
		mailEnvelope{
			fmt.Sprintf("%s@%s",relayConfig.FromUser,smtpHost),
			fmt.Sprintf("%s@%s",relayConfig.RelayUser,relayConfig.RelayDomain)},
		mailEnvelope{
			fmt.Sprintf("%s@[%s]",relayConfig.FromUser,connIpv4),
			fmt.Sprintf("%s@%s",relayConfig.RelayUser,relayConfig.RelayDomain)},
		mailEnvelope{
			fmt.Sprintf("%s@[%s]",relayConfig.FromUser,connIpv4),
			fmt.Sprintf("%%s",relayConfig.RelayUser,relayConfig.RelayDomain)},
	}

	// 开始测试

	for i,envelope := range envelopes {
		SingleTest(conn,envelope.MailFrom,envelope.RcptTo)

	}

}

func main() {
	logger = getLogger()
	config = readConfigJsonFile()
	relayConfig = readRelayConfigJsonFile(config.RelayConfigFile)
	f, err := os.Open(config.HostFile)
	if err != nil {
		fmt.Println(err.Error())
	}
	buf := bufio.NewReader(f)
	for {
		data, _, eof := buf.ReadLine()
		if eof == io.EOF {
			break
		}
		fmt.Printf("scaning %s:%d",string(data),25)
		go ScanHost(string(data),25)  // Goroutine
	}
}
