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
	totalBytes  int       //已下载图片的总大小
	totalNumber int       //已下载图片的中数量
	pageNumber  int       //已经访问的页面数量
	tmpBytes    int       //单位时间内下载图片的总大小
	totalTime   int       //总运行时间，秒
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
	randMachine     *rand.Rand   //用户创建随机数的对象
	mainClient      *http.Client //用于发送http请求的客户端对象
	getNameMutex    *sync.Mutex  //同步锁，生成随机数时用
	updataSizeMutex *sync.Mutex  //同步锁，更新已下载图片大小时用
)

//一些全局正则表达式对象
var (
	regexpFindAllATag   *regexp.Regexp //找出所有的 <a> 标签
	regexpFindAllImgTag *regexp.Regexp //找出所有的 <img> 标签
)

func init() {
	diggerState = 0 //未开始
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
	//未经测试修改以下正则可能引发panic
	regexpFindAllATag = regexp.MustCompile(`<a [^>]*href=[^>]*>`)
	regexpFindAllImgTag = regexp.MustCompile(`<img [^>]*src=[^>]*>`)
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
	err = ContinueDigger() //开始工作
	if err != nil {
		logs.Error("Start or continue fail: %v", err)
		return err
	}
	return nil
}

//暂停工作,保留状态 🐢
func PauseDigger() error {
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
	default:
		err = fmt.Errorf("Unknow diggerState: diggerState=%d", diggerState)
	}
	if err != nil {
		logs.Error(err)
		return err
	}
	return nil
}

//终止工作清楚状态 🐢
func StopDigger() error {
	return nil
}

//============================= 功能测试 =============================
func TEST1() {
	var link []string
	if err := getAllSpeciicPageLink("https://tb1.bdstatic.com/", &TmphtmlCode, &link); err != nil {
		logs.Error(err)
		return
	}
	fmt.Println(len(link))
	for i := 0; i < len(link); i++ {
		fmt.Println(link[i])
	}
	os.Exit(0)
}

//============================= 私有函数/工具函数 =====================

//开始或继续BFS策略工作
func runBFS() error {
	var err error
	pageHtml, url := "", "" //pageHtml暂存页面的html代码
	if url = lastPageEle.Value.(string); url == "" {
		err = errors.New("lastPageEle is empty string")
		logs.Error(err)
		return err
	}
	lastPageEle = lastPageEle.Next()
	if err = getHtmlCodeOfUrl(url, &pageHtml); err != nil {
		logs.Warn("getHtmlCodeOfUrl fail: url=%s  err=%v", url, err)
		return err
	}
	//获取符合转跳条件的链接
	var pagelink []string
	if err = getAllSpeciicPageLink(url, &pageHtml, &pagelink); err != nil {
		logs.Error("Get specific pagelink from pageHtml fail: err=%v", err)
		return err
	}
	if len(pagelink) == 0 {
		logs.Warn("No pagelink found in pageHtml, url=%v", url)
	}
	//获取所有符合条件的图片链接
	var imgLink []string
	if err = getAllSpeciicImgLink(url, &pageHtml, &imgLink); err != nil {
		logs.Error("Get specific images link from pageHtml fail: err=%v", err)
		return err
	}
	if len(imgLink) == 0 {
		logs.Warn("No images link found in pageHtml, url=%v", url)
	}

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

//获得htmlCode中全部符合配置指定的转跳链接，写到link[]中
//baseUrl为获得页面的URL，用于将得到的相对链接转换成绝对链接
func getAllSpeciicPageLink(baseUrl string, htmlCode *string, link *[]string) error {
	if htmlCode == nil || *htmlCode == "" {
		return errors.New("htmlCode is empty")
	}
	if *link == nil {
		*link = make([]string, 0)
	} else if len(*link) != 0 {
		return errors.New("link[] not empty")
	}
	//先获取包含链接的<a>标签
	allATag := regexpFindAllATag.FindAllString(*htmlCode, -1)
	if len(allATag) == 0 {
		logs.Warn("find zero <a> tag from htmlCode")
		return nil
	}
	//从<a>标签中筛选出链接，若配置有指定关键字则只从包含关键字的标签中取
	regexpFindLink := regexp.MustCompile(`href="[^"]*`)
	for i := 0; i < len(allATag); i++ {
		if linkKey != "" && !strings.Contains(allATag[i], linkKey) {
			continue
		}
		tmpLink := regexpFindLink.FindString(allATag[i])
		if len(tmpLink) < 7 {
			logs.Warn("find a danger url: %s", tmpLink)
			continue
		}
		tmpLink = tmpLink[6:] //去除href="前缀
		//将相对链接转换成绝对链接
		if err := CheckUrlAndConver(baseUrl, &tmpLink); err != nil {
			logs.Warn("check url %s not pass, err=%v", tmpLink, err)
			continue
		}
		*link = append(*link, tmpLink)
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
		if linkKey != "" && !strings.Contains(allATag[i], linkKey) {
			continue
		}
		tmpLink := regexpFindLink.FindString(allATag[i])
		if len(tmpLink) < 7 {
			logs.Warn("find a danger url: %s", tmpLink)
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

//处理指定工作方式的配置字符串，将其中的信息解析到全局变量之中
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
	return nil
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

//获取用于表示当前工作状态报告信息的字符串 🐢
func getReportString() (string, error) {
	var err error
	if diggerState == 0 { //未开始工作或已经终止
		err = errors.New("Can't get report string because Digger is not working")
		return "", err
	}
	duration := time.Since(lastTime)
	lastTime = time.Now()
	speed := duration.Seconds()
	tmpBytes = 0
	percentage := 30
	reportString := fmt.Sprintf("%d %d %d %d %.2fKB/s %s %d", totalNumber, totalBytes, foundPageList.Len(),
		pageNumber, speed, time.Since(startTime), percentage)
	return reportString, nil
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

//决定某个网页链接是否应该进入 🐢
func isShouldDig(url string) bool {
	return false
}

//判断某个标签中是否包含目标图片链接 🐢
func isHaveTargetImg(tag string) bool {
	return false
}

//判断某个标签中是否包含配置指定的转跳链接 🐢
func isHaveSpecifHref(tag string) bool {
	return false
}

//判断baseurl 格式是否正确可用
func isBaseUrlRight(baseurl string) bool {
	return true
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
	//若本身为绝对路径则无需转换
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
