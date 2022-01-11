# openRelayScan

基于go开发的open relay多线程扫描工具


## 预期功能

- [x]  连接smtp服务器，扫描开放转发
- [x]  通过redisbloom防止重复查询
- [x]  高并发
- [x]  测试结果异步写入mysql/clickhouse
- [x]  测试结果记录csv，加锁
- [x]  为每个服务器单独记录会话
- [ ]  自定义测试用例
- [x]  开发完成
- [ ]  发送真实邮件



## bug修复
- [x]  连接未正常关闭
- [ ]  logger开启数量过多（协程过多时出现）
- [x]  修复部分响应的解析出错
- [ ]  检查是否正确响应ehlo
- [ ]  redisbloom清空功能

## 代码结构

- main.go 主函数
- connect.go  发请求和解析响应相关的代码
- handle.go 一些配置文件相关
- scan.go 关于扫描啊，测试用例这样的
- test.go 相当于单元测试，就不git同步了


## 编译运行
安装依赖
```
go get github.com/jpillora/go-tld
go get github.com/go-sql-driver/mysql
go get -u github.com/panjf2000/ants/v2   
```
编译
```
go build test.go handle.go connect.go scan.go
go build main.go handle.go connect.go scan.go
```

## 一些思考

1. 为了防止个别服务器不认，还是加上了ehlo，等以后再测，如果影响不大再去掉。反正QUIT是不可能加的了
2. 会收集服务器的类型，用于以后做分析