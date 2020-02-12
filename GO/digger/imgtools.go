package digger

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/astaxie/beego/logs"
)

//find img url from html code and download some of then according to the config
func digAndSaveImgs(url string) {
	//get all img link from html code
	html, err := digHtml(url)
	if err != nil {
		logs.Warn(err)
		return
	}
	reg1, _ := regexp.Compile(`<img [^>]*>`)
	imgTags := reg1.FindAllString(html, -1)
	imgSlice := make([]string, 0)
	for _, j := range imgTags {
		imgSlice = append(imgSlice, getImgUrls(j, url)...)
	}
	if len(imgSlice) == 0 {
		return
	}
	logs.Info("url     [  %s  ]\n", url)
	logs.Info("<img>   [  %-6d  ]\n", len(imgSlice))
	//create some goroutine and distribute the workes
	urlChan := make(chan string, 100)
	resChan := make(chan int, 20)
	for i := 0; i < threadLimit; i++ {
		imgDownLoader(i, urlChan, resChan)
	}
	for _, j := range imgSlice { //begin to download images
		urlChan <- j
	}
	//wait for images download complete
	close(urlChan)
	close(resChan)
}

//used to distribute download_workes for mutil goroutine
//called by digAndSaveImgs()
func imgDownLoader(no int, urlChan <-chan string, resChan chan<- int) {
	for url := range urlChan {
		//这里加上是否继续的判断条件 🐢
		resChan <- downLoadImages(url)
	}
}

//=============================== the following is tools functions ================================

//download an image specied by url
func downLoadImages(imgUrl string) int {
	if !isImgUrl(imgUrl) {
		return 1
	}
	resp, err := http.Get(imgUrl)
	if err != nil {
		return 2
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 3
	}
	imgSize := len(body)
	if imgSize == 0 {
		return 4
	}
	if imgSize < minSizeLimit*1024 {
		return 5
	}
	if imgSize > maxSizeLimit*1048576 {
		return 6
	}
	imgName := getName(imgUrl)
	updateTotalSize(imgSize)
	imgPath := fmt.Sprint(savePath, string(os.PathSeparator), imgName)
	out, err := os.Create(imgPath)
	defer out.Close()
	if err != nil {
		logs.Error("%s  ----> error: %v \n", imgPath, err)
		return 7
	}
	_, err = io.Copy(out, bytes.NewReader(body))
	if err != nil {
		return 8
	}
	return 0
}

//get a file name for download images
func getName(name string) string {
	suffix := name[strings.LastIndex(name, "."):]
	getNameMutex.Lock()
	newName := strconv.Itoa(totalNumber) + suffix
	totalNumber++
	getNameMutex.Unlock()
	return newName
}

//record the size of download images
func updateTotalSize(addBytes int) {
	updataSizeMutex.Lock()
	totalBytes += addBytes
	updataSizeMutex.Unlock()
}

//judge if a url is a link to an images
func isImgUrl(imgUrl string) bool {
	reg, _ := regexp.Compile(`[^"]*.(jpg|png|jpeg|gif|ico)$`)
	imgUrl = strings.ToLower(imgUrl)
	return reg.MatchString(imgUrl)
}
