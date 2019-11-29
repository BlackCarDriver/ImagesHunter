package main

import (
	"errors"
	"fmt"
	"os"

	"./bridge"
	"./digger"
	"github.com/astaxie/beego/logs"
)

var tube *bridge.Bridge

func main() {
	logs.EnableFuncCallDepth(true)
	logs.SetLogFuncCallDepth(3)
	var err error
	tube, err = bridge.GetBridge(1024*100, 4747)
	if err != nil {
		logs.Error(err)
		os.Exit(1)
	}
	msgChan := make(chan *bridge.Unit, 100)
	go tube.Start(msgChan)
	runResult := <-msgChan
	//the first message is the result of socket connectting
	if runResult.Key == "[fail]" {
		logs.Error("Start fail: %s", runResult.Content)
		os.Exit(1)
	} else {
		logs.Info(runResult.Key)
	}
	//wait and handle the message
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
	digger.DigPWithClass()
}

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
		//check the savepath if exist
		if !checkDirExist(savePath) {
			err = fmt.Errorf("directory %s not exist", savePath)
			logs.Error(err)
			return err
		}
		//setting up digger
		digger.SizeLimit = sizeLimit
		digger.NumberList = numberLimit
		digger.ThreadLimit = threadLimit
		digger.Minmun = minmun
		digger.Maxmun = maxmun
		digger.LongestWait = loggestWait
		digger.Interval = interval
		digger.Savepath = savePath
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
		for {
			returnData, more := <-msg
			if !more {
				break
			}

		}

		return nil

	default:
		err = fmt.Errorf("Unexpect key name: %s", key)
		logs.Error(err)
		return err
	}
	return nil
}

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
