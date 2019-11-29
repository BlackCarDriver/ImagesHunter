package digger

import (
	"container/list"
	"math/rand"
	"net/http"
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
	savePath    string //the directory to save the download images
)

//some static message
var (
	totalBytes  int //the total size of images already download
	totalNumber int //unmbers of images already download
	pageNumber  int //how many page already visit
)

//some galbol container
var (
	mylist  *list.List
	urlList []string
	urlMap  map[string]bool //the url that already visit
	imgMap  map[string]bool //images that already download
)

//some galbol obeject
var (
	randMachine *rand.Rand
	mainClient  *http.Client
)

//some public regexp obeject
var ()

//check the base config
func CheckBaseConf() error {
	return nil
}

//hunt images in BFS model
func BFS_hunt(baseUrl, lineKey, targetKey string, msg chan<- string) {
	msg <- "end"
	close(msg)
}

//hunt images in DFS model
func DFS_hunt(baseURL, lineKey, targetKey string, msg chan<- string) {
	msg <- "end"
	close(msg)
}

//hunt images in forloop model
func ForLoop_hunt(baseUrl string, start int, stop int, msg chan<- string) {
	msg <- "end"
	close(msg)
}

//hunt images in urlList model
func UrlList_hunt(urlList string, msg chan<- string) {
	msg <- "end"
	close(msg)
}
