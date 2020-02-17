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
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/astaxie/beego/logs"
)

//等待图片队列的图片链接到达，自动下载图片到本地, number:同时下载图片的协程数量
//本函数会导致堵塞，需要运行在单独的协程中直接程序结束
//外部可以通过 downloadState=2 暂停下载任务，downloadState=0 结束任务
func WaitImgAndDownload(numbers int) error {
	//若正处于暂停状态，则让暂停状态变回正常运行状态
	if downloadState == 2 {
		downloadState = 1
		return nil
	}
	//若正在运行中，则很可能错误调用
	if downloadState != 0 {
		return errors.New("can't setup downloader becase downloadState != 0")
	}
	if numbers < 1 || numbers > 20 {
		return fmt.Errorf("numbers illegal: numbers=%d", numbers)
	}
	logs.Debug("Images download is running...")
	downloadState = 1
	defer func() {
		logs.Debug("Images download is close...")
		downloadState = 0
	}()
	workersNum := numbers //空闲的下载协程
	var workersMtx sync.Mutex
	var lastDownLoad *list.Element
	lastDownLoad = nil
	//以一秒为间隔，监听图片队列的变化
	ticker := time.NewTicker(1 * time.Second)
	for _ = range ticker.C {
		//外部可以通过downloadState来暂停和结束图片下载的工作
		if downloadState == 2 {
			continue
		}
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
			if lastImgEle == nil || workersNum == 0 || downloadState != 1 {
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
		//若到达结束条件，则结束工作
		if isShouldSopt() {
			logs.Info("The exit condition is triggered")
			sendMessage("function", "digger autoly stop")
			StopDigger()
			break
		}
	}
	return nil
}

//获得htmlCode中全部符合配置指定的图片链接，写到link[]中
//baseUrl为获得页面的URL，用于将得到的相对链接转换成绝对链接
func getAllSpeciicImgLink(baseUrl string, htmlCode *string, link *[]string) error {
	if htmlCode == nil || *htmlCode == "" {
		return errors.New("htmlCode is empty")
	}
	if *link == nil {
		*link = make([]string, 0)
	} else if len(*link) != 0 {
		return errors.New("link[] not empty")
	}
	//先获取包含链接的<img>标签
	allATag := regexpFindAllImgTag.FindAllString(*htmlCode, -1)
	if len(allATag) == 0 {
		logs.Warn("find zero <img> tag from htmlCode")
		return nil
	}
	//从<img>标签中筛选出链接，若配置有指定关键字则只从包含关键字的标签中取
	regexpFindLink := regexp.MustCompile(`src="[^"]*`)
	for i := 0; i < len(allATag); i++ {
		if targetKey != "" && !strings.Contains(allATag[i], targetKey) {
			continue
		}
		tmpLink := regexpFindLink.FindString(allATag[i])
		if len(tmpLink) < 7 {
			logs.Warn("find a danger url: %s", tmpLink)
			continue
		}
		if len(tmpLink) > 400 {
			logs.Warn("find a danger url, length=%d", len(tmpLink))
			continue
		}
		tmpLink = tmpLink[5:] //去除src="前缀
		//将相对链接转换成绝对链接
		if err := CheckUrlAndConver(baseUrl, &tmpLink); err != nil {
			logs.Warn("check url %s not pass, err=%v", tmpLink, err)
			continue
		}
		*link = append(*link, tmpLink)
	}
	return nil
}

//下载图片到配置指定的目录,同时通过管道发出下载报告
func DownLoadImg(imgUrl string) error {
	var err error
	var resp *http.Response
	var body []byte
	var out *os.File
	imgSize := 0
	imgName, imgPath := "", ""
	tag := "" //下载结果
	if !isImgUrl(imgUrl) {
		err = fmt.Errorf("not a images url: url=%s", imgUrl)
		tag = "url worng"
		goto end
	}
	resp, err = mainClient.Get(imgUrl)
	if err != nil {
		tag = "http fail"
		goto end
	}
	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("response status not ok: url=%s  statusCode=%d", imgUrl, resp.StatusCode)
		tag = fmt.Sprintf("Status:%d", resp.StatusCode)
		goto end
	}
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		err = fmt.Errorf("ioutil ReadAll fail: %v", err)
		tag = "read fail"
		goto end
	}
	//文件大小检验
	imgSize = len(body)
	if imgSize < minSizeLimit*1024 || imgSize > maxSizeLimit*1024 {
		err = fmt.Errorf("imgSize over limited: size=%d", imgSize)
		tag = "size excess"
		goto end
	}
	//保存文件到本地
	imgName, _ = getName(imgUrl)
	imgPath = fmt.Sprint(savePath, string(os.PathSeparator), imgName)
	out, err = os.Create(imgPath)
	defer out.Close()
	if err != nil {
		err = fmt.Errorf("Create file fail: err=%v", err)
		tag = "create fail"
		goto end
	}
	_, err = io.Copy(out, bytes.NewReader(body))
	if err != nil {
		err = fmt.Errorf("copy images fail, err=%v", err)
		tag = "copy fail"
		goto end
	}
end:
	if err != nil {
		logs.Warn("Download images fail, url=%s  err=%v", imgUrl, err)
		if err = sendResult(imgUrl, tag, "---", 0); err != nil {
			logs.Warn(err)
		}

	} else {
		if err = sendResult(imgUrl, "OK", imgName, imgSize); err != nil {
			logs.Warn(err)
		}
		//更新统计数值
		updataSizeMutex.Lock()
		totalBytes += imgSize
		tmpBytes += imgSize
		updataSizeMutex.Unlock()
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
