package digger

import (
	"errors"
	"fmt"
	"time"

	"github.com/astaxie/beego/logs"
)

/*
	reporter 负责向qt端发送工作状况
	通过 go setupReporter() 来启动这个功能，
	通过让reporterState=0来停止reporter
*/

//堵塞，定期向qt端发送统计数据报告，
//外部将reporterState设成非1的值可以结束循环
func setupReporter() {
	if reporterState > 0 {
		logs.Warn("another reporter sill running")
		return
	}
	logs.Info("reporter is running...")
	reporterState = 1
	defer func() {
		logs.Info("reporter exit...")
		reporterState = 0
	}()
	tigger := time.Tick(2 * time.Second)
	for _ = range tigger {
		if reporterState != 1 {
			break
		}
		reportStr, err := getReportString()
		if err != nil {
			logs.Error("getReportString fail: err=%v", err)
			break
		}
		if err := sendMessage("static", reportStr); err != nil {
			logs.Warn("send report fail: err=%v", err)
		}
	}
	return
}

//获取用于表示当前工作状态报告信息的字符串
//调用后部分统计数值将会被更新
func getReportString() (string, error) {
	if reporterState == 0 { //未开始工作或已经终止
		return "", errors.New("Can't get report string because reposter is not working")
	}
	if diggerState == 2 {
		return "", errors.New("Should'n send report because digger is pause")
	}
	duration := time.Since(lastTime)
	lastTime = time.Now()
	durationSecond := 1
	if int(duration.Seconds()) > 1 { //避免出现0
		durationSecond = int(duration.Seconds())
	}
	totalTime += durationSecond
	//表示当前速度
	speed := (tmpBytes / 1024) / durationSecond //多少KB/s
	speedStr := ""
	if speed < 1024 {
		speedStr = fmt.Sprintf("%dKb/s", speed)
	} else {
		speedStr = fmt.Sprintf("%dMb/s", speed/1024)
	}
	//表示使用时间
	totalTimeStr := ""
	if totalTime < 60 {
		totalTimeStr = fmt.Sprintf("%ds", totalTime)
	} else if totalTime < 3600 {
		totalTimeStr = fmt.Sprintf("%dm", totalTime/60)
	} else {
		totalTimeStr = fmt.Sprintf("%dh", totalTime/3600)
	}
	tmpBytes = 0
	//计算任务进度
	percentage = maxInt(
		totalBytes+1/(totalSizeLimit+1)*1024,
		totalNumber+1/numberLimit+1,
		pageNumber+1/pageLimit,
	)
	//计算已用空间
	var totalSize string //用于表示下载文件总大小的字符串
	if totalBytes < 1<<10 {
		totalSize = fmt.Sprintf("%dB", totalBytes)
	} else if totalBytes < 1<<20 {
		totalSize = fmt.Sprintf("%dKB", totalBytes>>10)
	} else {
		totalSize = fmt.Sprintf("%dMB", totalBytes>>20)
	}
	//按照协议格式生成报告字符串
	reportString := fmt.Sprintf("%d %s %d %d %s %s %d",
		totalNumber,
		totalSize,
		foundPageList.Len(),
		pageNumber,
		speedStr,
		totalTimeStr,
		percentage,
	)
	return reportString, nil
}

//初始化或重置一些统计数值
func initStaticValue() {
	totalBytes = 0
	totalNumber = 0
	pageNumber = 0
	tmpBytes = 0
	totalTime = 0
	lastTime = time.Now()
	startTime = time.Now()
}

//从多个整数中获取出最小的值,返回值在0～100之间
func maxInt(arg ...int) int {
	if len(arg) == 0 {
		return 0
	}
	var maxmun = 0
	for _, tmp := range arg {
		if maxmun < tmp {
			maxmun = tmp
		}
	}
	if maxmun > 100 {
		return 100
	}
	return maxmun
}
