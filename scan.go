package main

import (
	"fmt"
	tld "github.com/jpillora/go-tld"
	"log"
	"net"
	"strconv"
	"time"
)
var relayConfig RelayConfig

// SingleTest
// 0 测试通过
// 1 测试不通过
// -1 未执行测试
func SingleTest(conn net.Conn,mailFrom string,rcptTo string) int {
	var commandList []string
	var responseList []string
	mailFromCommand := fmt.Sprintf("MAIL FROM:<%s>\r\n", mailFrom)
	rcptToCommand := fmt.Sprintf("RCPT TO:<%s>\r\n", rcptTo)
	commandList = append(commandList, mailFromCommand)
	commandList = append(commandList, rcptToCommand)
	//commandList = append(commandList, "RSET\r\n")

	// 统一发送请求
	for _, command := range commandList {
		response := SendCommand(conn, command)
		responseList = append(responseList,response)
	}

	//统一解析响应
	var statusCodeList []string
	for _,response := range responseList {
		//statusCode,statusInfo := parseResponse(response)
		statusCode,_ := parseResponse(response)
		statusCodeList = append(statusCodeList, statusCode)
	}

	//分析响应码，不为250即不通过
	for _,statusCode := range statusCodeList {
		if statusCode != "250" {
			return 1
		}
	}
	return 0
}

// SingleTestOptimize
// 优化后的单个测试用例，合并发送，理论上能加快单个的测试速度，实际效果待更多的测试
func SingleTestOptimize(conn net.Conn,mailFrom string,rcptTo string) int {
	command := fmt.Sprintf("MAIL FROM:<%s>\r\nRCPT TO:<%s>\n", mailFrom,rcptTo)
	//commandList = append(commandList, "RSET\r\n")

	// 统一发送请求
	response := SendCommand(conn, command)
	response2 := getResponse(conn)

	responseList := []string{
		response,response2}

	var statusCodeList []string
	for _,response := range responseList {
		//statusCode,statusInfo := parseResponse(response)
		statusCode,_ := parseResponse(response)
		statusCodeList = append(statusCodeList, statusCode)
	}
	//fmt.Println(statusCodeList)
	for _,statusCode := range statusCodeList {
		if statusCode != "250" {
			return 1
		}
	}
	return 0
}

//发送RSET指令，若该指令执行不通过，则后续不再执行
func rsetSession(conn net.Conn) int {
	response :=SendCommand(conn, "RSET\r\n")
	statusCode,_ := parseResponse(response)
	if statusCode != "250" {
		return 1
	}
	return 0
}

func getResCode(response string) int {
	resCode, _ := strconv.ParseInt(response[:3], 0, 32)
	return int(resCode)
}

// 只保留域名的Tld和domain
// 输入 mx.qq.com 输出qq.com
// 引用的第三方库go-tld需要在前面补http://才能运行
// 解析失败会返回 .
func parseUrl(url string) string {
	newUrl := fmt.Sprintf("http://%s",url)
	u,err := tld.Parse(newUrl)
	if err != nil {
		u,_ = tld.Parse(url)
	}
	parsedUrl := fmt.Sprintf("%s.%s",u.Domain,u.TLD)
	return parsedUrl
}


// ScanHost connectHost + port
// 例如构造一封@163.com的邮件，需要连接mail.163.com
// mailHost 163.com
// connectHost mail.163.com
func ScanHost(connectHost string,port int,mailHost string) {
	address := fmt.Sprintf("%s:%d", connectHost, port)
	conn, err := net.Dial("tcp", address)
	if err != nil {
		logger.Fatal("connect", address, "fail: ", err)
	}
	defer conn.Close()
	logger.Printf("connect %s success: ", address)
	connIpv4 := conn.RemoteAddr().(*net.TCPAddr).IP.String()  // IP地址

	// 获取服务器的身份响应
	firstResponse := getResponse(conn)
	smtpHost, smtpServer := parseFirstResponse(firstResponse)

	// 可能有三个不同的域名
	// 1. 服务器返回的smtpHost解析后仅保留Domain和TLD
	// 2. mailHost 即dns解析前的key值
	// 3. connectHost解析后仅保留Domain和TLD

	var fromDomainList []string

	if mailHost != "" {
		fromDomainList = append(fromDomainList,mailHost)
	}
	parsedMailHost := parseUrl(mailHost)
	if parsedMailHost != "." && parsedMailHost != mailHost {
		fromDomainList = append(fromDomainList,parsedMailHost)
	}
	parsedConnectHost := parseUrl(connectHost)
	if parsedConnectHost != "." && parsedConnectHost != mailHost && parsedConnectHost != parsedMailHost{
		fromDomainList = append(fromDomainList,parsedConnectHost)
	}

	// 构造测试用例
	envelopes := []mailEnvelope {
		mailEnvelope{
			// 无MailFrom
			// 发往第三方服务器
			"",
			fmt.Sprintf("%s@%s",relayConfig.ToUser,relayConfig.ToDomain)},
		mailEnvelope{
			// 标准测试
			fmt.Sprintf("%s@%s",relayConfig.FromUser,relayConfig.FromDomain),
			fmt.Sprintf("%s@%s",relayConfig.ToUser,relayConfig.ToDomain)},
		mailEnvelope{
			// mailFrom以IP作为域名
			fmt.Sprintf("%s@[%s]",relayConfig.FromUser,connIpv4),
			fmt.Sprintf("%s@%s",relayConfig.ToUser,relayConfig.ToDomain)},
	}

	for _,domain := range fromDomainList {
		envelopes = append(envelopes, mailEnvelope{
			fmt.Sprintf("%s@%s", relayConfig.FromUser, domain),
			fmt.Sprintf("%s@%s", relayConfig.ToUser, relayConfig.ToDomain)})
	}

	// 开始测试
	SendCommand(conn, "EHLO imapmax.xyz\r\n")
	var testInfo = []string {
		strconv.FormatInt(time.Now().Unix(),10), // 时间戳
		connectHost,   // 实际连接的host
		connIpv4,      // 实际连接的ip
		strconv.Itoa(port),   // 实际连接的端口
		smtpHost,   // smtp响应的host，不一定和连接的host一致
		smtpServer, // smtp响应的server
		strconv.Itoa(len(envelopes))}  // case_num 测试用例数目

	skipTest := 0 // 代表是否执行测试
	successCnt := 0 // 成功次数
	//var testResultList []string

	var testResultScore = 0 // 用二进制的方式来表示测试的通过，例如101记录为5
	var skipCount = 0  // 跳过的测试数量
	for _,envelope := range envelopes {
		if skipTest == 1 {
			// 跳过测试，该记录直接记为-1
			skipCount += 1
			//testResultList = append(testResultList,"-1")
		} else {
			// 此处调用单次测试，

			testResult := SingleTest(conn,envelope.MailFrom,envelope.RcptTo)
			//testResult := SingleTestOptimize(conn,envelope.MailFrom,envelope.RcptTo)
			if testResult == 0 {
				// 测试成功
				successCnt += 1
				testResultScore = testResultScore*2 + 1
			}
			// testResultList = append(testResultList,strconv.Itoa(testResult))
			// 发送rset请求，如果失败则跳过后续测试
			skipTest = rsetSession(conn)
		}
	}
	testInfo = append(testInfo,strconv.Itoa(successCnt))
	testInfo = append(testInfo,strconv.Itoa(skipCount))
	testInfo = append(testInfo,strconv.Itoa(testResultScore))

	log.Printf("scan %s:%d %d success",connectHost,port,successCnt)
	if config.CsvMode == 1 {
		writeCsv(testInfo,"")
	}
	if config.MysqlMode == 1 {
		writeMysql(testInfo)
	}

}