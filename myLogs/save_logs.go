package myLogs

import (
	"MyLOLassisitant/consts"
	"fmt"
	"os"
	"time"
)

var fs *os.File

func InitLog() {
	if !Exists(consts.LogFileName) {
		_, _ = os.Create(consts.LogFileName)
	}
	var err error
	fs, err = os.OpenFile(consts.LogFileName, os.O_APPEND, 0666)
	if err != nil {
		panic(fmt.Sprintf("fail to openLogFile,err:", err))
	}
}

func Printf(format string, a ...interface{}) {
	info := fmt.Sprintf(format, a...)
	saveLogsInfo(info)
}

func Println(a ...interface{}) {
	info := fmt.Sprintln(a...)
	saveLogsInfo(info)
}

func saveLogsInfo(info string) {
	logInfo := fmt.Sprintf("%v:%v\n", time.Now().Format("2006-01-02 15:04:05"), info)
	_, err := fs.WriteString(logInfo)
	if err != nil {
		fmt.Println("fail to saveLogsInfo,err:", err)
		return
	}
}

func CloseLogFile() {
	fs.Close()
}

// 判断所给路径文件/文件夹是否存在
func Exists(path string) bool {
	_, err := os.Stat(path) //os.Stat获取文件信息
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}
