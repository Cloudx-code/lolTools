package service

import (
	"MyLOLassisitant/consts"
	"MyLOLassisitant/myLogs"
	"MyLOLassisitant/utils"
	"github.com/yusufpapurcu/wmi"
	"golang.org/x/sys/windows"
	"os"
	"strconv"
	"strings"
	"syscall"
)

// GetLolClientPortToken 获取LOL端口号和token
func GetLolClientPortToken() (int, string, error) {
	//改成以管理员方式运行
	MustRunWithAdmin()

	type Process struct {
		Commandline string
	}
	var dst []Process
	err := wmi.Query("select * from Win32_Process  WHERE name='LeagueClientUx.exe'", &dst)

	if err != nil || len(dst) == 0 {
		myLogs.Println("fail to wmi.Query,err:", err)
		return 0, "", consts.ErrLolProcessNotFound
	}
	myLogs.Println("success to GetLolClientPortToken dst:", utils.ToJson(dst))

	btsChunk := consts.LOLCommandlineReg.FindSubmatch([]byte(dst[0].Commandline))
	if len(btsChunk) < 3 {
		myLogs.Println("fail to FindSubmatch,btsChunk:", utils.ToJson(btsChunk))
		return 0, "", consts.ErrLolProcessNotFound
	}
	myLogs.Println("btsChunk[0]:", string(btsChunk[0]))

	token := string(btsChunk[1])
	port, err := strconv.Atoi(string(btsChunk[2]))
	if err != nil {
		myLogs.Println("fail to strconv.Atoi(string(btsChunk[2])),err:", err)
		return 0, "", err
	}
	return port, token, nil
}

func IsAdmin() bool {
	_, err := os.Open("\\\\.\\PHYSICALDRIVE0")
	return err == nil
}
func MustRunWithAdmin() {
	if !IsAdmin() {
		RunMeElevated()
	}
}
func RunMeElevated() {
	verb := "runas"
	exe, _ := os.Executable()
	cwd, _ := os.Getwd()
	args := strings.Join(os.Args[1:], " ")
	verbPtr, _ := syscall.UTF16PtrFromString(verb)
	exePtr, _ := syscall.UTF16PtrFromString(exe)
	cwdPtr, _ := syscall.UTF16PtrFromString(cwd)
	argPtr, _ := syscall.UTF16PtrFromString(args)
	myLogs.Printf("exe:%v,exePtr:%v", exe, (*exePtr))
	myLogs.Printf("cwd:%v,cwdPtr:%v", cwd, (*cwdPtr))
	myLogs.Printf("args:%v,argPtr:%v,verb:%v,verbPtr:%v", args, (*argPtr), verb, (*verbPtr))
	var showCmd int32 = 1 // SW_NORMAL
	err := windows.ShellExecute(0, verbPtr, exePtr, argPtr, cwdPtr, showCmd)
	if err != nil {
		myLogs.Println("fail to windows.ShellExecute,err:", err)
	}
	os.Exit(-2)
}
