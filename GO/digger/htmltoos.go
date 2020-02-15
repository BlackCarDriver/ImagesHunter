package digger

import (
	"errors"
	"fmt"
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
	*htmlCode = string(bytes)
	return nil
}

//=================================== Old Code =============================

func DigPWithClass(target, class string) (ps []string, err error) {
	var html string
	html, err = digHtml(target)
	if err != nil {
		return ps, err
	}
	aReg, err := regexp.Compile(`>.*<`)
	ps = aReg.FindAllString(html, -1)
	return ps, err
}

//visit an url and get the html code
func digHtml(url string) (html string, err error) {
	resp, err := mainClient.Get(url)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != 200 {
		return "", err
	}
	body, _ := ioutil.ReadAll(resp.Body)
	html = string(body)
	return html, err
}

//find all <a/> in a html code of specifed url
func digAtags(url string) []string {
	html, err := digHtml(url)
	res := make([]string, 0)
	if err != nil {
		fmt.Println(err)
		return res
	}
	//extract  <a/> tag from html code
	aReg, _ := regexp.Compile(`<a [^>]*>`)
	res = aReg.FindAllString(html, -1)
	return res
}

//find and select some url from an html text
//only right syntax url and frist_meet url would be returned
func digLinkUrls(url string) []string {
	//get a tag text from html code
	aTags := digAtags(url)
	newUrls := make([]string, 0)
	if len(aTags) == 0 {
		return newUrls
	}
	//extract some right and first_times_used url from <a/>
	for _, a := range aTags {
		aurl := getHref(a, url)
		if !isHaveSpecifHref(aurl) { //synax not right or already check before or out of basehref
			//errLog.Println(aurl)
			continue
		}
		newUrls = append(newUrls, aurl)
	}
	return newUrls
}

//find all image url from an <img/>
func getImgUrls(imgTag string, basehref string) []string {
	imgReg, _ := regexp.Compile(`="[^ ]*.(jpg|png|jpeg){1}"`)
	urls := imgReg.FindAllString(imgTag, -1)
	if len(urls) == 0 {
		imgTag = strings.Replace(imgTag, `'`, `"`, -1)
		urls = imgReg.FindAllString(imgTag, -1)
	}
	for i := 0; i < len(urls); i++ {
		urls[i] = urls[i][2 : len(urls[i])-1]
		if strings.HasPrefix(urls[i], `//`) {
			urls[i] = "http:" + urls[i]
		} else if strings.HasPrefix(urls[i], `/`) {
			urls[i] = basehref + urls[i]
		}
	}
	return urls
}

//extract and correct the url from a tag
//called by digurl()
func getHref(aTag string, basehref string) string {
	hrefReg, _ := regexp.Compile(`href="[^"]*`)
	url := hrefReg.FindString(aTag)
	if len(url) < 7 {
		return ""
	}
	url = url[6:] //erase 'href="'
	if len(url) < 2 {
		return ""
	}
	if strings.HasPrefix(url, "http") {
		return strings.TrimRight(url, `/`)
	}
	if strings.HasPrefix(url, `//`) { // "//aa" -> http：//aa
		url = `http:` + url
	} else { //such as "/aa" or "aa"  such append to basehref
		tindex := strings.Index(basehref, "?") // www.baidu.com/asdfad?index=...?
		if tindex > 0 {
			basehref = basehref[:tindex] // www.baidu.com/aadsfd
		}
		tindex = strings.LastIndex(basehref, `/`)
		if tindex > 0 {
			basehref = basehref[:tindex] // www.baudu.com/
			basehref = strings.TrimRight(basehref, `/`)
		}
		if url[0] != '/' {
			url = "/" + url
		}
		url = basehref + url
	}
	url = strings.TrimRight(url, `/`)
	return url
}

//to analyze which url can be visited
func analyze(url string) {
	resp, err := mainClient.Get(url)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("%s ----------> %v \n", url, resp.Status)
}

//=================== tools function place below =================================

//get a string to identified urls to same path
//called by wasUsed
func getUrlPath(url string) string {
	tindex := strings.Index(url, ":")
	url = url[tindex+1:]
	url = strings.Trim(url, `/`)
	return url
}

var TmphtmlCode = `<html xmlns="http://www.w3.org/1999/xhtml" id="sixapart-standard">
<head>
    <meta http-equiv="Content-Type" content="text/html; charset=UTF-8" />
<meta name="generator" content="Movable Type  5.2.2" />
<meta name="viewport" content="width=device-width, initial-scale=1" />
<!--link rel="stylesheet" href="http://www.ruanyifeng.com/blog/styles.css" type="text/css" /-->
<link rel="start" href="http://www.ruanyifeng.com/blog/" title="Home" />
<link rel="alternate" type="application/atom+xml" title="Recent Entries" href="http://feeds.feedburner.com/ruanyifeng" />
<script type="text/javascript" src="http://www.ruanyifeng.com/blog/mt.js"></script>
<!--
<rdf:RDF xmlns="http://web.resource.org/cc/"
         xmlns:dc="http://purl.org/dc/elements/1.1/"
         xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
<Work rdf:about="http://www.ruanyifeng.com/blog/2019/11/css-position.html">
<dc:title>CSS 定位详解</dc:title>
<dc:description>CSS 有两个最重要的基本属性，前端开发必须掌握：display 和 position。...</dc:description>
<dc:creator>阮一峰</dc:creator>
<dc:date>2019-11-19T09:23:51+08:00</dc:date>
<license rdf:resource="http://creativecommons.org/licenses/by-nc-nd/3.0/" />
</Work>
<License rdf:about="http://creativecommons.org/licenses/by-nc-nd/3.0/">
</License>
</rdf:RDF>
-->

<style>
body {
  background-color: #f5f5d5;
}

#container::before {
  display: block;
  width: 100%;
  padding: 10px;
  background: rgba(0,0,0,0.1);
  text-align: center;
  content: "本站显示不正常，可能因为您使用了广告拦截器。";
}

</style>
<script>
function loadjscssfile(filename, filetype){
    if (filetype=="js"){ //if filename is a external JavaScript file
        var fileref=document.createElement('script')
        fileref.setAttribute("type","text/javascript")
        fileref.setAttribute("src", filename)
    }
    else if (filetype=="css"){ //if filename is an external CSS file
        var fileref=document.createElement("link")
        fileref.setAttribute("rel", "stylesheet")
        fileref.setAttribute("type", "text/css")
        fileref.setAttribute("href", filename)
    }
    if (typeof fileref!="undefined")
        document.getElementsByTagName("head")[0].appendChild(fileref)
}
//loadjscssfile("http://www.ruanyifeng.com/blog/styles.css", "css");
loadjscssfile('/static/themes/theme_scrapbook/theme_scrapbook.min.css', "css");


function checker() {
// var img = document.querySelector('img[src^="http://www.ruanyifeng.com/blog/images"]');
var img = document.querySelector('a > img[src*="wangbase.com/blogimg/asset/"]');
  if (img && window.getComputedStyle(img).display === 'none'){
    var sponsor = document.querySelector('.entry-sponsor');
    var prompt = document.createElement('div');
    prompt.style = 'border: 1px solid #c6c6c6;border-radius: 4px;background-color: #f5f2f0;padding: 15px; font-size: 14px;';
  prompt.innerHTML = '<p>您使用了广告拦截器，导致本站内容无法显示。</p><p>请将 www.ruanyifeng.com 加入白名单，解除广告屏蔽后，刷新页面。谢谢。</p>';
    sponsor.parentNode.replaceChild(prompt, sponsor);
    document.querySelector('#main-content').innerHTML = '';
  }
}

setTimeout(checker, 1000);
</script>

    <link rel="EditURI" type="application/rsd+xml" title="RSD" href="http://www.ruanyifeng.com/blog/rsd.xml" />
    <title>阮一峰的网络日志</title>
<script type="text/javascript" src="tooltip.js"></script>
<script type="text/javascript">
document.addEventListener("DOMContentLoaded", function(event) {
  enableTooltips("latest-comments");
});

/*
window.addEventListener('load', function(event) {
  setTimeout(function () {
    hab('#homepage_cre');
  }, 1000);
});
*/
</script>

</head>
<body id="scrapbook" class="mt-main-index two-columns">
<script>
if (/mobile/i.test(navigator.userAgent) || /android/i.test(navigator.userAgent)) document.body.classList.add('mobile');
</script>
    <div id="container">
        <div id="container-inner">


            <div id="header">
    <div id="header-inner">
        <div id="header-content">


            <h1 id="header-name">阮一峰的网络日志</h1>

<div id="google_search">
<!-- SiteSearch Google -->
<form action="https://www.baidu.com/s" target="_blank" method="get" id="cse-search-box">
  <div>
  <input type="text" size="20" name="origin" class="searchbox" id="sbi" value="" />
  <input type="hidden" name="wd" value="" />
  <!--input type="image" src="/static/themes/theme_scrapbook/images/top_search_submit.gif" class="searchbox_submit" value="" alt="搜索" name="sa" onclick="this.form.wd.value = this.form.origin.value + ' inurl:www.ruanyifeng.com/blog'" /-->
  <input type="image" src="/static/themes/theme_scrapbook/images/top_search_submit.gif" class="searchbox_submit" value="" alt="搜索" name="sa" onclick="this.form.wd.value = this.form.origin.value + ' 阮一峰'" />
</div>
</form>

<!-- SiteSearch Google -->
</div>
<div id="feed_icon">
<a href="/feed.html" title="订阅Feed">
<img src="data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAACAAAAAgCAYAAABzenr0AAAABGdBTUEAAK/INwWK6QAAABl0RVh0U29mdHdhcmUAQWRvYmUgSW1hZ2VSZWFkeXHJZTwAAAUzSURBVHjavFdbbFRVFF3nPjoz7dTWTittaW0jUDRAUqaNojyqREnEQKgfUj9MqqAmhqRt/OCD4CuY+Kckoh+aiGKC+gMJbdHoRysJ8dkhhmJLNdDKtJU+6GMK87j3Hs85d2Z6HzNtMYWb3Dn3NWftvfba+5xNYDl+e6Fkj6yqb/oDRbWq14vlPBLRKCITkxf0ROLt+hNjp1PPSRK4kA3vF1dXNRcWlyA2OQU9eos9opAkAiKxD+XkKO6t15aRWO7J/MgmAZU8MEgexgZHMX518Dh72sYMmVKShnxWuWHdHtxKIDIYTgMuDzgfmSOIQkYMpdUF8OY92Hytt4/jvkg47czzU16iQovM3QFwmNck+Yyduu7D6NA0Z6JR4THntFs9V4tWQg6Ui3s6MwKDncsFTnXKLJhDSeUK3AgPtyhccDzmVs999buRt/1Vm4i0od+hX7+MRG87jPGB/w1u8FPj9xEw7McVrnYuOCvtpjTth3J/nTg99c8LRhKhr6D3dTB5R24bXFwbMXBsyZzeoXaycEpJ95TB09AGX/NpqLVNtw8urnVzLvHjFNxiFqRy2OOHuqUVnue+ACkoWzo4O6lGzTmuHq6nPvY2m9rVqjrIK2rMEKxqyG5NPAKt+wjo0LklgfNxJkZMA3KJvqRUk3z5UFY3QH14P0h+WUY79HPvgv7VuSg4ZRGY1YgZgqXmORccF17sy2ehnf9AeO085K2HQFbtXBScj0LcpgF2cN+WV+DZ/LJQu6gD4R7oV7pBJwbSgtMvfiPoVp56DySwxm7EtkMs1WdAB7qzggsDJKQYsHucSkOudrkiCPWR/fA2nYCn8SNIK4NptSMyAu3sAdDRkIsJdfth0LzSrODUoPNZ4KI9SxJI5UHk7D4GdQfz2us31c7CoHMjRkKuDPHseCMrONVhNcDJwMJpKFVvg9L4OaTiNWm1x789KCqkrXhVBiEz0WYCT2nAzQAD1/vaETv1GrRfP4Vx5cfMNcDPwvP0h0DhanPym7OIf/+O67vcJ1/PCJ4KgdzaUP6Wz+dU+5yIL6fV+PsHGAOdwlPpvvUOyeeAVGyCdqkDNB6DPjsBSrnndfOGevOh3RhGItxvA+fX1CtbGFhgYUFkFMZPR6F1HnClHq8HyubWtJexX06CRmdt33hrd7nA7SFY4qoGpnYuOKcRykPPgDCBcsHx9Iv+fNL2PueBehCWUfYQIIMGLOCcOmXDXsh1+yCt35tUPfvzGFuSvzvoinXOxqa02qOhM6733nVP2MAdaej2XN11DPKjLZCD+yBvahGCo7JfTKAN9UD7s8Oe9zUNIhz8fWI8DG2k38WCFdxugANcXrvTVd1IEbuv3Jour7Hzn7jLMBNfKs7R3i67gRVrbeCOEDhinmWhAatsqdquM2XzHZINhK2cqTjHr/XZdVJUbgN3MWAVXKbSyg9jesRW2xP9di+lwrL5ojM3m2H/kG9hwcIA37c71W6wJdW2J2S5nrjYbq/t1AHAhJsKQeyfPvf6IMJgghPJhFZ4x0KlfLFvt22du45Au/A1SOlGc0P672XXwhLtOcM0kTTEMMd0qkVmMNXxMd/tsedUjInr4SQDgOfeXMSiN0FCL5WHah4L1qqYXPJOJlttd+a5M+YpcG5poLYKQ5f+6JJ4r8bbJYP47hq4r7QAs9PjYNhHJd4o8l5taiwuOpa7AS4XKqI/5NjJbTnaWK92nLdLuhQAJayRNMiygXPBeQN+Qbvu0zDc3y+aUzhbkGR73sI7ljvUnndx2q3t+X8CDAD66FtrIL864AAAAABJRU5ErkJggg==" alt="" style="border: 0pt none;" />
</a></div>

        </div>
    </div>
</div>



            <div id="content">
                <div id="content-inner">


                    <div id="alpha">
                        <div id="alpha-inner">


                            <div id="entry-2158" class="entry-asset asset hentry">
    <div class="asset-header">
        <h2 class="asset-name entry-title"><a href="http://www.ruanyifeng.com/blog/2020/02/weekly-issue-94.html" rel="bookmark">科技爱好者周刊：第 94 期</a></h2>
<p id="asset-tags">


分类<span class="delimiter">：</span>
<a href="http://www.ruanyifeng.com/blog/weekly/" rel="tag">周刊</a>



</p>
</div>
    <div class="asset-content entry-content">

        <div class="asset-body">
            <p>这里记录每周值得分享的科技内容，周五发布。</p>

        </div>



        <div class="asset-more-link">
            <p><a href="http://www.ruanyifeng.com/blog/2020/02/weekly-issue-94.html" rel="bookmark">继续阅读全文 »</a></p>
        </div>

    </div>
    <div class="asset-footer">
<div class="asset-meta">
           <p> <span class="byline">

           <abbr class="published" title="2020-02-14T10:09:35+08:00">2020年2月14日 10:09</abbr>
            </span>

            <span class="separator">|</span> <a href="http://www.ruanyifeng.com/blog/2020/02/weekly-issue-94.html#comments">留言（12）</a>


        </p></div>
</div>
</div>




<div id="homepage">
<h3>最新文章</h3>
<ul>

<li class="module-list-item"><span>2020年02月 7日 » </span><a href="http://www.ruanyifeng.com/blog/2020/02/weekly-issue-93.html">科技爱好者周刊：第 93 期</a></li>

<li class="module-list-item"><span>2020年01月31日 » </span><a href="http://www.ruanyifeng.com/blog/2020/01/weekly-issue-92.html">科技爱好者周刊：第 92 期</a></li>

<li class="module-list-item"><span>2020年01月26日 » </span><a href="http://www.ruanyifeng.com/blog/2020/01/deno-intro.html">Deno 运行时入门教程：Node.js 的替代品</a></li>

<li class="module-list-item"><span>2020年01月17日 » </span><a href="http://www.ruanyifeng.com/blog/2020/01/weekly-issue-91.html">科技爱好者周刊：第 91 期</a></li>

<li class="module-list-item"><span>2020年01月16日 » </span><a href="http://www.ruanyifeng.com/blog/2020/01/china-technology-review.html">我对中国科技行业的看法（译文）</a></li>

<li class="module-list-item"><span>2020年01月14日 » </span><a href="http://www.ruanyifeng.com/blog/2020/01/ffmpeg.html">FFmpeg 视频处理入门教程</a></li>

<li class="module-list-item"><span>2020年01月10日 » </span><a href="http://www.ruanyifeng.com/blog/2020/01/weekly-issue-90.html">科技爱好者周刊：第 90 期</a></li>

  <li class="module-list-item"><a href="http://www.ruanyifeng.com/blog/archives.html"><strong>更多文章……</strong></a></li>
</ul>



</div>

<div style="clear:both;float:none;text-align:center;margin:2em auto;display: inline-block ! important;width: 100%;
    max-width: 100%;">
</div>




                        </div>
                    </div>


                    <div id="beta">
    <div id="beta-inner">
<div class="module-comments module"  id="latest-comments">
  <h2 class="module-header">最新留言</h2>
  <div class="module-content">
  <ul class="module-list">

    <li class="module-list-item">

      <a href="http://www.ruanyifeng.com/blog/2020/02/weekly-issue-94.html#comment-416227" title="科技爱好者周刊：第 94 期" tooltip="
引用kiddyu的发言：

“一张网页的大小，目前通常是50MB，而不是5KB。”
这个不太理解



因为这句话翻……">laixintao</a>

    </li>

    <li class="module-list-item">

      <a href="http://www.ruanyifeng.com/blog/2020/02/weekly-issue-94.html#comment-416224" title="科技爱好者周刊：第 94 期" tooltip="另外8种人类如果没有消失，现在的世界文明是不是会更加灿烂呢
？……">荒原之梦</a>

    </li>

    <li class="module-list-item">

      <a href="http://www.ruanyifeng.com/blog/2020/02/weekly-issue-94.html#comment-416221" title="科技爱好者周刊：第 94 期" tooltip="“一张网页的大小，目前通常是50MB，而不是5KB。”
这个不太理解……">kiddyu</a>

    </li>

    <li class="module-list-item">

      <a href="http://www.ruanyifeng.com/blog/2018/10/restful-api-best-practices.html#comment-416220" title="RESTful API 最佳实践" tooltip="我也存在和订单类似的问题，比如是用户的启用与禁用，
接口该如何设计呢？是PUT /users/{id}/enbale 还是……">我是一只小小鸟</a>

    </li>

    <li class="module-list-item">

      <a href="http://www.ruanyifeng.com/blog/2020/02/weekly-issue-93.html#comment-416219" title="科技爱好者周刊：第 93 期" tooltip=" 呀，语雀不更新了？……">lisa</a>

    </li>

    <li class="module-list-item">

      <a href="http://www.ruanyifeng.com/blog/2020/02/weekly-issue-94.html#comment-416218" title="科技爱好者周刊：第 94 期" tooltip="这一期质量很高，几个感兴趣打算深度读一下……">Feng</a>

    </li>

    <li class="module-list-item">

      <a href="http://www.ruanyifeng.com/blog/2020/02/weekly-issue-94.html#comment-416217" title="科技爱好者周刊：第 94 期" tooltip="这手表太费电，哈哈 完全背离了手表的作用 但是很COOL……">ix
x</a>

    </li>

    <li class="module-list-item">

      <a href="http://www.ruanyifeng.com/blog/2020/02/weekly-issue-93.html#comment-416216" title="科技爱好者周刊：第 93 期" tooltip="实际上，我无法理解“地球不再宜居就移民火星”这种论调。瘦死
的骆驼比马大，地球得变得多么不宜居才能比火星还要不宜居？移民火……">nikolaus</a>

    </li>

  </ul>

  </div>
</div>


<div class="module-categories module">
<h2 class="module-header">关于</h2>
<div class="module-content">
  <ul class="module-list">
  <p class="module-list-item"><a href="http://www.ruanyifeng.com/blog/images/person_shot.jpg" onclick="window.open('http://www.ruanyifeng.com/blog/images/person2.jpg','popup','width=480,height=640,scrollbars=no,resizable=no,toolbar=no,directories=no,location=no,menubar=no,status=no,left=0,top=0'); return false"><img src="http://www.ruanyifeng.com/blog/images/person2_s.jpg" width="80" height="106" alt="个人照片" /></a>
  </p>
    <!--li class="module-list-item"><a href="/about.html">个人简介</a>，<a href="2015/02/turing-interview.html" target="_blank">访谈</a></li-->
    <!--li class="module-list-item">Email：<br /><a href="mailto:yifeng.ruan@gmail.com">yifeng.ruan@gmail.com</a></li-->
    <li class="module-list-item">文章：<a href="http://www.ruanyifeng.com/blog/archives.html">1864</a></li>
    <li class="module-list-item">留言：55565</li>

  </ul>
</div>
</div>




    </div>
</div>






                </div>
            </div>


            <div id="footer">
<div id="footer-inner">
  <div id="footer-content">


<!--
      <section>
     </section>
  <p><a title="Instagram" target="_blank" href="http://instagram.com/ruanyf">Instagram</a></p>
       <p><a title="订阅" href="https://app.feedblitz.com/f/f.fbz?Sub=348868" target="_blank">邮件订阅</a></p>

       <h2>Feed</h2>
       <p><a title="FeedBurner" href="http://feeds.feedburner.com/ruanyifeng" target="_blank">FeedBurner</a></p>
       <p><a title="atom.xml" href="http://www.ruanyifeng.com/blog/atom.xml" target="_blank">atom.xml</a></p>

      <p>支付宝：<a title="支付宝" href="alipays://platformapi/startapp?appId=20000067&url=http%3A%2F%2Fwww.ruanyifeng.com%2Fblog" target="_blank">yifeng.ruan@gmail.com</a></p>
-->

     <section>
<!--
       <h2>授权方式</h2>
       <p><a title="许可证" href="http://creativecommons.org/licenses/by-nc-nd/3.0/deed.zh" target="_blank">自由转载-非商用-非衍生-保持署名</a></p>
       <h2>微信公号</h2>
       <h2>社交帐号</h2>
       <h2>联系方式</h2>
     -->
       <p><img src="https://www.wangbase.com/blogimg/asset/202001/bg2020013101.jpg" alt="微信扫描"></p>
       <p>
         <a title="微博" href="http://weibo.com/ruanyf" target="_blank">Weibo</a> |
         <a title="Twitter" target="_blank" href="https://twitter.com/ruanyf">Twitter</a> |
         <a title="GitHub" target="_blank" href="https://github.com/ruanyf">GitHub</a>
       </p>

     <p>Email: <a title="电子邮件" href="mailto:yifeng.ruan@gmail.com" target="_blank">yifeng.ruan@gmail.com</a></p>
     </section>




  </div>
</div>
</div>

<script>
  (function(i,s,o,g,r,a,m){i['GoogleAnalyticsObject']=r;i[r]=i[r]||function(){
  (i[r].q=i[r].q||[]).push(arguments)},i[r].l=1*new Date();a=s.createElement(o),
  m=s.getElementsByTagName(o)[0];a.async=1;a.src=g;m.parentNode.insertBefore(a,m)
  })(window,document,'script','//www.google-analytics.com/analytics.js','ga');

  ga('create', 'UA-46829782-1', 'ruanyifeng.com');
  ga('send', 'pageview');

</script>

<script type="text/javascript" src="/blog/stats.js"></script>
<script>
var supportImg = document.querySelector('#support-img');

if (supportImg && _hmt) {
  _hmt.push(['_trackEvent', 'banner', 'load']);
  supportImg.addEventListener('click', function () {
    _hmt.push(['_trackEvent', 'banner', 'click']);
  }, false);
}

var homepageImg = document.querySelector('#homepage_sponsor');
if (homepageImg && _hmt) {
  _hmt.push(['_trackEvent', 'homepage-banner', 'load']);
  homepageImg.addEventListener('click', function () {
    _hmt.push(['_trackEvent', 'homepage-banner', 'click']);
  }, false);
}
</script>



        </div>

    </div>


</body>
</html>
`
