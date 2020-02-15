package digger

import (
	"bytes"
	"container/list"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/astaxie/beego/logs"
)

//等待图片队列的图片链接到达，自动下载图片到本地, number:同时下载图片的协程数量
//本函数会导致堵塞，需要运行在单独的协程中直接程序结束
func WaitImgAndDownload(numbers int) error {
	if downloadState != 0 {
		return errors.New("can't setup downloader becase downloadState != 0")
	}
	if numbers < 1 || numbers > 20 {
		return fmt.Errorf("numbers illegal: numbers=%d", numbers)
	}
	downloadState = 1
	defer func() {
		downloadState = 0
	}()
	workersNum := numbers //空闲的下载协程
	var workersMtx sync.Mutex
	var lastDownLoad *list.Element
	lastDownLoad = nil
	//以一秒为间隔，监听图片队列的变化
	ticker := time.NewTicker(1 * time.Second)
	for _ = range ticker.C {
		//外部可以通过downloadState来结束这个协程
		if downloadState == 0 {
			logs.Info("downloader shut down because state=0")
			break
		}
		//队列为空发生在开始工作前，仅第一次有效
		if foundImgList.Len() == 0 {
			continue
		}
		//队列又空边成非空后，获取的第一个元素才有意义。仅第一次有效
		if lastImgEle == nil && lastDownLoad == nil {
			lastImgEle = foundImgList.Front()
		}
		//处理上次由于遇到队列尾部而终止，但是之后队列新增元素的情况
		if lastImgEle == nil && lastDownLoad.Next() != nil {
			lastImgEle = lastDownLoad
		}
		//从图片队列中取尽可能多的链接出来进行下载
		for {
			//退出循环条件：下载协程数用尽、队列遇到末尾、用户暂停
			if lastImgEle == nil || workersNum == 0 || diggerState != 1 {
				break
			}
			imgUrl := lastImgEle.Value.(string)
			lastDownLoad = lastImgEle      //非空值
			lastImgEle = lastImgEle.Next() //可能为空值
			if !isImgUrl(imgUrl) {
				logs.Warn("skip a fake imgUrl: imgUrl=%s", imgUrl)
				continue
			}
			go func() {
				workersMtx.Lock()
				workersNum--
				workersMtx.Unlock()
				err := DownLoadImg(imgUrl)
				if err != nil {
					logs.Warn("Download Images fail: url=%s  err=%v", imgUrl, err)
				}
				workersMtx.Lock()
				workersNum++
				workersMtx.Unlock()
			}()
		}
	}
	return nil
}

//检查url是否符合图片链接的格式
func isImgUrl(imgUrl string) bool {
	imgUrl = strings.ToLower(imgUrl)
	return regexpIsImgUrl.MatchString(imgUrl)
}

//根据Url，制定一个将保存的图片的文件名
func getName(imgUrl string) (string, error) {
	if !isImgUrl(imgUrl) {
		return "", fmt.Errorf("imgUrl format not pass: imgUrl=%s", imgUrl)
	}
	lastDotIdx := strings.LastIndex(imgUrl, ".")
	lastSlashIdx := strings.LastIndex(imgUrl, "/")
	if lastSlashIdx < 0 {
		lastSlashIdx = 0
	}
	suffix := imgUrl[lastDotIdx:] //获取扩展名,包括‘.’
	prefix := imgUrl[lastSlashIdx+1 : lastDotIdx]
	getNameMutex.Lock()
	index := totalNumber //获取文件编号
	totalNumber++
	getNameMutex.Unlock()
	newName := fmt.Sprintf("%d_%s%s", index, prefix, suffix)
	return newName, nil
}

//下载图片到配置指定的目录
func DownLoadImg(imgUrl string) error {
	logs.Debug("DownLoadImg(): imgUrl=%s", imgUrl)
	if !isImgUrl(imgUrl) {
		return fmt.Errorf("not a images url: url=%s", imgUrl)
	}
	resp, err := mainClient.Get(imgUrl)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("response status not ok: url=%s  statusCode=%d", imgUrl, resp.StatusCode)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("ioutil ReadAll fail: %v", err)
	}
	//文件大小检验
	imgSize := len(body)
	if imgSize < minSizeLimit*1024 || imgSize > maxSizeLimit*1024 {
		return fmt.Errorf("imgSize over limited: size=%d", imgSize)
	}
	//更新统计数值
	updataSizeMutex.Lock()
	totalBytes += imgSize
	updataSizeMutex.Unlock()
	//保存文件到本地
	imgName, _ := getName(imgUrl)
	imgPath := fmt.Sprint(savePath, string(os.PathSeparator), imgName)
	out, err := os.Create(imgPath)
	defer out.Close()
	if err != nil {
		return fmt.Errorf("Create file fail: err=%v", err)
	}
	_, err = io.Copy(out, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("copy images fail, err=%v", err)
	}
	return nil
}
