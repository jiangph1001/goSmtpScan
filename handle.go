package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sync"
	"time"
)

var config Config
var logger *log.Logger
var mutex sync.RWMutex

type Config struct {
	HostFile         string `json:"host_file"`
	RelayConfigFile  string `json:"relay_config_file"`
	SendLogScreen    int    `json:"send_log_screen"`
	ReceiveLogScreen int    `json:"receive_log_screen"`
	ResultMode       string `json:"result_mode"`
	ResultFile       string `json:"result_file"`
	RedisBloomHost   string `json:"redis_bloom_host"`
	RedisBloomKeys   string `json:"redis_bloom_keys"`
}

type RelayConfig struct {
	RelayDomain string `json:"relay_domain"`
	RelayUser   string `json:"relay_user"`
	FromUser    string `json:"from_user"`
}

type mailEnvelope struct {
	MailFrom string `json:"mail_from"`
	RcptTo   string `json:"rcpt_to"`
}
func readConfigJsonFile() Config {
	jsonContent, err := ioutil.ReadFile("config.json")
	if err != nil {
		fmt.Printf("Open config file failed [Err:%s]\n", err.Error())
	}
	config := Config{}
	err = json.Unmarshal(jsonContent, &config)
	if err != nil {
		fmt.Println("解析数据失败", err)
	}
	return config
}

func readRelayConfigJsonFile(filename string) RelayConfig {
	jsonContent, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Printf("Open relay config file failed [Err:%s]\n", err.Error())
	}
	config := RelayConfig{}
	err = json.Unmarshal(jsonContent, &config)
	if err != nil {
		fmt.Println("解析数据失败", err)
	}
	return config
}
func getLogger() *log.Logger {
	currentTime := time.Now()
	t1 := currentTime.Year()   //年
	t2 := currentTime.Month()  //月
	t3 := currentTime.Day()    //日
	t4 := currentTime.Hour()   //小时
	t5 := currentTime.Minute() //分钟
	logName := fmt.Sprintf("log/%d-%d-%d-%d-%d.log", t1, t2, t3, t4, t5)
	file, err := os.OpenFile("log/smtp.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("fail to create %s file:%s\n", logName, err)
	}
	logger := log.New(file, "", log.LstdFlags)
	return logger
}

func writeLog(logger *log.Logger, record string, flag int) {
	logger.Print(record)
	if flag == 1 {
		log.Print(record)
	}
}

func writeCsv(data []string) {
	File,err:=os.OpenFile(config.ResultFile,os.O_RDWR|os.O_APPEND|os.O_CREATE,0666)
	if err!=nil{
		logger.Fatal(config.ResultFile,"csv文件打开失败！")
	}
	defer File.Close()

	// 加锁
	mutex.Lock()
	//创建写入接口
	WriterCsv:=csv.NewWriter(File)

	//写入一条数据，传入数据为切片(追加模式)
	err1:=WriterCsv.Write(data)
	if err1!=nil{
		logger.Fatal(config.ResultFile,"csv文件写入失败！")
	}
	WriterCsv.Flush() //刷新，不刷新是无法写入的

	// 解锁
	mutex.Unlock()
}
