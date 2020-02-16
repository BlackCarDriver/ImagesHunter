package digger

import (
	"errors"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"

	"github.com/astaxie/beego/logs"
)

//访问某个页面链接，将请求得到的html代码保存到htmlCode中
func getHtmlCodeOfUrl(url string, htmlCode *string) error {
	if url == "" {
		return errors.New("url is empty string")
	}
	if htmlCode == nil {
		return errors.New("htmlCode is nil")
	}
	resp, err := mainClient.Get(url)
	if err != nil {
		logs.Error("Get fail: err=%v", err)
		return err
	}
	if resp.StatusCode != http.StatusOK {
		logs.Error("response status not ok, url=%s  statusCode=", url, resp.Status)
		return errors.New("response status not OK")
	}
	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logs.Error(err)
		return err
	}
	if len(bytes) == 0 {
		return errors.New("read nothing from response body")
	}
	pageNumber++
	*htmlCode = string(bytes)
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
