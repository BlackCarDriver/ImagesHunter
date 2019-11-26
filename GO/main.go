package main

import (
	"fmt"
	"os"
	"time"

	"./bridge"
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
	if runResult.Key == "[fail]" {
		logs.Error("Start fail: %s", runResult.Content)
		os.Exit(1)
	} else {
		logs.Info(runResult.Key)
	}
	time.Sleep(3 * time.Second)
	for i := 0; i < 10; i++ {
		if err := tube.SendMessage("test", "https://blackcardriver.cn/blackcardriver/blackcardriver/blackcardriver/blackcardriver/blackcardriver 100kb ok! 101.png"); err != nil {
			logs.Error(err)
		}

	}

	/*
		for {
			newMsg, more := <-msgChan
			if more {
				SignalHandler(newMsg)
			} else {
				logs.Info("Go socket client close!")
				break
			}
		}
	*/
}

func SignalHandler(signal *bridge.Unit) error {
	key := signal.Key
	switch key {
	case "msg":
		logs.Info("%v", signal.Content)
		if err := tube.SendMessage("msg", fmt.Sprintf("<h1>%s</h1>", signal.Content)); err != nil {
			logs.Error(err)
			return err
		}
	default:
		logs.Info("%v", signal)
	}
	return nil
}
