package main

func main() {
	logger = getLogger("smtp")
	config = readConfigJsonFile()
	relayConfig = readRelayConfigJsonFile(config.RelayConfigFile)
	if config.MysqlMode == 1 {
		mysqlDB = initMysqlConnect()
	}
	if config.RedisBloomMode == 1{
		initRedisBloom()
	}

	// 创建channel数组，用于分配每个协程的任务
	chs := make([]chan scanTarget,config.Concurrent)
	for i:=0;i<config.Concurrent;i++ {
		chs[i] = make(chan scanTarget,5)
	}
	stopCode := make(chan int,config.Concurrent)

	if config.ReadMode == "txt" {
		// txt模式暂时未完成
		logger.Printf("read_mode:txt\nexit\n")
	} else if config.ReadMode == "json" {
		go readMXJson(chs,config.JsonFile,0)
		for i:=0;i<config.Concurrent;i++ {
			go concurrentScan(i,chs[i],stopCode) // 开始并发扫描
		}
		for i:=0;i<config.Concurrent;i++ {
			// 开始执行主进程退出
			id := <- stopCode
			logger.Printf("goroutine %d exit\n",id)
		}
	} else {
		logger.Printf("read_mode error\n")
	}
}
