package main

import (
	"fmt"
	"os"

	"./bridge"
	"github.com/astaxie/beego/logs"
)

var tube *bridge.Bridge

func main() {
	logs.EnableFuncCallDepth(true)
	logs.SetLogFuncCallDepth(3)
	var err error
	tube, err = bridge.GetBridge(1024*100, 4747) //参数为最大传输单元和端口
	if err != nil {
		logs.Error(err)
		os.Exit(1)
	}
	msgChan := make(chan *bridge.Unit, 100) //msgChan receive the message send by qt mainwindow
	go tube.Start(msgChan)                  //这个协程等待、处理并通过msgChan返回qt端数据
	runResult := <-msgChan
	//msgChan返回的第一条消息用于表示是否连接成功
	if runResult.Key == "[fail]" {
		logs.Error("Start fail: %s", runResult.Content)
		os.Exit(1)
	} else {
		logs.Info("Run success: %s", runResult.Key)
	}
	//开始等待以处理好的消息到来
	for {
		newMsg, more := <-msgChan
		if more {
			err := SignalHandler(newMsg)
			if err != nil {
				logs.Error(err)
				tube.SendMessage("error", fmt.Sprint(err))
			}
		} else {
			logs.Info("Go socket client close!")
			break
		}
	}
}

func test() {
	//digger.DigPWithClass()
}

func SignalHandler(signal *bridge.Unit) error {
	logs.Info(signal)
	tube.SendMessage("test", "It is a test.....")
	return nil
}

// func SignalHandler(signal *bridge.Unit) error {
// 	key := signal.Key
// 	var err error
// 	switch key {
// 	case "msg":
// 		logs.Info("%v", signal.Content)
// 		if err = tube.SendMessage("msg", fmt.Sprintf("<h1>%s</h1>", signal.Content)); err != nil {
// 			logs.Error(err)
// 			return err
// 		}

// 	case "start": //start huntting images
// 		if digger.IsRunning {
// 			err := errors.New("Digger is running")
// 			logs.Error(err)
// 			return err
// 		}
// 		logs.Info("%v", signal.Content)
// 		sizeLimit, numberLimit, threadLimit, minmun, maxmun, loggestWait, interval := 0, 0, 0, 0, 0, 0, 0
// 		startPoint, endPoint := 0, 0
// 		method, savePath, baseUrl, lineKey, targetKey := "", "", "", "", ""
// 		sucNum, err := fmt.Sscanf(signal.Content, "%s %s %d %d %d %d %d %d %d %s %s %s %d %d",
// 			&method, &savePath, &sizeLimit, &numberLimit, &threadLimit, &minmun, &maxmun, &loggestWait, &interval,
// 			&baseUrl, &lineKey, &targetKey, &startPoint, &endPoint)
// 		if err != nil {
// 			logs.Error(err)
// 			return err
// 		}
// 		if sucNum != 14 {
// 			err = errors.New("Success scanf numbers not right")
// 			logs.Error(err)
// 			return err
// 		}
// 		//restore blank character
// 		method = strings.ReplaceAll(method, "&npsp", " ")
// 		savePath = strings.ReplaceAll(savePath, "&npsp", " ")
// 		baseUrl = strings.ReplaceAll(baseUrl, "&npsp", " ")
// 		lineKey = strings.ReplaceAll(lineKey, "&npsp", " ")
// 		targetKey = strings.ReplaceAll(targetKey, "&npsp", " ")

// 		//check the savepath if exist
// 		if !checkDirExist(savePath) {
// 			err = fmt.Errorf("directory %s not exist", savePath)
// 			logs.Warn(err)
// 			return err
// 		}
// 		//setting up digger
// 		digger.SizeLimit = sizeLimit
// 		digger.NumberLimit = numberLimit
// 		digger.ThreadLimit = threadLimit
// 		digger.Minmun = minmun
// 		digger.Maxmun = maxmun
// 		digger.LongestWait = loggestWait
// 		digger.Interval = interval
// 		digger.SavePath = savePath
// 		if err = digger.CheckBaseConf(); err != nil {
// 			logs.Error(err)
// 			return err
// 		}
// 		//switch to different model according to method
// 		msg := make(chan string, 100)
// 		switch method {
// 		case "bfs":
// 			go digger.BFS_hunt(baseUrl, lineKey, targetKey, msg)
// 		case "dfs":
// 			go digger.DFS_hunt(baseUrl, lineKey, targetKey, msg)
// 		case "forloop":
// 			go digger.ForLoop_hunt(baseUrl, startPoint, startPoint, msg)
// 		case "urllist":
// 			go digger.UrlList_hunt(baseUrl, msg)
// 		default:
// 			err = fmt.Errorf("Unexpect method name: %s", method)
// 			logs.Error(err)
// 			defer close(msg)
// 			return err
// 		}
// 		//listen and send table data to qt mainwindows
// 		go func() {
// 			for {
// 				returnData, more := <-msg
// 				if !more {
// 					tube.SendMessage("info", "Images hunter is stop !")
// 					break
// 				}
// 				if err = tube.SendMessage("table", returnData); err != nil {
// 					logs.Error(err)
// 				}
// 			}
// 		}()
// 		//listen and send static data to qt mainwindows
// 		go func() {
// 			for {
// 				<-time.Tick(time.Second)
// 				staticData := digger.GetStatic()
// 				if staticData == "pause" {
// 					continue
// 				}
// 				if staticData == "end" {
// 					return
// 				}
// 				tube.SendMessage("static", staticData)
// 			}
// 		}()

// 	case "pause":
// 		digger.Pause()

// 	case "stop":
// 		digger.Stop()

// 	default:
// 		err = fmt.Errorf("Unexpect key name: %s", key)
// 		logs.Error(err)
// 		return err
// 	}
// 	return nil
// }

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
