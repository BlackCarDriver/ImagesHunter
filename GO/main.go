package main

import (
	"fmt"
	"os"
	"time"

	"./bridge"
	"./digger"
	"github.com/astaxie/beego/logs"
)

var myBridge *bridge.Bridge

func main() {
	logs.EnableFuncCallDepth(true)
	logs.SetLogFuncCallDepth(3)
	var err error
	//创建Bridge实例
	myBridge, err = bridge.GetBridge(1024*100, 4747)
	if err != nil {
		logs.Error(err)
		os.Exit(1)
	}
	test1()
	//注册消息处理函数
	myBridge.RegisterFunc("test", TestHandler)
	myBridge.RegisterFunc("start", StartHunter)
	myBridge.RegisterFunc("pause", PauseHunter)
	myBridge.RegisterFunc("continue", ContinueHunter)
	myBridge.RegisterFunc("stop", StopHunter)
	//等待并处理图片保存状况信息
	go func() {
		err := waitAndSendResult()
		if err != nil {
			logs.Error("Can't setup result senter: err=%v", err)
			os.Exit(1)
		}
		return
	}()
	fmt.Println("Start ListenAndServer()...")
	//开始工作
	err = myBridge.ListenAndServer()
	if err != nil {
		logs.Error(err)
	}
}

//模拟 QT 端点击开始按钮
func test1() {
	content := "BFS D:/TEMP 100 2 5 5 10240 10 0 http://www.ruanyifeng.com/blog/ &empty &empty 0 0"
	err := StartHunter(content)
	if err != nil {
		logs.Error(err)
	} else {
		logs.Info("Test complete!")
	}
	time.Sleep(10 * time.Second)
	os.Exit(0)
}

//功能性测试
func test2() {
	time.Sleep(time.Second * 1)
	digger.TEST1()
}

//=====================

//等待digger发来的图片下载报告，直接发送给qt端显示
//会造成堵塞，需要在新协程中运行
func waitAndSendResult() error {
	var err error
	resultChan := make(chan string, 10)
	if err = digger.SetupResultChan(&resultChan); err != nil {
		logs.Error(err)
	}
	for {
		result, more := <-resultChan
		if more {
			err = myBridge.SendMessage("table", result)
			if err != nil {
				logs.Warn(err)
			}
			continue
		}
		break
	}
	return nil
}

//===================== 消息处理函数 ==============

//测试接口,可临时代替SignalHandler
func TestHandler(content string) error {
	logs.Info("content=", content)
	for i := 0; ; i++ {
		logs.Info("TestHandler, i=", i)
		time.Sleep(3000)
		switch i {
		case 0:
			myBridge.SendMessage("debug", "It is a test....")
		case 1:
			myBridge.SendMessage("error", "It is a test....")
		case 2:
			myBridge.SendMessage("info", "It is a test...")
		case 3:
			myBridge.SendMessage("table", "https://urlofimages/urlofimages/urlofimages/urlofimages/example.png 1234.kb ok name.png")
		case 4:
			myBridge.SendMessage("static", "1 2 3 3 4.56KB/s 78mb 90")
		default:
			return nil
		}
	}
}

//开始图片获取功能
func StartHunter(content string) error {
	var err error
	logs.Info("TODO: startHunter(), content=" + content)
	err = digger.StartDigger(content) //不堵塞
	if err != nil {
		logs.Error(err)
		return err
	}
	return nil
}

//暂停正在进行的图片捕获功能
func PauseHunter(content string) error {
	var err error
	fmt.Println("TODO: PauseHunter(), content=" + content)
	err = digger.PauseDigger()
	if err != nil {
		logs.Error(err)
		return err
	}
	return nil
}

//恢复正在暂停的任务
func ContinueHunter(content string) error {
	var err error
	fmt.Println("TODO: ContinueHunter(), content=" + content)
	err = digger.ContinueDigger()
	if err != nil {
		logs.Error(err)
		return err
	}
	return nil
}

//终止正在进行的图片捕获功能
func StopHunter(content string) error {
	var err error
	fmt.Println("TODO: StopHunter(), content=" + content)
	err = digger.StopDigger()
	if err != nil {
		logs.Error(err)
		return err
	}
	return nil
}
