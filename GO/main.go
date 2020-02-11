package main

import (
	"fmt"
	"os"
	"time"

	"./bridge"
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
	//注册消息处理函数
	myBridge.RegisterFunc("test", TestHandler)
	myBridge.RegisterFunc("start", StartHunter)
	myBridge.RegisterFunc("stop", StopHunter)

	fmt.Println("Start ListenAndServer()...")
	//开始工作
	err = myBridge.ListenAndServer()
	if err != nil {
		logs.Error(err)
	}
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
	fmt.Println("TODO: startHunter(), content=" + content)
	return nil
}

//暂停正在进行的图片捕获功能
func PauseHunter(content string) error {
	fmt.Println("TODO: PauseHunter(), content=" + content)
	return nil
}

//终止正在进行的图片捕获功能
func StopHunter(content string) error {
	fmt.Println("TODO: StopHunter(), content=" + content)
	return nil
}

/*
func SignalHandler(signal *bridge.Unit) error {
	key := signal.Key
	var err error
	switch key {
	case "msg":
		logs.Info("%v", signal.Content)
		if err = tube.SendMessage("msg", fmt.Sprintf("<h1>%s</h1>", signal.Content)); err != nil {
			logs.Error(err)
			return err
		}

	case "start": //start huntting images
		if digger.IsRunning {
			err := errors.New("Digger is running")
			logs.Error(err)
			return err
		}
		logs.Info("%v", signal.Content)
		sizeLimit, numberLimit, threadLimit, minmun, maxmun, loggestWait, interval := 0, 0, 0, 0, 0, 0, 0
		startPoint, endPoint := 0, 0
		method, savePath, baseUrl, lineKey, targetKey := "", "", "", "", ""
		sucNum, err := fmt.Sscanf(signal.Content, "%s %s %d %d %d %d %d %d %d %s %s %s %d %d",
			&method, &savePath, &sizeLimit, &numberLimit, &threadLimit, &minmun, &maxmun, &loggestWait, &interval,
			&baseUrl, &lineKey, &targetKey, &startPoint, &endPoint)
		if err != nil {
			logs.Error(err)
			return err
		}
		if sucNum != 14 {
			err = errors.New("Success scanf numbers not right")
			logs.Error(err)
			return err
		}
		//restore blank character
		method = strings.ReplaceAll(method, "&npsp", " ")
		savePath = strings.ReplaceAll(savePath, "&npsp", " ")
		baseUrl = strings.ReplaceAll(baseUrl, "&npsp", " ")
		lineKey = strings.ReplaceAll(lineKey, "&npsp", " ")
		targetKey = strings.ReplaceAll(targetKey, "&npsp", " ")

		//check the savepath if exist
		if !checkDirExist(savePath) {
			err = fmt.Errorf("directory %s not exist", savePath)
			logs.Warn(err)
			return err
		}
		//setting up digger
		digger.SizeLimit = sizeLimit
		digger.NumberLimit = numberLimit
		digger.ThreadLimit = threadLimit
		digger.Minmun = minmun
		digger.Maxmun = maxmun
		digger.LongestWait = loggestWait
		digger.Interval = interval
		digger.SavePath = savePath
		if err = digger.CheckBaseConf(); err != nil {
			logs.Error(err)
			return err
		}
		//switch to different model according to method
		msg := make(chan string, 100)
		switch method {
		case "bfs":
			go digger.BFS_hunt(baseUrl, lineKey, targetKey, msg)
		case "dfs":
			go digger.DFS_hunt(baseUrl, lineKey, targetKey, msg)
		case "forloop":
			go digger.ForLoop_hunt(baseUrl, startPoint, startPoint, msg)
		case "urllist":
			go digger.UrlList_hunt(baseUrl, msg)
		default:
			err = fmt.Errorf("Unexpect method name: %s", method)
			logs.Error(err)
			defer close(msg)
			return err
		}
		//listen and send table data to qt mainwindows
		go func() {
			for {
				returnData, more := <-msg
				if !more {
					tube.SendMessage("info", "Images hunter is stop !")
					break
				}
				if err = tube.SendMessage("table", returnData); err != nil {
					logs.Error(err)
				}
			}
		}()
		//listen and send static data to qt mainwindows
		go func() {
			for {
				<-time.Tick(time.Second)
				staticData := digger.GetStatic()
				if staticData == "pause" {
					continue
				}
				if staticData == "end" {
					return
				}
				tube.SendMessage("static", staticData)
			}
		}()

	case "pause":
		digger.Pause()

	case "stop":
		digger.Stop()

	default:
		err = fmt.Errorf("Unexpect key name: %s", key)
		logs.Error(err)
		return err
	}
	return nil
}
*/

//========== tools function ==============

//check if a directory is exist  📂
func checkDirExist(dir string) bool {
	info, err := os.Stat(dir)
	if err != nil {
		if !os.IsNotExist(err) {
			logs.Error("error with file exist: %v", err)
		}
		return false
	}
	return info.IsDir()
}
