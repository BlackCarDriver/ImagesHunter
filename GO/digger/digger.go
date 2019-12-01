package digger

import (
	"container/list"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/astaxie/beego/logs"
)

//some base config
var (
	SizeLimit   int    //cost how many space in dish at most
	NumberLimit int    //download how many images at most
	ThreadLimit int    //download how many images in sametime at most
	Minmun      int    //the max size of a image
	Maxmun      int    //the smallest size of a image
	LongestWait int    //the longest time waiting the response of a request
	Interval    int    //wait how many second from page to page
	SavePath    string //the directory to save the download images
)

//some static message
var (
	totalBytes  int       //the total size of images already download
	totalNumber int       //unmbers of images already download
	pageNumber  int       //how many page already visit
	tmpBytes    int       //how many bytes of images have download after last static time
	lastTime    time.Time //the time of last static
)

//some galbol container
var (
	mylist  *list.List
	urlList []string
	urlMap  map[string]bool //the url that already visit
	imgMap  map[string]bool //images that already download
)

//some public value
var (
	IsRunning bool      //whether hunter is huntting
	IsPause   bool      //whether hunter is In the pause
	startTime time.Time //the time when start huntting
)

//some galbol obeject
var (
	randMachine *rand.Rand
	mainClient  *http.Client
)

//some public regexp obeject
var ()

func init() {
	mylist = list.New()
	urlMap = make(map[string]bool)
	imgMap = make(map[string]bool)
	IsPause = false
	IsRunning = false
	randMachine = rand.New(rand.NewSource(time.Now().UnixNano()))
}

//check the base config
func CheckBaseConf() error {
	return nil
}

//hunt images in BFS model
func BFS_hunt(baseUrl, lineKey, targetKey string, msg chan<- string) {
	logs.Info("BFS_hunt is running!")
	IsRunning = true

	//return information to mainwindows
	for i := 0; i < 100; i++ {
		msg <- "https://urlofimages/urlofimages/urlofimages/urlofimages/example.png 1234.kb ok name.png"
		time.Sleep(time.Second * 2)
	}

	IsRunning = false
	close(msg)
}

//hunt images in DFS model
func DFS_hunt(baseURL, lineKey, targetKey string, msg chan<- string) {
	IsRunning = true

	IsRunning = false
	close(msg)
}

//hunt images in forloop model
func ForLoop_hunt(baseUrl string, start int, stop int, msg chan<- string) {
	IsRunning = true

	IsRunning = false
	close(msg)
}

//hunt images in urlList model
func UrlList_hunt(urlList string, msg chan<- string) {
	IsRunning = true

	IsRunning = false
	close(msg)
}

//pause images huntting
func Pause() {
	if IsPause {
		return
	}
	logs.Debug("Hunter Pause!")
	IsPause = true
	return
}

//stop images huntting
func Stop() {
	if !IsRunning {
		return
	}
	logs.Debug("Hunter Stop!")
	return
}

//get static data
func GetStatic() string {
	if IsPause {
		return "pause"
	}
	if !IsRunning {
		return "end"
	}
	duration := time.Since(lastTime)
	lastTime = time.Now()
	speed := duration.Seconds()
	tmpBytes = 0
	percentage := 30
	static := fmt.Sprintf("%d %d %d %d %.2fKB/s %s %d", totalNumber, totalBytes, mylist.Len(),
		pageNumber, speed, time.Since(startTime), percentage)
	//logs.Debug(static)
	return static
}
