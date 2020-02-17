package digger

import (
	"container/list"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/astaxie/beego/logs"
)

//一些公用配置、参数
var (
	totalSizeLimit int    //图片总大小限制,MB
	minSizeLimit   int    //文件大小最小值，KB
	maxSizeLimit   int    //文件大小最大值，KB
	numberLimit    int    //图片下载数量限制
	pageLimit      int    //爬取页面数量限制(测试用)
	threadLimit    int    //下载引擎数量限制
	waitTimeLimit  int    //最长等待时间，单位秒
	intervalTime   int    //等待间隔，秒
	savePath       string //保存路径
	method         string //策略：BFS\DFS\FOR\LIST
	baseUrl        string //BFS\DFS车裂下的网页入口链接，或FOR策略下的模板, list策略下放url列表
	linkKey        string //转跳链接关键字
	targetKey      string //包含目标图片链接的表情关键字
	startPoint     int    //FOR策略下的变量起点(包含)
	endPoint       int    //FOR策略下的变脸终点(包含)
)

//一些统计数值
var (
	totalBytes  int       //已下载图片的总大小 (维护位置：DownLoadImg())
	totalNumber int       //已下载图片的中数量 (维护位置：getName())
	pageNumber  int       //已经访问的页面数量 (维护位置：getHtmlCodeOfUrl())
	tmpBytes    int       //单位时间内下载图片的总大小 (维护位置：getReportString() & DownLoadImg() )
	totalTime   int       //总运行时间，秒
	percentage  int       //任务进度，(百分之多少)
	lastTime    time.Time //上次统计下载速度的时间
	startTime   time.Time //上次点击开始或继续的时间
)

//一些全局数据容器
var (
	foundPageList *list.List      //已经发现的网页链接地址列表
	lastPageEle   *list.Element   //优先待处理的网页地址
	foundImgList  *list.List      //已经发现的图片链接列表
	lastImgEle    *list.Element   //优先待处理的图片链接
	pageUrlMap    map[string]bool //记录已经访问过的页面避免重复爬取
	imgUrlMap     map[string]bool //记录已经下载过的图片避免重复下载
)

//一些全局变量或对象
var (
	diggerState     int32        //工作状态：0未开始或已终止，1运行中，2暂停中
	downloadState   int32        //图片下载功能状态：0未启动，1已启动
	reporterState   int          //已启动的统计数据发送器的数量
	randMachine     *rand.Rand   //用户创建随机数的对象
	mainClient      *http.Client //用于发送http请求的客户端对象
	getNameMutex    *sync.Mutex  //同步锁，生成随机数时用
	sendMsgMutex    *sync.Mutex  //发送消息同步锁
	updataSizeMutex *sync.Mutex  //同步锁，更新已下载图片大小时用
	msgChan         *chan string //用于将消息直接通过bridge来实现发送到管道
)

//一些全局正则表达式对象
var (
	regexpFindAllATag   *regexp.Regexp //找出所有的 <a> 标签
	regexpFindAllImgTag *regexp.Regexp //找出所有的 <img> 标签
	regexpIsImgUrl      *regexp.Regexp //判断是否一个图片链接
)

func init() {
	initStaticValue()
	diggerState = 0 //未开始
	downloadState = 0
	reporterState = 0
	pageLimit = 4000
	msgChan = nil
	randMachine = rand.New(rand.NewSource(time.Now().UnixNano()))
	mainClient = new(http.Client)
	foundPageList = list.New()
	foundImgList = list.New()
	pageUrlMap = make(map[string]bool)
	imgUrlMap = make(map[string]bool)
	if tmpJar, err := cookiejar.New(nil); err != nil {
		logs.Warn("Create a cookieJar fail: err=%v", err)
		tmpJar = nil
	} else {
		mainClient.Jar = tmpJar
	}
	//初始化同步锁
	updataSizeMutex = new(sync.Mutex)
	getNameMutex = new(sync.Mutex)
	sendMsgMutex = new(sync.Mutex)
	//未经测试修改以下正则可能引发panic
	regexpFindAllATag = regexp.MustCompile(`<a [^>]*href=[^>]*>`)
	regexpFindAllImgTag = regexp.MustCompile(`<img [^>]*src=[^>]*>`)
	regexpIsImgUrl = regexp.MustCompile(`https?://[^ "]*.(jpg|png|jpeg|gif|ico)$`)
}

//开始工作，config 为指定工作方式的配置说明
func StartDigger(config string) error {
	var err error
	if err := setUpConfig(config); err != nil { //初始化
		logs.Error("Setupconfig fail: %v", err)
		return err
	}
	if suc, err := canVisitBaseUrl(); !suc { //检查网络状态
		logs.Warn("Can't not visit BaseUrl, err=%v", err)
		return err
	}
	initStaticValue()
	err = ContinueDigger() //开始工作
	if err != nil {
		logs.Error("Start or continue fail: %v", err)
		return err
	}
	return nil
}

//设置用于返回图片下载情况的管道
func SetupMsgChan(newChan *chan string) error {
	if newChan == nil {
		return errors.New("newChan is nil")
	}
	if msgChan != nil {
		return errors.New("setup msgChan fail because msgChan not nil")
	}
	msgChan = newChan
	logs.Info("msgChan have been setup")
	return nil
}

//暂停工作,保留状态
func PauseDigger() error {
	diggerState = 2   //停止继续发掘网页
	reporterState = 0 //停止发送报告
	downloadState = 2 //暂停继续下载图片
	return nil
}

//终止工作清楚状态 🐢
func StopDigger() error {
	diggerState = 0
	reporterState = 0    //结束发送报告任务
	downloadState = 0    //结束图片下载任务
	var totalSize string //用于表示下载文件总大小的字符串
	if totalBytes < 1<<10 {
		totalSize = fmt.Sprintf("%dB", totalBytes)
	} else if totalBytes < 1<<20 {
		totalSize = fmt.Sprintf("%dKB", totalBytes>>10)
	} else {
		totalSize = fmt.Sprintf("%dMB", totalBytes>>20)
	}
	logs.Info("StopDigger() have been called, data have been clear")
	logs.Info("Achievement: PageListLen=%d \t imagesListLen=%d \t PageNumber=%d \t imagesNumber=%d \t totalSize=%s",
		foundPageList.Len(),
		foundImgList.Len(),
		pageNumber,
		totalNumber,
		totalSize,
	)
	//清除数据
	foundPageList.Init()
	foundImgList.Init()
	imgUrlMap = make(map[string]bool)
	pageUrlMap = make(map[string]bool)
	initStaticValue()
	return nil
}

//开始或继续工作
func ContinueDigger() error {
	var err error
	switch diggerState {
	case 1: //正在工作中
		err = errors.New("Can not start or continue because digger is running")
	case 0, 2:
		if method == "BFS" {
			err = runBFS()
		} else if method == "DFS" {
			err = runDFS()
		} else if method == "FOR" {
			err = runFOR()
		} else if method == "LIST" {
			err = runLIST()
		} else {
			err = errors.New("Unexpect method: mthod=" + method)
		}
		//启动或恢复图片下载以及发送统计数据的功能
		if err == nil {
			go setupReporter()
			go WaitImgAndDownload(threadLimit)
		}
	default:
		err = fmt.Errorf("Unknow diggerState: diggerState=%d", diggerState)
	}
	if err != nil {
		logs.Error(err)
		return err
	}
	return nil
}

//============================= 功能测试 =============================
func TEST1() {
	for i := 0; i < 30; i++ {
		err := sendResult("http:itisimgurl.com/testting", "OK", "test.jpg", i*10240)
		if err != nil {
			logs.Error(err)
		}
		time.Sleep(time.Millisecond * 100)
	}
	time.Sleep(time.Second)
	os.Exit(1)
}

//============================= 执行策略 =====================

//开始或继续BFS策略工作
func runBFS() error {
	var err error
	if diggerState != 0 {
		return errors.New("diggerState not 0")
	}
	if foundPageList.Len() < 1 {
		return errors.New("foundPageList is empty")
	}
	//开始BFS
	diggerState = 1
	go func() {
		for diggerState == 1 {
			//到达停止条件，注意不仅要停止digger，而且要停止reporter 和 downloader
			if isShouldSopt() {
				logs.Info("The exit condition is triggered")
				sendMessage("function", "auto_stop") //🐉
				StopDigger()
				break
			}
			pageHtml, url := "", "" //pageHtml暂存页面的html代码
			var imgLink []string    //暂存图片链接
			var pagelink []string   //暂存转跳链接
			//取出页面队列的第一个元素
			if url = lastPageEle.Value.(string); url == "" {
				logs.Error("lastPageEle is empty string")
				goto end
			}
			//发送http请求获取页面代码
			if err = getHtmlCodeOfUrl(url, &pageHtml); err != nil {
				logs.Warn("getHtmlCodeOfUrl fail: url=%s  err=%v", url, err)
				goto end
			}
			//直接过滤掉长度太短的页面代码
			if len(pageHtml) < 1000 {
				logs.Info("pageHtml not used because too short. length=%d", len(pageHtml))
				goto end
			}
			//获取所有符合条件的图片链接，并加入到图片队列
			if err = getAllSpeciicImgLink(url, &pageHtml, &imgLink); err != nil {
				logs.Warn("Get specific images link from pageHtml fail: err=%v", err)
			} else if len(imgLink) == 0 {
				logs.Warn("No images link found in pageHtml, url=%v", url)
			} else {
				for _, value := range imgLink {
					if imgUrlMap[value] == false {
						foundImgList.PushBack(value)
						imgUrlMap[value] = true
						// logs.Debug("new img: %s", value)
					}
				}
			}
			//获取符合转跳条件的链接
			if err = getAllSpeciicPageLink(url, &pageHtml, &pagelink); err != nil {
				logs.Error("Get specific pagelink from pageHtml fail: err=%v", err)
			} else if len(pagelink) == 0 {
				logs.Warn("No pagelink found in pageHtml, url=%v", url)
			} else {
				for _, value := range pagelink {
					if pageUrlMap[value] == false {
						foundPageList.PushBack(value)
						pageUrlMap[value] = true
						// logs.Debug("new page: %s", value)
					}
				}
			}
			logs.Info("a page is ok: url:%s	pageLen:%d	imgNum:%d	linkNum:%d", url, len(pageHtml), len(imgLink), len(pagelink))
		end:
			//若页面队列已经到底，则BFS结束
			if lastPageEle == foundPageList.Back() {
				logs.Info("last element of foundPageList have been used...")
				break
			} else {
				lastPageEle = lastPageEle.Next()
			}
			time.Sleep(time.Second * time.Duration(intervalTime))
		}
		diggerState = 0
		logs.Info("gorounting in runBFS() go to the end, imgList's length=%d   pageList's length=%d", foundImgList.Len(), foundPageList.Len())
	}()
	return nil
}

//开始或继续DFS策略工作  🐢
func runDFS() error {
	logs.Warn("TODO: runDFS()")
	return nil
}

//开始或继续FOR策略工作  🐢
func runFOR() error {
	logs.Warn("TODO: runFOR()")
	return nil
}

//开始或继续LIST策略工作 🐢
func runLIST() error {
	logs.Warn("TODO: runLIST()")
	return nil
}

//======================= 次要流程 ==============

//处理指定工作方式的配置字符串，将其中的信息解析到全局变量之中
//仅在第一开始、结束后重新开始时调用，暂停后继续不调用
func setUpConfig(config string) error {
	sucNum, err := fmt.Sscanf(config, "%s %s %d %d %d %d %d %d %d %s %s %s %d %d",
		&method,
		&savePath,
		&totalSizeLimit,
		&numberLimit,
		&threadLimit,
		&minSizeLimit,
		&maxSizeLimit,
		&waitTimeLimit,
		&intervalTime,
		&baseUrl,
		&linkKey,
		&targetKey,
		&startPoint,
		&endPoint)
	if sucNum != 14 || err != nil {
		logs.Error("Scanf config from given string fail, err=", err)
		return errors.New("syntax worng")
	}
	if checkBaseConf() == false {
		err = errors.New("config checking not pass")
		logs.Error(err)
		return err
	}
	mainClient.Timeout = time.Second * time.Duration(waitTimeLimit)
	if len(pageUrlMap) > 0 {
		pageUrlMap = make(map[string]bool)
	}
	if len(imgUrlMap) > 0 {
		imgUrlMap = make(map[string]bool)
	}
	if foundPageList.Len() > 0 {
		foundPageList.Init()
	}
	if foundImgList.Len() > 0 {
		foundImgList.Init()
	}
	if method == "BFS" || method == "DFS" {
		lastPageEle = foundPageList.PushBack(baseUrl)
	}
	lastImgEle = foundImgList.Front()
	targetKey = strings.Replace(targetKey, "&empty", "", -1)
	targetKey = strings.Replace(targetKey, "&space", " ", -1)
	linkKey = strings.Replace(linkKey, "&empty", "", -1)
	linkKey = strings.Replace(linkKey, "&space", " ", -1)
	return nil
}

//每次保存图片成功后通过管道向qt端发送一条报告
//size 的单位为字节
func sendResult(imgUrl string, result string, saveName string, size int) error {
	if imgUrl == "" || strings.Contains(imgUrl, " ") {
		return errors.New("imgUrl is null or contain space")
	}
	if saveName == "" || strings.Contains(saveName, " ") {
		return errors.New("saveName is null or contain space")
	}
	if result == "" || strings.Contains(result, " ") {
		return errors.New("result is null or contain space")
	}
	if size < 0 {
		return errors.New("size illeagle")
	}
	if msgChan == nil {
		return errors.New("msgChan not set up")
	}
	resultStr := fmt.Sprintf("table@%s %dKB %s %s", imgUrl, (size+1)/1024, result, saveName) //size+1避免0作除数
	sendMsgMutex.Lock()
	logs.Debug("sendResult:   %s", resultStr)
	(*msgChan) <- resultStr
	sendMsgMutex.Unlock()
	return nil
}

//向qt端发送消息，达到控制组件或消息显示等目的
func sendMessage(key, content string) error {
	if key == "" || content == "" {
		return errors.New("key or content is null string")
	}
	if strings.Contains(key, "@") {
		return errors.New("key or content contain '@'")
	}
	resultStr := fmt.Sprintf("%s@%s", key, content)
	sendMsgMutex.Lock()
	logs.Debug("sendMessage:   %s", resultStr)
	(*msgChan) <- resultStr
	sendMsgMutex.Unlock()
	return nil
}

//============== 判断与检查代码 =============

//检验一个url，且将相对地址转换为绝对地址,若有中文经过转码将会被还原 📇
func CheckUrlAndConver(baseUrl string, targetUrl *string) error {
	if baseUrl == "" {
		return errors.New("baseUrl is empty")
	}
	if targetUrl == nil {
		return errors.New("targetUrl is nil")
	}
	target, err := url.Parse(*targetUrl)
	if err != nil {
		return err
	}
	//将经过转码的URL字符串还原
	if converLink, err := url.QueryUnescape(*targetUrl); err != nil {
		logs.Warn("QueryUnescape url %s fail: err=%d", *targetUrl, err)
	} else {
		*targetUrl = converLink
	}
	//去除表示位置的内容
	if idx := strings.Index(*targetUrl, "#"); idx > 0 {
		*targetUrl = (*targetUrl)[0 : idx+1]
	}
	//若本身为绝对路径则无需继续
	if target.IsAbs() {
		return nil
	}
	base, err := url.Parse(baseUrl)
	if err != nil {
		return fmt.Errorf("BaseUrl not right. err=%v", err)
	}
	if !base.IsAbs() {
		return errors.New("BaseUrl is not absolute url")
	}
	*targetUrl = fmt.Sprintf("%s://%s%s", base.Scheme, base.Host, target.Path)
	return nil
}

//检查是否已到达应该结束任务的条件
func isShouldSopt() bool {
	if diggerState == 0 {
		return true
	}
	if totalBytes >= totalSizeLimit<<20 { //下载总大小到达上限
		logs.Info("total size of download files reach the limit")
		return true
	}
	if pageNumber >= pageLimit { //访问页面数量到达上限
		logs.Info("total page reach the limit")
		return true
	}
	if totalNumber >= numberLimit { //下载数量到达上限
		logs.Info("total numbers of download files reach the limit")
		return true
	}
	//todo 等待时间到达上限
	return false
}

//判断 baseurl 格式是否正确可用
func isBaseUrlRight(baseurl string) bool {
	regHttpUrl := regexp.MustCompile(`https?://[^ "]*`)
	return regHttpUrl.MatchString(baseUrl)
}

//检查文件夹是否存在  📂
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

//检查与配置相关的全局变量，返回检查结果, 一些特定的非法值被设置成默认值而非报错
func checkBaseConf() bool {
	if method != "BFS" && method != "DFS" && method != "FOR" && method != "LIST" {
		logs.Warn("config 'method' not right. method=%s", method)
		return false
	}
	if savePath == "" || checkDirExist(savePath) == false {
		logs.Warn("config 'savepath' not right. savePath=%s", savePath)
		return false
	}
	if totalSizeLimit <= 0 || totalSizeLimit > 100<<20 { //100GB
		logs.Warn("config 'totalSizeLimit' not right. totalSizeLimit=%d", totalSizeLimit)
		return false
	}
	if numberLimit <= 0 {
		logs.Warn("config 'numberLimit' erase from %d to 1", numberLimit)
		numberLimit = 1
	} else if numberLimit > 1000000 {
		logs.Warn("config 'numberLimit' erase from %d to 1000000", numberLimit)
		numberLimit = 1000000
	}
	if threadLimit <= 0 {
		logs.Warn("config 'threadLimit' erase from %d to 1", threadLimit)
		threadLimit = 1
	} else if threadLimit > 20 {
		logs.Warn("config 'threadLimit' erase from %d to 20", threadLimit)
		threadLimit = 20
	}
	if minSizeLimit < 0 {
		logs.Warn("config 'minSizeLimit' erase from %d to 0", minSizeLimit)
		minSizeLimit = 20
	}
	if maxSizeLimit < 0 || maxSizeLimit > 10240 { //100MB
		logs.Warn("config 'maxSizeLimit' erase from %d to 10240", maxSizeLimit)
		minSizeLimit = 20
	}
	if waitTimeLimit < 0 || waitTimeLimit > 10000 {
		logs.Warn("config 'waitTimeLimit' erase from %d to 0", waitTimeLimit)
		waitTimeLimit = 0
	}
	if intervalTime < 0 || intervalTime > 10000 {
		logs.Warn("config 'intervalTime' erase from %d to 0", intervalTime)
		intervalTime = 0
	}
	if isBaseUrlRight(baseUrl) == false {
		logs.Warn("config 'baseUrl' not right. baseUrl=%d", baseUrl)
		return false
	}
	if len(linkKey) > 50 {
		logs.Warn("config 'linkKey' is cancel because too long")
		linkKey = ""
	}
	if len(targetKey) > 50 {
		logs.Warn("config 'targetKey' is cancel because too long")
		targetKey = ""
	}
	if method == "FOR" && startPoint > endPoint {
		logs.Warn("config 'startPoint' and 'endPoing' swap value")
		tmp := startPoint
		startPoint = endPoint
		endPoint = tmp
	}
	return true
}

//检查是否能正常访问baseURL指定的第一个网页,可检测网络状态以及BaseUrl
func canVisitBaseUrl() (bool, error) {
	var err error
	if baseUrl == "" {
		err = errors.New("baseUrl is empty")
		logs.Error(err)
		return false, err
	}
	if mainClient == nil {
		err = errors.New("mainClient is null")
		logs.Error(err)
		return false, err
	}
	if resp, err := mainClient.Get(baseUrl); err != nil {
		logs.Warn("test fail: err=%v", err)
		return false, err
	} else if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("test fail: code=%d  status=%s", resp.StatusCode, resp.Status)
		logs.Warn(err)
		return false, err
	}
	return true, nil
}
