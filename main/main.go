package main

import (
	"MyLOLassisitant/myLogs"
	"MyLOLassisitant/service"
	"log"
	"time"
)

func main() {
	// 初始化日志打印
	myLogs.InitLog()
	// 主流程
	if err := service.NewLolHelper().Run(); err != nil { //开始运行
		log.Fatal(err)
	}
	defer myLogs.CloseLogFile()
	time.Sleep(time.Second * 5)
}
