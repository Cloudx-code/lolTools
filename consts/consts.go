package consts

import "regexp"

var (
	//获取port和token时的正则匹配
	LOLCommandlineReg = regexp.MustCompile(`--remoting-auth-token=(.+?)" "--app-port=(\d+)"`)
)

const (
	//获取token，port重试次数
	TryTimes = 10
	//日志存放位置
	LogFileName = `myLogs/日志.txt`
)
