package main

import (
	"bufio"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"io/ioutil"
	"log"
	"os"
	"sync"
	"time"
)

var config Config
var logger *log.Logger
var mutex sync.RWMutex
var mysqlDB *sql.DB
var channel = make(chan scanTarget,1000)
type Config struct {
	HostFile         string `json:"host_file"`
	RelayConfigFile  string `json:"relay_config_file"`
	SendLogScreen    int    `json:"send_log_screen"`
	ReceiveLogScreen int    `json:"receive_log_screen"`
	CsvMode          int    `json:"csv_mode"`
	CsvFile          string `json:"csv_file"`
	MysqlMode        int    `json:"mysql_mode"`
	MysqlConfig      struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Host     string `json:"host"`
		Port     int    `json:"port"`
		Database string `json:"database"`
		Table  string `json:"table"`
		Column string `json:"column"`
	} `json:"mysql_config"`
	RedisBloomMode int    `json:"redis_bloom_mode"`
	RedisBloomHost string `json:"redis_bloom_host"`
	RedisBloomKeys string `json:"redis_bloom_keys"`
}
type fdnsRecord struct {
	Timestamp string `json:"timestamp"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	Value     string `json:"value"`
}

type scanTarget struct {
	Name   string
	Value  string
}

type MysqlRow struct {
	TimeStamp int
	Name string
	Type string
	Value string
}

type RelayConfig struct {
	FromUser   string `json:"from_user"`
	FromDomain string `json:"from_domain"`
	ToUser     string `json:"to_user"`
	ToDomain   string `json:"to_domain"`
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

// printScreenFlag 用于决定是否将日志输出到屏幕，由配置文件控制
func writeLog(logger *log.Logger, record string, printScreenFlag int) {
	logger.Print(record)
	if printScreenFlag == 1 {
		log.Print(record)
	}
}

// 读取包含MX记录的json文件
func readMXJson(jsonFile string,breakCnt int) {
	fp, err := os.OpenFile(jsonFile, os.O_RDONLY, 0755)
	if err != nil {
		fmt.Printf("Error open mx json file:%s:\n %s\n",jsonFile, err)
		return
	}
	scanner := bufio.NewScanner(fp)
	i := 0
	for scanner.Scan() {
		jsonMessage := scanner.Text() //按行读取文件
		//fmt.Println(jsonMessage)
		record := fdnsRecord{}
		err := json.Unmarshal([]byte(jsonMessage), &record)
		if err != nil {
			logger.Printf("parse json error:%s\n",err)
		}
		//fmt.Println(record)
		if record.Type == "mx" {
			var value string
			var high int
			fmt.Sscanf(record.Value,"%d %s.",&high,&value)
			st := scanTarget{record.Name,value[:len(value)-1]}
			channel <- st
		}
		i += 1
		if i>= breakCnt && breakCnt > 0{
			break
		}
	}
}

// 初始化mysql连接
func initMysqlConnect() *sql.DB {
	dsn := fmt.Sprintf("%s:%s@%s(%s:%d)/%s",config.MysqlConfig.Username,config.MysqlConfig.Password, "tcp",config.MysqlConfig.Host,config.MysqlConfig.Port,config.MysqlConfig.Database)
	mysqlDB, err := sql.Open("mysql", dsn)
	if err != nil{
		log.Fatalf("Open mysql failed,err:%v\n",err)
		return nil
	}
	mysqlDB.SetConnMaxLifetime(100*time.Second)  //最大连接周期，超过时间的连接就close
	mysqlDB.SetMaxOpenConns(2000)//设置最大连接数
	mysqlDB.SetMaxIdleConns(16) //设置闲置连接数
	creatTableSql := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (" +
		"`id` int unsigned NOT NULL AUTO_INCREMENT,\n" +
		"`timestamp` int DEFAULT NULL,\n" +
		"`mx_host` varchar(100) NOT NULL,\n" +
		"`ip` varchar(100) DEFAULT NULL,\n" +
		"`port` int DEFAULT NULL,\n" +
		"`smtp_host` varchar(100) NOT NULL,\n" +
		"`smtp_server` varchar(200) NOT NULL,\n" +
		"`case_num` int not null,\n" +
		"`success_cnt` int not null,\n" +
		"`skip_cnt` int not null,\n" +
		"`test_score` int not null,\n" +
		"PRIMARY KEY (`id`)\n" +
	") ENGINE=InnoDB AUTO_INCREMENT=712507 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci",config.MysqlConfig.Table)
	_,err = mysqlDB.Exec(creatTableSql)
	if err!=nil{
		logger.Fatal("create table error,%s\n",err.Error())
	}
	return mysqlDB
}

// 将信息写入mysql
func writeMysql(data []string) {
	//dataString := strings.Join(data , ",")
	dataString := fmt.Sprintf("%s,\"%s\",\"%s\",%s,\"%s\",\"%s\",%s,%s,%s,%s",
		data[0],data[1],data[2],data[3],data[4],data[5],data[6],data[7],data[8],data[9])
	sql := fmt.Sprintf("insert into %s(%s) values (%s) ",config.MysqlConfig.Table,config.MysqlConfig.Column,dataString)
	// INSERT INTO smtp_result (timestamp, mx_host, ip, port, smtp_host
	//				,smtp_server, case_num, success_cnt, skip_cnt, test_score)
	//      VALUES (1641534044, 'mx.imapmax.xyz', '82.157.127.171', 25, 'mail.imapmax.xyz'
	//	          ,'smtp4dev ready', 5, 5, 0, 31)
	_,err := mysqlDB.Exec(sql)
	if err != nil {
		fmt.Printf("insert data err:%s\n", err.Error())
		return
	}
}


// data 要写入的单行数据
// csvFile csv文件名，为空则读取配置文件
func writeCsv(data []string,csvFile string) {
	if csvFile == "" {
		csvFile = config.CsvFile
	}
	File,err:=os.OpenFile(fmt.Sprintf("result/%s",csvFile),os.O_RDWR|os.O_APPEND|os.O_CREATE,0666)
	if err!=nil{
		logger.Fatal(config.CsvFile,"csv文件打开失败！")
	}
	defer File.Close()

	// 加锁
	mutex.Lock()
	//创建写入接口
	WriterCsv:=csv.NewWriter(File)

	//写入一条数据，传入数据为切片(追加模式)
	err1:=WriterCsv.Write(data)
	if err1!=nil{
		logger.Fatal(config.CsvFile,"csv文件写入失败！")
	}
	WriterCsv.Flush() //刷新，不刷新是无法写入的

	// 解锁
	mutex.Unlock()
}
