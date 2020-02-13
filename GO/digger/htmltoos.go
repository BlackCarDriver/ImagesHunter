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

var TmphtmlCode = `
<!DOCTYPE html><!--STATUS OK--><html><head><meta name="keywords" content="璐村惂,鐧惧害璐村惂,璁哄潧,鍏磋叮,绀惧尯,BBS"/><meta name="description" content="鐧惧害璐村惂鈥斺€斿叏鐞冩渶澶х殑涓枃绀惧尯銆傝创鍚х殑浣垮懡鏄蹇楀悓閬撳悎鐨勪汉鐩歌仛銆備笉璁烘槸澶т紬璇濋杩樻槸灏忎紬璇濋锛岄兘鑳界簿鍑嗗湴鑱氶泦澶ф壒鍚屽ソ缃戝弸锛屽睍绀鸿嚜鎴戦閲囷紝缁撲氦鐭ラ煶锛屾惌寤哄埆鍏风壒鑹茬殑鈥滃叴瓒ｄ富棰樷€滀簰鍔ㄥ钩鍙般€傝创鍚х洰褰曟兜鐩栨父鎴忋€佸湴鍖恒€佹枃瀛︺€佸姩婕€佸ū涔愭槑鏄熴€佺敓娲汇€佷綋鑲层€佺數鑴戞暟鐮佺瓑鏂规柟闈㈤潰锛屾槸鍏ㄧ悆鏈€澶х殑涓枃浜ゆ祦骞冲彴锛屽畠涓轰汉浠彁渚涗竴涓〃杈惧拰浜ゆ祦鎬濇兂鐨勮嚜鐢辩綉缁滅┖闂达紝骞朵互姝ゆ眹闆嗗織鍚岄亾鍚堢殑缃戝弸銆? /><meta charset="UTF-8"><meta name="baidu-site-verification" content="BeXfvLldS5" /><meta http-equiv="X-UA-Compatible" content="IE=edge,chrome=1"><meta name="baidu-site-verification" content="jpBCrwX689" /><link rel="search" type="application/opensearchdescription+xml" href="/tb/cms/content-search.xml" title="鐧惧害璐村惂" /><title>鐧惧害璐村惂鈥斺€斿叏鐞冩渶澶х殑涓枃绀惧尯</title><script type="text/javascript">void function(t,e,n,a,o,i,r){t.alogObjectName=o,t[o]=t[o]||function(){(t[o].q=t[o].q||[]).push(arguments)},t[o].l=t[o].l||+new Date,a="https:"===t.location.protocol?"https://fex.bdstatic.com"+a:"http://fex.bdstatic.com"+a;var c=!0;if(t.alogObjectConfig&&t.alogObjectConfig.sample){var s=Math.random();t.alogObjectConfig.rand=s,s>t.alogObjectConfig.sample&&(c=!1)}c&&(i=e.createElement(n),i.async=!0,i.src=a+"?v="+~(new Date/864e5)+~(new Date/864e5),r=e.getElementsByTagName(n)[0],r.parentNode.insertBefore(i,r))}(window,document,"script","/hunter/alog/alog.min.js","alog"),void function(){function t(){}window.PDC={mark:function(t,e){alog("speed.set",t,e||+new Date),alog.fire&&alog.fire("mark")},init:function(t){alog("speed.set","options",t)},view_start:t,tti:t,page_ready:t}}(),void function(t){var e=!1;t.onerror=function(t,n,a,o){var i=!0;return!n&&/^script error/i.test(t)&&(e?i=!1:e=!0),i&&alog("exception.send","exception",{msg:t,js:n,ln:a,col:o}),!1},alog("exception.on","catch",function(t){alog("exception.send","exception",{msg:t.msg,js:t.path,ln:t.ln,method:t.method,flag:"catch"})})}(window),window.xssfw=function(t){function e(t,e){if(!(10<++p)){var n={type:"INLINE",path:t,code:e.substr(0,400),len:e.length};alog("xss.send","xss",n),d.push(n),c()}}function n(t,e){if(t&&0!=e){var a=t.tagName;if("BODY"!=a){t.id&&(a+="#"+t.id);var o=t.className;return o&&(a+="."+o.split(" ")[0]),(o=n(t.parentNode,e-1))?o+" "+a:a}}}function a(t){for(var e=g.length-1;e>=0;e--){var n=g[e];if(t>n.b)return n}}function o(t){for(var e=f.length-1;e>=0;e--){var n=f[e];if(n.b.test(t))return n}}function i(t,i){function r(s){var l=s._k;if(l||(l=s._k=++u),l=l<<8|i,!m[l]&&(m[l]=!0,1==s.nodeType)){var f;s[t]&&(f=s.getAttribute(t))&&(l=a(f.length)||o(f)||{},l.a&&(s[t]=null),l.c&&e(n(s,5)+"["+t+"]",f)),c&&"A"==s.tagName&&"javascript:"==s.protocol&&(f=s.href.substr(11),l=a(f.length)||o(f)||{},l.a&&(s.href="javascript:void(0)"),l.c&&e(n(s,5)+"[href]",f)),r(s.parentNode)}}var c="onclick"==t;document.addEventListener(t.substr(2),function(t){r(t.target)},!0)}function r(t){var e=[];if(t)for(var n=t.length-1;n>=0;n--){var a=t[n],o=a.target;e.push({b:a.match,a:/D/.test(o),c:/W/.test(o)})}return e}function c(){if(s){for(var t=0;t<d.length;t++)s(d[t]);d=[]}}var s,l,f,g,p=0,m={},u=0,d=[],h={};return h.init=function(e){if(t.addEventListener&&!l){l=!0,alog("xss.create",{dv:5,postUrl:"https:"===document.location.protocol?"https://gsp0.baidu.com/5aAHeD3nKhI2p27j8IqW0jdnxx1xbK/tb/pms/img/st.gif":"https://gsp0.baidu.com/5aAHeD3nKhI2p27j8IqW0jdnxx1xbK/tb/pms/img/st.gif",page:"tb-xss"}),g=r(e["len-limit"]),f=r(e["key-limit"]),e=0;for(var n in document)/^on./i.test(n)&&i(n,e++)}},h.watch=function(t){s=t,c()},h}(this),xssfw.init({"len-limit":[{match:400,target:"Warn"}],"key-limit":[{match:/createElement/,target:"Warn"},{match:/fromCharCode|eval|getScript|xss/,target:"Warn,Deny"},{match:/alert\(|prompt/,target:"Warn"}]});</script><!--[if lt IE 9]><script>(function(){    var tags = ['header','footer','figure','figcaption','details','summary','hgroup','nav','aside','article','section','mark','abbr','meter','output','progress','time','video','audio','canvas','dialog'];    for(var i=tags.length - 1;i>-1;i--){ document.createElement(tags[i]);}})();</script><![endif]--><link rel="shortcut icon" href="//tb1.bdstatic.com/tb/favicon.ico" />
<link rel="stylesheet" href="//tb1.bdstatic.com/??tb/static-common/style/tb_ui_cc7cc6e.css,tb/static-common/style/tb_common_ae4a3c6.css" />
<link rel="stylesheet" href="//tb1.bdstatic.com/??/tb/_/card_86ffd75.css,/tb/_/js_pager_5be1e39.css,/tb/_/login_dialog_91323a4.css,/tb/_/user_head_35f26e0.css,/tb/_/icons_a2a62be.css,/tb/_/wallet_dialog_3dd7f7b.css,/tb/_/flash_lcs_d41d8cd.css,/tb/_/new_message_system_9425a2a.css,/tb/_/base_user_data_8391559.css,/tb/_/cashier_dialog_7b07e3f.css,/tb/_/qianbao_cashier_dialog_32966aa.css,/tb/_/base_dialog_user_bar_362ad46.css,/tb/_/qianbao_purchase_member_8559cf6.css,/tb/_/pay_member_d41d8cd.css,/tb/_/http_transform_d41d8cd.css,/tb/_/userbar_ced98ce.css,/tb/_/poptip_f0fdc70.css,/tb/_/feed_inject_d41d8cd.css,/tb/_/new_2_index_9b0a69e.css,/tb/_/pad_overlay_ea29d54.css,/tb/_/suggestion_21a5e89.css,/tb/_/search_bright_62ee7ff.css,/tb/_/top_banner_13dc075.css,/tb/_/couplet_78341bd.css,/tb/_/slide_show_aad29db.css,/tb/_/carousel_area_v3_614f6fa.css,/tb/_/interest_num_v2_fa3eaa9.css,/tb/_/shake_bear_f28994d.css,/tb/_/payment_dialog_title_c08d368.css,/tb/_/qianbao_purchase_tdou_4b31f54.css" />
<link rel="stylesheet" href="//tb1.bdstatic.com/??/tb/_/tdou_get_ee4778d.css,/tb/_/like_tip_a860770.css,/tb/_/tb_region_e558306.css,/tb/_/umoney_query_81b1336.css,/tb/_/nameplate_4111fc0.css,/tb/_/my_current_forum_247ac38.css,/tb/_/dialog_6ed86bb.css,/tb/_/cont_sign_card_84b27b3.css,/tb/_/sign_tip_cdf6543.css,/tb/_/sign_mod_bright_a8fa968.css,/tb/_/tb_spam_814ec43.css,/tb/_/my_tieba_90b3f2d.css,/tb/_/icon_tip_db299f2.css,/tb/_/popframe_1ca0182.css,/tb/_/scroll_panel_eb74727.css,/tb/_/often_visiting_forum_3068078.css,/tb/_/grade_f796a9b.css,/tb/_/onekey_sign_9010c3d.css,/tb/_/spage_game_tab_877aa7a.css,/tb/_/index_aggregate_entrance_506eea1.css,/tb/_/forum_directory_3f957c1.css,/tb/_/forum_rcmd_v2_150be0e.css,/tb/_/spage_liveshow_slide_b642d80.css,/tb/_/activity_carousel_9183806.css,/tb/_/affairs_nav_075fcf8.css,/tb/_/threadItem_2_2449095.css,/tb/_/affairs_1f1dc66.css,/tb/_/voice_efe9e00.css,/tb/_/tbshare_aa97892.css,/tb/_/aside_float_bar_5737c12.css" />
<link rel="stylesheet" href="//tb1.bdstatic.com/??/tb/_/app_download_7090d60.css,/tb/_/topic_rank_365c394.css,/tb/_/aside_v2_d3f83ef.css,/tb/_/feedback_b53cf14.css,/tb/_/common_footer_promote_e5dd6ab.css,/tb/_/new_footer_820125e.css,/tb/_/stats_6e5e6bc.css,/tb/_/tshow_out_date_warn_76587dd.css,/tb/_/ticket_warning_2081b92.css,/tb/_/member_upgrade_tip_a458868.css,/tb/_/fixed_bar_fbd9428.css,/tb/_/tpl_14_d41d8cd.css" />
        <script>    var PageData = {        "tbs": "",        "charset": "UTF-8",        "product": "index",        "page": "index"    };        PageData.user = {        "id": "",        "user_id": "",        "name": "",        "user_name": "",        "user_nickname" : "",        "name_url": "",        "no_un": 0,        "is_login": 0,        "portrait": "",        "balv": {}, /* Ban 杩欎釜妯″潡鐪熷璁ㄥ帉鐨?*/"Parr_props": null,"Parr_scores": null,"mParr_props": null,        "vipInfo": null,        "new_iconinfo": null,        "power": {}    };    PageData.search_what = "";    var Env = {        server_time: 1581579052000};    var Tbs = {"common":""};</script><script type="text/javascript">function resizePic_temp(e,t,i,s,n){function r(e,t,i,s){var n=0,r=e,a=t;switch(e>i&&(n+=1),t>s&&(n+=2),n){case 1:r=i,a=t*i/e;case 2:a=s,r=e*s/t;case 3:a=t/s>e/i?s:t*i/e,r=t/s>e/i?e*s/t:i}return 0!=n&&(l=!0),[r,a]}var a=t||120,c=i||120,l=!1,p=new Image;p.src=e.src;var h=r(p.width,p.height,a,c);return e.style.width=h[0]+"px",e.style.height=h[1]+"px","function"==typeof n&&n.apply(this,arguments),e.style.visibility="visible",1==s&&(e.style.marginTop=(i-parseInt(h[1]))/2+"px"),p=null,l}</script><script type="text/javascript">alog("speed.set","ht",new Date);</script><script>var _hmt = _hmt || [];(function() {  var hm = document.createElement("script");  hm.src = "https://hm.baidu.com/hm.js?98b9d8c2fd6608d564bf2ac2ae642948";  var s = document.getElementsByTagName("script")[0];   s.parentNode.insertBefore(hm, s);})();</script></head><body><script type="text/template" id="u_notify"><div class="u_notity_bd">    <ul class="sys_notify j_sys_notify j_category_list">    </ul>    <ul class="sys_notify_last">        <li class="category_item  category_item_last j_category_item_last">            <a target="_blank" href="/sysmsg/index?type=notity">                鎴戠殑閫氱煡<span class="unread_num">0</span>            </a>            <ul class="new_message j_new_message j_category_list">            </ul>        </li>    </ul></div></script><script type="text/template" id="u_notify_item"><%for (var i = 0; i < list.length; i++) {%>    <li class="category_item <% if(list[i].unread_count == 0) {%>category_item_empty<%}%>">    <%if ( list[i].category_href ) {%>    <a class="j_cleardata" href="<%=list[i].category_href%>" target="_blank" data-type="<%=list[i].type%>"><%=list[i].category_name%>        <% if(list[i].unread_count != 0) {%>            <span class="unread_num"><%=list[i].unread_count%></span>        <% } %>    </a>    <%} else {%>    <a href="/sysmsg/index?type=notity&category_id=<%=list[i].category_id%>" target="_blank" data-type="<%=list[i].type%>"><%=list[i].category_name%>        <% if(list[i].unread_count != 0) {%>            <span class="unread_num"><%=list[i].unread_count%></span>        <% } %>    </a>    <% } %>    </li><%}%></script><div id="local_flash_cnt"></div><div class="wrap1"><div class="wrap2"><script>PageData.tbs = "6e1dcd9b4c5eb12e1581579052";PageData.is_iPad = 0;PageData.itbtbs = "a906665dd7a0c1c7";PageData.userTages = {};</script><div class="page-container"><div class="search-sec"><div id="head" class="search_bright_index search_bright clearfix" style=""><div class="head_inner"><div class="search_top clearfix" ><div class="search_nav j_search_nav" style=""><a rel="noopener" param="wd" href=https://www.baidu.com/s?cl=3&amp; >缃戦〉</a><a rel="noopener" param="word" href="http://news.baidu.com/ns?cl=2&amp;rn=20&amp;tn=news&amp;">鏂伴椈</a><b>璐村惂</b><a rel="noopener" param="word" href="http://zhidao.baidu.com/q?ct=17&amp;pn=0&amp;tn=ikaslist&amp;rn=10&amp;">鐭ラ亾</a><a rel="noopener" param="word" href="http://www.baidu.com/sf/vsearch?pd=video&amp;tn=vsearch&amp;ct=301989888&amp;rn=20&amp;pn=0&amp;db=0&amp;s=21&amp;rsv_spt=11&amp;">瑙嗛</a><a rel="noopener" param="key" href="http://music.baidu.com/search?fr=tieba&">闊充箰</a><a rel="noopener" param="word" href="http://image.baidu.com/i?tn=baiduimage&amp;ct=201326592&amp;lm=-1&amp;cl=2&amp;ie=gbk&amp;">鍥剧墖</a><a rel="noopener" param="word" href="http://map.baidu.com/m?fr=map006&amp;">鍦板浘</a><a rel="noopener" href="http://wenku.baidu.com/search?fr=tieba&lm=0&od=0&" param="word">鏂囧簱</a></div></div><div class="search_main_wrap"><div class="search_main clearfix"><div class="search_form"><a rel="noopener" title="鍒拌创鍚ч椤? href="/" class="search_logo"  style=""></a>                <form name="f1" class="clearfix j_search_form" action="/f" id="tb_header_search_form"><input class="search_ipt search_inp_border j_search_input tb_header_search_input" name="kw1" value="" type="text" autocomplete="off" size="42" tabindex="1" id="wd1" maxlength="100" x-webkit-grammar="builtin:search" x-webkit-speech="true"/><input autocomplete="off" type="hidden" name="kw" value="" id="wd2" /><span class="search_btn_wrap search_btn_enter_ba_wrap"><a rel="noopener" class="search_btn search_btn_enter_ba j_enter_ba" href="#" onclick="return false;" onmousedown="this.className+=' search_btn_down'" onmouseout="this.className=this.className.replace('search_btn_down','')">杩涘叆璐村惂</a></span><span class="search_btn_wrap"><a rel="noopener" class="search_btn j_search_post" href="#" onclick="return false;" onmousedown="this.className+=' search_btn_down'" onmouseout="this.className=this.className.replace('search_btn_down','')">鍏ㄥ惂鎼滅储</a></span><a rel="noopener" class="senior-search-link" href="//tieba.baidu.com/f/search/adv">楂樼骇鎼滅储</a>                    </form><p style="display:none;" class="switch_radios"><input type="radio" class="nowtb" name="tb" id="nowtb"><label for="nowtb">鍚у唴鎼滅储</label><input type="radio" class="searchtb" name="tb" id="searchtb"><label for="searchtb">鎼滆创</label><input type="radio" class="authortb" name="tb" id="authortb"><label for="authortb">鎼滀汉</label><input type="radio" class="jointb" checked="checked" name="tb" id="jointb"><label for="jointb">杩涘惂</label><input type="radio" class="searchtag" name="tb" id="searchtag" style="display:none;"><label for="searchtag" style="display:none;">鎼滄爣绛?/label></p></div></div></div></div>  </div></div><div class="main-sec clearfix"><div class="top-sec clearfix"><div id="rec_left" class="rec_left"><div class="carousel_wrap tbui_slideshow_container" id="carousel_wrap"><ul class="img_list tbui_slideshow_list"><li alog-group="img_list_21pic" alog-alias="img_list_21pic" class="img_list_3pic tbui_slideshow_slide" style="display: none;"><div class="fd1bdb07b6  clearfix" ad-dom-img="true">
    <a href="/link?client_type=pc_web&tbjump=lid%253D1855570029_26931_13___0%2526url%253D3fe2sxFoUuut6lLf6JBtHvgSvejGRLzjyVW7zhkPlEOw6AZTOO_bdCLZHcLAlRCQCBhF1r3Y8IQ3FznznnISnDB9Ls5Pqn8jkNeMZba8aztOR-1IPUHVHW9LJNdCaTuHZZo5AN0cB-uRIBjqvP7QdQwsxMNeTRK1GSIDZyvjL0f792B5DSluEv1DHxMSK56yC_IHr6tSen_1GhYtYWKW2O2jFsuUBn8vnSmj5ymEWJJZqoJxoIxpOZeI-XcKfrcKpooXeqqkDBGTaLjXK9Wwmq0EZFojUReo3eDeq0VuEdrTcSIhFsUYagXwkECystYzH1LqIO_sbtYM1JKFMdC3vQVeMVmsmgke-wNbZwzqSwK2A5B9MXYUuUZb8YBRb1rmONh5AG9FeOysua1nSV6zvE3VOvmrU1fZtcAfY2f2xaQ1XnDoP5k3G267M29xK-T1pj7y57U_ME7CTesWoGWSfc-ATjFwGuXe-S1zVm37j80C74xWPZ-QdQP14xk8OjenMQyTf85NP81cO6rt02WGAgntFMboKy3aPYD1kzxRl1Rk65pH4dWa3Z4pzGnV_iNNL0QjK9zbMnD5c7j32xwsAECzM4lGZomvdluHu3iohUtrTkL09mjl5IxEYSwJaJzCdG5Xjnsi-LFUFDo5stEUDx8EIdx7tAjjLHD5RPS-dLWsOJpkspLN-B3h3bt3ZzlD63J7OHfgmzffFa9wUbUwcxMP0OLy4Du4h0MzxgZvAgcUfrQmMUNLnDzIhDC7J-GCP2zmKgRcEKfoSjagPEM_mptHT6mCP4QQAapmnMFq0Vw8O7QV6_CM17TlrDhXTxnMOSAdvhCd8oPXhgIiSWDKacHWamV61E-gdMke5J3YNV3kXHrg5XSfhrBcjYcE2OytEx07-wPOtYuAKpQDrVQRPitY07zlPtfdGf9Iyx07RWXen6VH7Sro2aW5vQ3-XkNSFLVrMeYzOZ17tEWMWOwdPvRDFgf4G241Hl-Blv94MYNzKZHbojhVZQBQMqnXyeBpdsxZtvLLWHv0MrZ3kVzATsuEq18a1lQvPlpCJindX6NvIJgWIOoEBrUak9vAdHM-z9zBmpzLf6nMWDYIHdftTrJGLTK4HAf3HZkHFHiGFONJ1tw-tsHJ2AfqT9-c0jU9cnqoEagQPVEWVRpwhw0rqTfy2GFhU59gKf1XWcHZ2fnLYThwgx8A2K8K6rYqLvqGRnXTT1epwndzHWU1_KWqSVipxgZadCI_mwVde26MiJ3RSt1OiIWIhFOOKevfd4hsLVHgZDwJ10wGPo_FXWCa8Cjus-wQC1VbYD8Fkzi5udPVrjokmWilydg4MQE3pQVgCn0njw&task=&locate=&page=index&type=click&url=http%3A%2F%2Ftieba.baidu.com%2Findex.html%3F&refer=&fid=&fname=&uid=&uname=&is_new_user=&tid=&_t=1581579052&obj_id=26931" target="_blank" class="j_click_stats" data-locate="鍥剧墖">
        <img src="https://tb1.bdstatic.com/tb/cms/ngmis/images/file_1581300641834.jpg" style="width: 100%;height: 100%;" ad-dom-img="true" alt="璐村惂鍥剧墖" title="璐村惂鍥剧墖"/>
        <div class="img_txt">
            <p class="img_title">2鏈?3鏃ユ柊鍨嬭偤鐐庣柅鎯?/p>
        </div>
    </a>
</div>
<div class="carousel_promotion_replacement" data-index="1" style="width:187px;height:180px;">
	<a rel="noopener" href="https://tieba.baidu.com/f?ie=utf-8&kw=%E8%B4%B4%E5%90%A7%E5%A8%B1%E4%B9%90" target="_blank">
		<img src="https://tb1.bdstatic.com/tb/r/image/2016-03-25/034117a81ce8afcd9bee4fcd73ee8561.png" alt="璐村惂鍥剧墖" title="璐村惂鍥剧墖" />
		<div class="img_txt">
            <p class="img_title">鐖辫眴浠瓑浣犳潵缈荤墝锛?/p>
        </div>
	</a>
</div> <div class="z2447df76d  clearfix" ad-dom-img="true">
    <a href="/link?client_type=pc_web&tbjump=lid%253D1855570029_26726_13___0%2526url%253D6e07D0Oorvxw01554onGW9oIYtjabVWLhko476DlCgefJe0CvJ5DHikTtwKkFAd2cWvRihwC5YUzGX-y__WCdYn0pJHJoktTPvjI40_lTRy9A3kpXxh-T92qXqQmZzlQQ1m3uEaLbVCNG_s8s02BFPz2ZhJ4KZ-48d3uCYb9hZjudgyvaKbXTYwOUlE6wC5F97hFxEecUP1PLhVnjZKLG019nihV-zXI_rzn-N0SrIwJ8aMjr-d-WpIVbiI-d0AjeJ8SyKDlsSVZmi_Ly3z5pIl-OQE3wJhb0CK6GTc1ESyv39u5O-6KsHQJQu3gymT5IHbVnhh_yF9jW_Gfw4c0VTls717SLn-g9Ubp9N2vnvujZByUtF6i0TVaPat7Wp2qVICkFGW2jN1lMX8It2_GaiERoF_wU9ER9AscM9456vHPjFCKx0uCF1KkYAuQAjJGi9qVWnxHfGn2FnIv9vtgKSvUgrN_bau6_kbCqeHYgm355EXb0LQhJIzf-yUMXjuIjRH9Hegl5q7dTy8PRIWUPV7M8xm-RaQM0Fk23EwlufgmXEBCgstlAFoBsCbZcffJgYX2HNhOGHyQOHndw_x0fuUcmIX8ZrK8u33lho5f1K6pE9c5Vy8QHeIHYddmtEbCHUuXBEQxibK16W6GcoEbYI-Sr1L2-60YxZlXWDKDdf72JmvTWbjmibubY6gQj40CNegR0hXFST61uaN_Y_-NeXQH7LwsMpu7cXeutZtbY0292RTq3iG469OKdfw9AS7c_hOlIJwV3ag2JhxFV8smAKYOmQ5aWXXnj5b3GwMaSqk5teg9el-4GiNRhU3lo1bYtECynfg8BhTPB6DnSn2eTyGIhkoGOgeYaioYFaB0ORGzVKgIaI84e766qDpMt_Oy9OpfRg2l6RMmVZ7pHGIiL_rlssiMWqpDMD9lIgo59orb1oIMCT81cReQi6I1lzad_vuQZrg9vGl6y7Zeku43pM7y26ZVlTjHddkahH4Nu4iLsGNR-vGC7uBijwczCLWX7e01k01JWFtrwtM7I9DTVOichniBKCsbHngoOWXkOm0YTrET-b3hcmbMsltlCjSiyi1jon1Fx7x_lfzvk3A02CB7HRG53XY3-HesBsQHVZJYoK5GzlJ0Kdeyw-y67n_KvH5QsmPqXwILBW-9TmDkN6v7nfFegIgARoJIytlnmQNiZ18cbryj20SRxkeDpkjEG5XvAr6dAUKNSv9GZMrmliAZuynECS_FyMe89tCXiXLqftLowSHCNKcaaoeriFstzy7bFp_mpbaNozopK9DGGyogUHLoDxPdNqg_JhrKbCQEYZMGyMkknojbybrLT8SGzB4Cm6nHpvo0JilTvnhJozzWx71IbA&task=&locate=&page=index&type=click&url=http%3A%2F%2Ftieba.baidu.com%2Findex.html%3F&refer=&fid=&fname=&uid=&uname=&is_new_user=&tid=&_t=1581579052&obj_id=26726" target="_blank" class="j_click_stats" data-locate="鍥剧墖">
        <img src="https://tb1.bdstatic.com/tb/cms/ngmis/images/file_1581502342824.jpg" style="width: 100%;height: 100%;" ad-dom-img="true" alt="璐村惂鍥剧墖" title="璐村惂鍥剧墖"/>
        <div class="img_txt">
            <p class="img_title">涓夋槦鍙戝竷S20绯诲垪鎵嬫満</p>
        </div>
    </a>
</div>

</li><li alog-group="img_list_3pic" alog-alias="img_list_3pic" class="img_list_3pic tbui_slideshow_slide" style="display: none;">          <div class="rf8b272285  clearfix" ad-dom-img="true">
    <a href="/link?client_type=pc_web&tbjump=lid%253D1855570029_36522_13___0%2526url%253DfabdJRQ9c0xtxSdvgQFnNFAAG8onuC7CnxW4dbw9Jtq2GtUIVbxgKFQ0PiJRTdLL9Av_9nPMsPA-Bpf666-5nVdv6iyH1ZSMlmjY-DoTgYNqngkSpba6wgeILfd2kcFFXXHF6aSbNDmSYwB8TREQS506NZiP1YsOMTE0MXe0Z86yA-VuDNmYKpS1Vb_7IVxBEsiT5HXka9e2BkIgyOU3n1XXJY8VpRkXlBjfrOHD39Ax3maMN6Qiz6GT1RmVe5KwGtNK3EkgO-ttvHySK5wikG0QJZmo3SlL0kc2wsGBJd8Tq5FCmW-yUHuxzPjVjyOwWXSVEy9aRBP1Dc_HfCFK2YxjrpQJItHhLD1AqQQzCxTnXStTcmGnas821qz_-1iuEKZL7DdW3dHnAWu9J5UvEM5eJspjch2L6609YdFHxvun5KlPYkoVYkMLQTd2Bj3-cRY__lQ7tgXnhpSS1JyDUXayhKGRdHA-OjCTaA5ZKgmJ5vVr7N2BkFwTg4GtMa9bFjK6ZcCa-lQWEXyQ7TuO4ERdwa541hj_QFLpFtkNb0K2_ts10bktyROhIedlKOHq4W66NxDK5O4UMfwYq5F69IfmueHU3aPOcku06p24UKNzKF-ysGLNOlSrnE284zDJhtxXDT4fBPoh56XixGJgU3U2-20kmN59dLiZlxk4V81i2gQuzbk5GtqZ_-zJco6F8he00N0a7CX_nZ8XCmWh8gDUygxEyjg7td-oP0BTtAehfDvYZdkLmMjUcmQQoeJteqeEwtfK88MGK-BRTzoPFaB_f4s9tDy7mVlCq4vxM_QpF0oPeL7vlyd1xanvxcAMceWjdPsfcYqZvdg0fDW2IlCC3JJ2F4hwCmDOQXMtV3Q8UZ435iuzTf5OVGRgqxrGgB__7ksPZXhTwPsEpmdDpzoDsIrSW09tpfLkexbM_D-t-AUX4J4YA6Fw0qGWTWfnznWvw7dae7Gu7F7ef6MZOXgy1EUrH35OmL4DQdF6gHb0ecyhuW5gJN9a0RbaoHyUfe1DRQM70fdLzml_NX0MZJNQo1Z4SYFVGL-SiaZ-7k_QjhmzdXE-eOrlGCL2gVQm6_Vw6Z3D2PzM44memsMz99upyfxTKQ5oMIIguMOGNvhE7cHBCFhweH8ZfZ99cktLO4yi65gUmxh--NJEdJfkTdzgv588ORTDJwkQ6zZ2lTaDTIxdQ4SqzYzeqFYVWmz66Aqu4PeGttLRSshogaLkVnRNwwsMVhwDTH4N8aPVmFDHH7Q2Lc4eQ8BwXjDBZR0ZSlI2oPBAMl6AiIV9_OgS5PSgaJGsKxbjPH8vlztSeDUbXfZbJgL0g0uyUtkJOuHHUKjWP53WD2Am0F51iLO-riKBUfAhUANsiMZ5Tmc-Ny5jKKStEw&task=&locate=&page=index&type=click&url=http%3A%2F%2Ftieba.baidu.com%2Findex.html%3F&refer=&fid=&fname=&uid=&uname=&is_new_user=&tid=&_t=1581579052&obj_id=36522" target="_blank" class="j_click_stats" data-locate="鍥剧墖">
        <img src="https://tb1.bdstatic.com/tb/cms/ngmis/images/file_1581501243150.jpg" style="width: 100%;height: 100%;" ad-dom-img="true" alt="璐村惂鍥剧墖" title="璐村惂鍥剧墖"/>
        <div class="img_txt">
            <p class="img_title">瑗胯棌鍞竴纭瘖鐥呬緥娌绘剤鍑洪櫌</p>
        </div>
    </a>
</div>
<div class="s7ca3cf4d8  clearfix" ad-dom-img="true">
    <a href="/link?client_type=pc_web&tbjump=lid%253D1855570029_26934_13___0%2526url%253Df807Z7_tdErMIwWZ0VIcPzus8AhZMOjhOnFI0S_mLoBwz9hGBGKzk0iG6xJ-p8c6b9kqN-VpdqeAbsiYTfGHbOy2AoLxSjQHeJVNRk6ajuxLZi1iiwfFAw78y9KFECjxUZ8_jrPJnIuPItI2PTbCy7d363V0X9mWJtlNXjOmHcG02ozLAXTKIU60iRbIs7m0pI-b6Fovm58-hrU0gTOhokscA4E8oUzKDYzLO1CsBGEu0CrMzXmr95mkCH9Ct-aMl9q3JV711G5Fm6KmEQfUlKm26jc7Q9jDeGjJOTrH9Jv-tChTKMhQ92lVDyUjCYsQI6CV5lfcJLdebRjkxq0TlGTq_tczqfRV9Clc8Rf4L79jSMkCwG9EgufAb0gZ9gjNXDX2g1_ibPWL4I6CnGNxdgMoHHInCsGDpM6Rxgikn_JodseA9Kls_lOZ7tftM0lqXZMJHHksP9NrQC4iPNnp6YVZbdoPqdJtoLJvSPdpsdMpNqRdKGeM3SVJDoGtmMfRDTuqCHydnQom00DFC2KlmxYziSPUvaqjpe51cW5iImY1qGvh1veeeSMp1yS6NFxB1wLRKETAK2XUOHd-uuVKkpjoB6opBLwLsi7iTeLKue0d7eh6LM5pnN4ZUMgWkWPnV3PBkpGX4Y0HoH3560GBInSCbbQj5mVqrC4r7GUgXdYATwgVriUVLJ5aIPtPZt2WDn_IXZFf1aKlJD1uSEJ1opQ-j6CJFhdJK9N_qsXsDxRoF9oLUPZh4AQATF4_CKhpTLFyvKAbmL1JhMAGRzCy2VYWZvPb3zikh6zpyFMj-MGGbQGWjxj-9X0KPJCD2Yug-H2ZsrjQF4Fu5cKacVn4nNcBjBiF_5JVfw5HiL5cnZBsoPCmoRcIcDApeAyuUUgr5V8iNhkMtfcqyVuxKZzzKGqWcmxwRB61nbu6TsuaP6HrWLeCWeQsrQ00jfU0QmfFn8uMNHJcqR515dT4zm2oNaruKbMQ5Gpsg4jWFC5oJL2ntlVIu_QbqaaMSzJzSfeqFGln3XRYTZ45NJzi1y0jew5mmwhKlR5hsnQujqjpL4mqZC6F0zTvZZDcP0ubgAcRGFt1BYSswaiAqN3D3wj9lmT39X5KZWHmdpPMsfq7Z5aJILpVGZ_0rJ5atNydy8Cxdx9UeqNu4QEFu7rrJWa9KrE46dxjz0yYnmHAphb0rrKPGpXZSgibXfT-y9AQybjA555lN8nXhxLweLTYUlJFIWs91MhwqOg0e_PH2vSRMaI8TjIa-EjyHUH6Q7MOZPv_PXdKVSMb1LjryVpPtZY1RVTvWID_qrhmDKf0nJr5r0VIA5Sp0YwNs7NP5YAVqrZl1cw0tn-FrDrxcP9ifuANm6GMm4agrZ7-Xhwbj9MhZq5xwLGY0g&task=&locate=&page=index&type=click&url=http%3A%2F%2Ftieba.baidu.com%2Findex.html%3F&refer=&fid=&fname=&uid=&uname=&is_new_user=&tid=&_t=1581579052&obj_id=26934" target="_blank" class="j_click_stats" data-locate="鍥剧墖">
        <img src="https://tb1.bdstatic.com/tb/cms/ngmis/images/file_1581409575937.jpg" style="width: 100%;height: 100%;" ad-dom-img="true" alt="璐村惂鍥剧墖" title="璐村惂鍥剧墖"/>
        <div class="img_txt">
            <p class="img_title">鏂板啝鐥呮瘨涓嶆€曠値鐑箍娑﹀ぉ姘?/p>
        </div>
    </a>
</div>
<div class="o1f9895447  clearfix" ad-dom-img="true">
    <a href="/link?client_type=pc_web&tbjump=lid%253D1855570029_26728_13___0%2526url%253Dbaf8tg-GLZirDXEHVgmj7OV2KfZP1VEzdvzMjul0RKlXmT_rS_MawIOECL_eaglBPTDUkCQwuFoZQC7c65uJu-hqgu1291841oqYUaQn_Ec1cZW50tMrXA_lwRXxkv7nc6ZNoNOwPLacoJzPeB9Vatrt20bgiiKBSbzC-qNTajTLCJPqzlS-ttNf9BH_n-5g5wU5KNsTS4-RoFEIsN5RXeFtQiFZCclctZQgOmLlknHFZ-iCWprNMgN6N6j1Y2C8mgbcxhekh1pBz8X6l04Wxe7ONN8Y1MiI4AYrDHRK5cK4fT2uj-oe2N3EQ1pDBgjUGEd5m1YJi1OkBsee9G6DGc2BTP_aNYhkYZA5ahBZXyn58fjbtQs9TKCtwOyfdOmjtoAVvcCbg1Q3bZs-FJTDePBk-35RP12txgbH5LMXxtWZ0Xv5PrzGc3aZop3_OAQVGQ_0FzaTzhRD--UMbphutjXp0dMfZPAlyGDu7IqHFW0PmDxunHkDsAsl5ikhgTKgFl8ZxGRaHXI6S90rV7AYh_PZ8hNxp7Q0z-XJY1x18Aa7oMpZRZHf9MaYwZs7FZUsoq0vPSQakp6lZeLXh9WMyJf_egC4c1n0ZKLikC6iegg5FDuu-XUmqFSfVXPWVU9AlGeXRcmzNdY6eF9ujDtN0tRKC35nysxMDesOKhxEDedKDoXQsAmqlj4JwkG2kLQkd7MUURcjHbHH6ODjLWIxttBeiKObmhZAD0sgTmzQ357Hp4hEjMdE8qhNiPjY15LjXDmyg4IMv3RubsvR3QZfCrS0NjTbrNg-eKo6Ke4KOzyfTXV5qtfW037TeqFz-U8IMGaVWlaT5jIDV4s07AEoKwi8CkUyGJMn68X_5abB140kXDZgQwY2K4x390XAZ8vnTVlpGcAPOmoDSCmzmYZ-yJTe6vGtUzqsq-8JxhAwQiDrfpAuN6Bdq_aO2lZ00PPTxcZaaG21HfiNRK6ZpFXlzXFelGpiSmFO4wLhfuSQ36afDQihdDyJKrDUeft6Hh4FcLuxiNWwYQm76Xw-cfPI85gO25JVeLs9TCaU5Hmj-XnbEeq_ytseAdnGp1ek0o71MZi_vJnM6fZnrTbm53Kzyad6m6JqxdZpsrhBFalABhSNlWg8IadNemlAbPNIUnqKCg6rpgk_FnS43pY1QAquLAmMTrm8sfKOAyjLHO40ph9Q&task=&locate=&page=index&type=click&url=http%3A%2F%2Ftieba.baidu.com%2Findex.html%3F&refer=&fid=&fname=&uid=&uname=&is_new_user=&tid=&_t=1581579052&obj_id=26728" target="_blank" class="j_click_stats" data-locate="鍥剧墖">
        <img src="https://tb1.bdstatic.com/tb/cms/ngmis/images/file_1581385467619.jpg" style="width: 100%;height: 100%;" ad-dom-img="true" alt="璐村惂鍥剧墖" title="璐村惂鍥剧墖"/>
        <div class="img_txt">
            <p class="img_title">銆婁娇鍛藉彫鍞ゃ€嬫柊浣滀粖骞村彂鍞?/p>
        </div>
    </a>
</div>

</li><li alog-group="img_list_1pic" alog-alias="img_list_1pic" class="img_list_1pic tbui_slideshow_slide" style="display: none;">          <div class="zb4a290776  clearfix" ad-dom-img="true">
    <a href="/link?client_type=pc_web&tbjump=lid%253D1855570029_23709_13___0%2526url%253D6b188rYOEBzlX6WLnjiR-xmyXEQvJu0rr3El0KPyAOFDz46mCmM7gLqOIomwDniVwVXiAp9-pHGR3GjnvqKcrM49JWUZXMlFIOUxjO9_eBVI9Dqg6WF1pXAJMIJfE7sQVKv8ve-CgJgGq9nhg2MNO56b3wiD6MoqsFqniVZUHltCh3GaConaqwSpKxN_NAN4A1yV-EI-emVrU-C2WlYijJK4jY5-38X12y5P-KGGdh7w1Z-n_t3hEDi9ys_b9dc6eJnz3Si7Cd2uWeRGp-ohI3OZMO5cwkolNF-Oh8vr50bkouohdfA-hGJMbwwxJqQesdeYriXOseWZpH2utaEqm3UOnZwPHk4_CVudrguDofuaA5nYVXE6MojF30_7obrFri6TyyFbRVgS3OnhulUlmzdQ-3Nq1u3XDwsVsEgx7PBFh6YG4Hcv4-kcKuvDP39HzB2RJjO04ZwUEWHDIPwvui0p_XnQkTbc0LdYMfPTI8iqqExe8puWNa6d8xD1u5yG882c_ke5tyWJb_MuzUuprFp4VBcDF1X2hCcqM-p4ygYlzGTP6wf03zKi4d-H2qbbiT0bzztAlDThDZ6XGpp8eX0Y0k03dqBD2qyeLSIA5K0wLpZjBRHaQqrEq_VELObF1hIwtWkvAoTyHPVNIKUiUmGzRCbhSzagdiKJrNMVyJfDJm4g_02LOvYvzBar20MTzad1xRsot_DP4QkUSNpn4yYyTNjmhB4laewAqNp8tLfdEDjdEU36627T0zDXUmpjNMGXFC5WrzNoROCNWdpRnRq5ssEh4pKW2fDAcniwB2tkLmLI9Nv5f9HfvMpM423jnzp6_56BPQexUq8zN8C3rqpv4YF3cr7UwLHk6LPE_ZYJwRNRqV8Tt4QUmNx2WUceOAr9VII7779z5kAaIC9YWb5bZRJW59Sx_8zwCzxR70eHZ0UZ_wZCBcrqznQHsAMAUGOJRqQcwt_6uqi00FjIYUy3ORY_2x5x5dfjzMyFsudoxnA_tHAcPG-KBIDwBkWxJQ5h3icz59UPY1I8BtLJvrc2qnAdwjg6vBYayDOAU9YHcj3CcdXKRx27hYEnqjk7lYBGkPzfwEV9eTuxR4eV04vzRWI6LxZQQS-nmu5AimwYTEwkGnkNemyJzKWZ&task=&locate=&page=index&type=click&url=http%3A%2F%2Ftieba.baidu.com%2Findex.html%3F&refer=&fid=&fname=&uid=&uname=&is_new_user=&tid=&_t=1581579052&obj_id=23709" target="_blank" class="j_click_stats" data-locate="鍥剧墖">
        <img src="https://tb1.bdstatic.com/tb/cms/ngmis/images/file_1580895007419.jpg" style="width: 100%;height: 100%;" ad-dom-img="true" alt="璐村惂鍥剧墖" title="璐村惂鍥剧墖"/>
        <div class="img_txt">
            <p class="img_title"></p>
        </div>
    </a>
</div>
</li><li alog-group="img_list_2pic" alog-alias="img_list_2pic" class="img_list_2pic tbui_slideshow_slide" style="display: none;">          <div class="carousel_promotion_replacement" data-index="7" style="width:280px;height:180px;">
	<a rel="noopener" href="https://tieba.baidu.com/f?kw=%E7%AB%9E%E6%8A%80%E6%B8%B8%E6%88%8F" target="_blank">
		<img src="https://tb1.bdstatic.com/tb/r/image/2018-01-17/149ff66c960363c086b12f7022fea755.jpeg" alt="璐村惂鍥剧墖" title="璐村惂鍥剧墖" />
		<div class="img_txt">
            <p class="img_title">鍚у弸浠殑鐢电珵姊︽兂鑱氶泦鍦?/p>
        </div>
	</a>
</div> <div class="carousel_promotion_replacement" data-index="8" style="width:400px;height:180px;">
	<a rel="noopener" href="https://gss3.bdstatic.com/y0kThD4a2gU2pMbgoY3K/index.php?r=selfbuilt%2Fservicelist&game_id=9" target="_blank">
		<img src="https://tb1.bdstatic.com/tb/r/image/2019-12-25/1caed6f669a17d5fd9b04dca657ed53a.png" alt="璐村惂鍥剧墖" title="璐村惂鍥剧墖" />
		<div class="img_txt">
            <p class="img_title">椋庨潯鍏ㄧ悆鐨勭儳鑴戠ぞ浜ゆ娓?/p>
        </div>
	</a>
</div> </li></ul></div><div class="top_progress_bar"><span id="top_progress_prev" class="top_progress_prev">1</span><ul id="img_page" class="img_page"><li>1</li><li>2</li><li>3</li><li>4</li></ul><span id="top_progress_next" class="top_progress_next">2</span></div></div><div class="index-forum-num-ten-millions"><div class="rec_right " id="rec_right"><div id="in_forum_num" class="num_list num_list02"><span class="num_span"><i class="num_icon"></i></span><span class="num_span"><i class="num_icon"></i></span><span class="num_span"><i class="num_icon"></i></span><span class="num_span"><i class="num_icon"></i></span><span class="num_span"><i class="num_icon"></i></span><span class="num_span"><i class="num_icon"></i></span><span class="num_span"><i class="num_icon"></i></span>         <span class="num_span"><i class="num_icon"></i></span>         </div><a rel="noopener" href="#" class="btn_login"></a></div></div></div><div class="middle-sec clearfix"><div class="de28b39df0  hide_style">
    <div class="iframe_wrapper clearfix">
    </div>
</div>
<div class="item media_item" id="media_item" alog-alias="media_item">
  <div class="item_hd">
            <span class="title">璐村惂濞变箰</span>
            <a class="verify_link" target="_blank" href="Mailto:tiebayule@baidu.com">鍚堜綔娌熼€?/a>
  </div>
  <a class="media" href="https://tieba.baidu.com/p/6448244014"鈭?target="_blank">
        <img src="https://tb1.bdstatic.com/tb/r/image/2020-01-17/b66ac3faa1a0f0416fea7eba32278f3c.png" height="90" width="220">
  </a>
  <ul class="media_list">
   <li>
        	<a href="https://tieba.baidu.com/p/6472401886" target="_blank">閲庣嫾disco娑夊珜鎶勮</a>
      </li>
     <li>
        	<a href="http://tieba.baidu.com/p/6469139647" target="_blank">褰辫鍓ф殏鍋滄媿鎽?/a>
      </li>
      
      <li>
        	<a href="http://tieba.baidu.com/p/6462729690" target="_blank">姣曞織椋炴€掑柗寰愬偿</a>
      </li>
  
    
  </ul>
</div> 

<div alog-alias="notice_item" id="notice_item" class="item notice_item">
   <div class="item_hd">
     	<span class="title">鍏憡鏉?/span>
  </div>
<a class="notice" href="http://tieba.baidu.com/p/5989281491">
      <img width="220" height="90" src="https://tb1.bdstatic.com/tb/%E8%B4%B4%E5%90%A7%E5%BC%80%E5%B1%95%E6%B6%89%E9%BB%84%E6%B6%89%E8%B5%8C%E4%B8%93%E9%A1%B9%E6%B8%85%E7%90%86%E8%A1%8C%E5%8A%A8.png">
  </a>
  <ul alog-group="notice_list" class="notice_list">

       <li>
        	<a href="http://tieba.baidu.com/p/5757349508 ">璐村惂寮€灞曡繚娉曡祵鍗氫笓椤规竻鐞嗚鍔?/a>
        </li>
 
        <li>
        	<a href="http://tieba.baidu.com/p/6210168245 ">鍏充簬绔ユ槦闈㈣瘯銆佹嫑鑱橀獥灞€鐩稿叧淇℃伅涓撻」娓呯悊閫氱煡</a>
        </li>

    </ul>
</div>



<script>
var pv_img = new Image();
pv_img.src = "https://gsp0.baidu.com/5aAHeD3nKhI2p27j8IqW0jdnxx1xbK/tb/img/pv.gif?fr=tb0_forum&st_mod=new_spage&st_type=pv_sum&_t="+(new Date().getTime());
</script></div></div></div></div><div class="bottom-bg"></div></div></div><div class="footer" alog-alias="footer">    <p alog-group="copy">&copy;2020 Baidu<a  rel="noreferrer" href="//www.baidu.com/duty/">浣跨敤鐧惧害鍓嶅繀璇?/a><a  rel="noreferrer" href="https://gsp0.baidu.com/5aAHeD3nKhI2p27j8IqW0jdnxx1xbK/tb/eula.html">璐村惂鍗忚</a><a href="https://tieba.baidu.com/tb/cms/tieba-fe/tieba_promise.html">闅愮鏀跨瓥</a><a  rel="noreferrer" href="//tieba.baidu.com/hermes/feedback">鎶曡瘔鍙嶉</a>淇℃伅缃戠粶浼犳挱瑙嗗惉鑺傜洰璁稿彲璇?0110516<a  rel="noreferrer" href="http://net.china.cn/chinese/index.htm" target="_blank"><img src="//tb2.bdstatic.com/tb/static-common/img/net_7a8d27a.png" width="20"></a><a  rel="noreferrer" href="http://www.bj.cyberpolice.cn/index.htm" target="_blank"><img title="棣栭兘缃戠粶110鎶ヨ鏈嶅姟" src="//tb2.bdstatic.com/tb/static-common/img/110_97c9232.png" width="20"></a></p></div><script>window.alogObjectConfig = {  product: '14',     page: '14_3',        monkey_page: 'tieba-index', speed_page: '3',         speed: {            sample: '0.2'   },        monkey: {            sample: '0.01',      hid: '762'       },        exception: {                 sample: '0.1'   },    };</script><div id="fixed_bar" class=""><img src="//tb1.bdstatic.com/tb/cms/PC%E7%AB%AF%E5%BA%95%E9%83%A8%E9%80%9A%E6%A0%8F%E5%BC%B9%E5%B1%821000x120.png" alt=""><img src="//tb2.bdstatic.com/tb/static-spage/widget/fixed_bar/images/icon_close_b98955a.png" alt="" class="close"></div>
 </div></div><script type="text/javascript">alog&&alog("speed.set","drt",+new Date);</script><script>PageUnitData={"search_input_tip":"\u8f93\u5165\u4f60\u611f\u5174\u8da3\u7684\u4e1c\u4e1c","dasense_messenger_whitelist":[["http:\/\/fedev.baidu.com"],["http:\/\/jiaoyu.baidu.com"],["http:\/\/caifu.baidu.com"],["http:\/\/jiankang.baidu.com"],["http:\/\/tieba.dre8.com"],["http:\/\/tdsp.nuomi.com"],["http:\/\/baiju.baidu.com"],["http:\/\/temai.baidu.com"],["http:\/\/tieba.baidu.com"],["http:\/\/zt.chuanke.com"],["http:\/\/window.sturgeon.mopaas.com"],["http:\/\/api.union.vip.com"],["http:\/\/api.dongqiudi.com"],["http:\/\/www.chuanke.com"],["http:\/\/dcp.kuaizitech.com\/"]],"like_tip_svip_black_list":"","icons_category":{"101":["\u5df4\u897f\u4e16\u754c\u676f"],"102":["\u661f\u5ea7\u5370\u8bb0"],"104":["\u8d34\u5427\u5370\u8bb0"]}};</script>
<script src="//tb1.bdstatic.com/??tb/static-common/lib/tb_lib_384d873.js,tb/static-common/ui/common_logic_v2_d234a93.js,tb/static-common/js/tb_ui_d99119b.js,/tb/_/ban_8422f6c.js"></script>
<script>    (function(F){var _JSSTAMP = {"spage\/component\/feed_data\/feed_data.js":"\/tb\/_\/feed_data_a6a0a69.js","common\/component\/dynamic_load\/dynamic_load.js":"\/tb\/_\/dynamic_load_b34ab50.js","common\/widget\/picture_rotation\/picture_rotation.js":"\/tb\/_\/picture_rotation_a27cde5.js","spage\/component\/popframe\/popframe.js":"\/tb\/_\/popframe_cd135d9.js","common\/component\/slide_show\/slide_show.js":"\/tb\/_\/slide_show_27c9ac6.js","common\/widget\/aside_float_bar\/aside_float_bar.js":"\/tb\/_\/aside_float_bar_fdcd8e0.js","common\/widget\/scroll_panel\/scroll_panel.js":"\/tb\/_\/scroll_panel_51b7780.js","common\/component\/often_visiting_forum\/often_visiting_forum.js":"\/tb\/_\/often_visiting_forum_4e65277.js","common\/widget\/tbshare\/tbshare.js":"\/tb\/_\/tbshare_0f12fc7.js","common\/widget\/lcs\/lcs.js":"\/tb\/_\/lcs_43bf602.js","common\/widget\/card\/card.js":"\/tb\/_\/card_99bd0cd.js","common\/widget\/wallet_dialog\/wallet_dialog.js":"\/tb\/_\/wallet_dialog_ab5f9b0.js","common\/widget\/new_message_system\/new_message_system.js":"\/tb\/_\/new_message_system_14c29aa.js","common\/widget\/cashier_dialog\/cashier_dialog.js":"\/tb\/_\/cashier_dialog_0c1473f.js","common\/widget\/messenger\/messenger.js":"\/tb\/_\/messenger_040cae5.js","common\/widget\/base_user_data\/base_user_data.js":"\/tb\/_\/base_user_data_72c8498.js","common\/widget\/pay_member\/pay_member.js":"\/tb\/_\/pay_member_440a15e.js","common\/widget\/http_transform\/http_transform.js":"\/tb\/_\/http_transform_e33a140.js","common\/widget\/login_dialog\/login_dialog.js":"\/tb\/_\/login_dialog_0844e5e.js","common\/widget\/search_handler\/search_handler.js":"\/tb\/_\/search_handler_638443d.js","common\/widget\/suggestion\/suggestion.js":"\/tb\/_\/suggestion_1902cc7.js","common\/widget\/animate_base\/animate_base.js":"\/tb\/_\/animate_base_51879f8.js","common\/component\/captcha\/captcha.js":"\/tb\/_\/captcha_28c5dc5.js","common\/component\/captcha_meizhi\/captcha_meizhi.js":"\/tb\/_\/captcha_meizhi_5f61aad.js","common\/component\/image_uploader\/image_uploader.js":"\/tb\/_\/image_uploader_bdbf433.js","common\/component\/image_exif\/image_exif.js":"\/tb\/_\/image_exif_1b57cf0.js","common\/component\/captcha_dialog\/captcha_dialog.js":"\/tb\/_\/captcha_dialog_b73e617.js","common\/component\/postor_service\/postor_service.js":"\/tb\/_\/postor_service_53ed8e8.js","common\/component\/scroll_panel\/scroll_panel.js":"\/tb\/_\/scroll_panel_9e28dd8.js","common\/component\/suggestion\/suggestion.js":"\/tb\/_\/suggestion_9b05426.js","common\/component\/toolbar\/toolbar.js":"\/tb\/_\/toolbar_5516683.js","common\/component\/sketchpad_dialog\/sketchpad_dialog.js":"\/tb\/_\/sketchpad_dialog_abf416a.js","common\/component\/tabs\/tabs.js":"\/tb\/_\/tabs_fca6d95.js","common\/widget\/word_limit\/word_limit.js":"\/tb\/_\/word_limit_c99778f.js","common\/component\/editor_pic\/editor_pic.js":"\/tb\/_\/editor_pic_e438e58.js","common\/component\/editor_video\/editor_video.js":"\/tb\/_\/editor_video_a1c7028.js","common\/component\/editor_smiley\/editor_smiley.js":"\/tb\/_\/editor_smiley_8ce8765.js","common\/component\/editor_music\/editor_music.js":"\/tb\/_\/editor_music_d741e09.js","common\/component\/editor_sketchpad\/editor_sketchpad.js":"\/tb\/_\/editor_sketchpad_796a342.js","common\/component\/area_select\/area_select.js":"\/tb\/_\/area_select_f396383.js","common\/component\/follower\/follower.js":"\/tb\/_\/follower_7ce74f3.js","common\/widget\/image_uploader_manager\/image_uploader_manager.js":"\/tb\/_\/image_uploader_manager_087e784.js","common\/component\/sketchpad\/sketchpad.js":"\/tb\/_\/sketchpad_2226150.js","common\/component\/interest_smiley\/interest_smiley.js":"\/tb\/_\/interest_smiley_c9da6ca.js","common\/component\/animate_keyframes_bouncein\/animate_keyframes_bouncein.js":"\/tb\/_\/animate_keyframes_bouncein_8d70c27.js","common\/component\/animate_keyframes_bounceout\/animate_keyframes_bounceout.js":"\/tb\/_\/animate_keyframes_bounceout_8f15463.js","common\/component\/animate_keyframes_fadein\/animate_keyframes_fadein.js":"\/tb\/_\/animate_keyframes_fadein_178e937.js","common\/component\/animate_keyframes_fadeout\/animate_keyframes_fadeout.js":"\/tb\/_\/animate_keyframes_fadeout_44f964c.js","common\/component\/animate_keyframes_flip\/animate_keyframes_flip.js":"\/tb\/_\/animate_keyframes_flip_44dec23.js","common\/component\/animate_keyframes_focus\/animate_keyframes_focus.js":"\/tb\/_\/animate_keyframes_focus_de0bedc.js","common\/component\/animate_keyframes_lightspeed\/animate_keyframes_lightspeed.js":"\/tb\/_\/animate_keyframes_lightspeed_6109fe5.js","common\/component\/animate_keyframes_rotatein\/animate_keyframes_rotatein.js":"\/tb\/_\/animate_keyframes_rotatein_0b7ba89.js","common\/component\/animate_keyframes_rotateout\/animate_keyframes_rotateout.js":"\/tb\/_\/animate_keyframes_rotateout_884da6a.js","common\/component\/animate_keyframes_slidein\/animate_keyframes_slidein.js":"\/tb\/_\/animate_keyframes_slidein_38b544d.js","common\/component\/animate_keyframes_slideout\/animate_keyframes_slideout.js":"\/tb\/_\/animate_keyframes_slideout_a86a043.js","common\/component\/animate_keyframes_special\/animate_keyframes_special.js":"\/tb\/_\/animate_keyframes_special_fa9a9be.js","common\/component\/animate_keyframes_zoomin\/animate_keyframes_zoomin.js":"\/tb\/_\/animate_keyframes_zoomin_9b12f77.js","common\/component\/animate_keyframes_zoomout\/animate_keyframes_zoomout.js":"\/tb\/_\/animate_keyframes_zoomout_73cbdb0.js","user\/widget\/icons\/icons.js":"\/tb\/_\/icons_cab285d.js","user\/widget\/user_api\/user_api.js":"\/tb\/_\/user_api_c1c17f1.js","common\/widget\/qianbao_purchase_member\/qianbao_purchase_member.js":"\/tb\/_\/qianbao_purchase_member_44418d7.js","common\/widget\/tdou\/tdou_open_type.js":"\/tb\/_\/tdou_open_type_6e74792.js","common\/widget\/qianbao_cashier_dialog\/qianbao_cashier_dialog.js":"\/tb\/_\/qianbao_cashier_dialog_5d910cd.js","common\/widget\/base_dialog_user_bar\/base_dialog_user_bar.js":"\/tb\/_\/base_dialog_user_bar_9d205a7.js","common\/widget\/show_dialog\/show_dialog.js":"\/tb\/_\/show_dialog_1644928.js","common\/widget\/placeholder\/placeholder.js":"\/tb\/_\/placeholder_e682b0c.js","common\/widget\/tbcopy\/tbcopy.js":"\/tb\/_\/tbcopy_a946019.js","common\/widget\/tdou_get\/tdou_get.js":"\/tb\/_\/tdou_get_ed8eb55.js","tbmall\/widget\/tbean_safe_ajax\/tbean_safe_ajax.js":"\/tb\/_\/tbean_safe_ajax_94e7ca2.js","common\/widget\/umoney_query\/umoney_query.js":"\/tb\/_\/umoney_query_c9b7960.js","common\/widget\/qianbao_purchase_tdou\/qianbao_purchase_tdou.js":"\/tb\/_\/qianbao_purchase_tdou_f7bef41.js","common\/widget\/umoney\/umoney.js":"\/tb\/_\/umoney_ed41085.js","common\/widget\/payment_dialog_title\/payment_dialog_title.js":"\/tb\/_\/payment_dialog_title_a606194.js","common\/widget\/tdou\/tdou_data.js":"\/tb\/_\/tdou_data_621617e.js","common\/widget\/tdou\/tdou_view_pay.js":"\/tb\/_\/tdou_view_pay_166b2c6.js","common\/widget\/audio_player\/audio_player.js":"\/tb\/_\/audio_player_3ce73ee.js","common\/widget\/voice_player\/voice_player.js":"\/tb\/_\/voice_player_9a9b6dc.js","common\/component\/js_pager\/js_pager.js":"\/tb\/_\/js_pager_ebc4a27.js","common\/widget\/user_head\/user_head.js":"\/tb\/_\/user_head_e60a83e.js","common\/component\/image_previewer\/image_previewer.js":"\/tb\/_\/image_previewer_73d5f03.js","common\/component\/image_editor\/image_editor.js":"\/tb\/_\/image_editor_7d1aff6.js","common\/component\/image_previewer_list\/image_previewer_list.js":"\/tb\/_\/image_previewer_list_67f0ada.js","common\/component\/image_previewer_rotate\/image_previewer_rotate.js":"\/tb\/_\/image_previewer_rotate_c7cabf0.js","common\/component\/image_uploader_queue\/image_uploader_queue.js":"\/tb\/_\/image_uploader_queue_233b0a9.js","common\/component\/image_progress_bar\/image_progress_bar.js":"\/tb\/_\/image_progress_bar_12c6eb4.js","common\/widget\/block_user\/block_user.js":"\/tb\/_\/block_user_639364d.js","common\/widget\/pic_poster\/pic_poster.js":"\/tb\/_\/pic_poster_08a159b.js","common\/component\/image_water\/image_water.js":"\/tb\/_\/image_water_bc89548.js","common\/component\/image_flash_editor\/image_flash_editor.js":"\/tb\/_\/image_flash_editor_7e35ada.js","common\/widget\/params_xss_handler\/params_xss_handler.js":"\/tb\/_\/params_xss_handler_bbb0828.js","common\/widget\/bsk_service\/bsk_service.js":"\/tb\/_\/bsk_service_72c6560.js","common\/component\/select\/select.js":"\/tb\/_\/select_8d82f79.js","ucenter\/widget\/like_tip\/like_tip.js":"\/tb\/_\/like_tip_f90e312.js","tbmall\/widget\/tbean_safe\/tbean_safe.js":"\/tb\/_\/tbean_safe_14418e9.js","common\/widget\/jiyan_service\/jiyan_service.js":"\/tb\/_\/jiyan_service_44ae7c8.js","common\/widget\/post_service\/post_service.js":"\/tb\/_\/post_service_844b124.js","common\/widget\/post_prefix\/post_prefix.js":"\/tb\/_\/post_prefix_00d6326.js","common\/widget\/post_signature\/post_signature.js":"\/tb\/_\/post_signature_8c3c4ae.js","common\/widget\/mouse_pwd\/mouse_pwd.js":"\/tb\/_\/mouse_pwd_f31d0b4.js","common\/component\/bubble_factory\/bubble_factory.js":"\/tb\/_\/bubble_factory_f970c47.js","common\/component\/quick_reply_edit\/quick_reply_edit.js":"\/tb\/_\/quick_reply_edit_9678e8c.js","common\/widget\/paypost_data\/paypost_data.js":"\/tb\/_\/paypost_data_62a7ae4.js","props\/widget\/props_data\/props_data.js":"\/tb\/_\/props_data_a25400f.js","common\/component\/slide_select\/slide_select.js":"\/tb\/_\/slide_select_01ec4cf.js","common\/component\/post_props\/post_props.js":"\/tb\/_\/post_props_73bc086.js","common\/component\/attachment_uploader\/attachment_uploader.js":"\/tb\/_\/attachment_uploader_a9da3e8.js","common\/component\/picture_album_selector\/picture_album_selector.js":"\/tb\/_\/picture_album_selector_6b0a6cf.js","common\/component\/picture_selector\/picture_selector.js":"\/tb\/_\/picture_selector_a2c03a8.js","common\/component\/picture_uploader\/picture_uploader.js":"\/tb\/_\/picture_uploader_37f84f9.js","common\/component\/picture_web_selector\/picture_web_selector.js":"\/tb\/_\/picture_web_selector_f9e9dfc.js","common\/component\/scrawl\/scrawl.js":"\/tb\/_\/scrawl_e0ae790.js","common\/component\/ueditor_emotion\/ueditor_emotion.js":"\/tb\/_\/ueditor_emotion_a903913.js","common\/component\/ueditor_music\/ueditor_music.js":"\/tb\/_\/ueditor_music_0276fa3.js","common\/component\/ueditor_video\/ueditor_video.js":"\/tb\/_\/ueditor_video_9daefbc.js","common\/component\/colorful\/colorful.js":"\/tb\/_\/colorful_4f85c36.js","common\/component\/custom_emotion\/custom_emotion.js":"\/tb\/_\/custom_emotion_c7f15af.js","common\/component\/post_bubble\/post_bubble.js":"\/tb\/_\/post_bubble_9f3833e.js","common\/component\/tb_gram\/tb_gram.js":"\/tb\/_\/tb_gram_5afa029.js","common\/component\/formula\/formula.js":"\/tb\/_\/formula_58b7814.js","common\/component\/post_setting\/post_setting.js":"\/tb\/_\/post_setting_ac01c94.js","common\/component\/paypost\/paypost.js":"\/tb\/_\/paypost_62e57ae.js","common\/widget\/join_vip_dialog\/join_vip_dialog.js":"\/tb\/_\/join_vip_dialog_e8b24ea.js","common\/component\/quick_reply_data_handler\/quick_reply_data_handler.js":"\/tb\/_\/quick_reply_data_handler_256a70d.js","common\/widget\/detect_manager_block\/detect_manager_block.js":"\/tb\/_\/detect_manager_block_713b838.js","common\/widget\/verify_manager_phone\/verify_manager_phone.js":"\/tb\/_\/verify_manager_phone_6f07b28.js","common\/widget\/tb_lcs\/tb_lcs.js":"\/tb\/_\/tb_lcs_544b5c9.js","common\/widget\/event_center\/event_center.js":"\/tb\/_\/event_center_ca531c9.js"};         F.tbConfig(_JSSTAMP);    })(F);</script><script src="//tb1.bdstatic.com/??/tb/_/app_f6b8e80.js,/tb/_/card_99bd0cd.js,/tb/_/js_pager_ebc4a27.js,/tb/_/login_dialog_0844e5e.js,/tb/_/user_head_e60a83e.js,/tb/_/user_api_c1c17f1.js,/tb/_/icons_cab285d.js,/tb/_/wallet_dialog_ab5f9b0.js,/tb/_/event_center_ca531c9.js,/tb/_/lcs_43bf602.js,/tb/_/tb_lcs_544b5c9.js,/tb/_/flash_lcs_ccd5d3e.js,/tb/_/new_message_system_14c29aa.js,/tb/_/messenger_040cae5.js,/tb/_/base_user_data_72c8498.js,/tb/_/cashier_dialog_0c1473f.js,/tb/_/qianbao_cashier_dialog_5d910cd.js,/tb/_/base_dialog_user_bar_9d205a7.js,/tb/_/qianbao_purchase_member_44418d7.js,/tb/_/pay_member_440a15e.js,/tb/_/http_transform_e33a140.js,/tb/_/userbar_a657792.js,/tb/_/footer_af59471.js,/tb/_/poptip_74068e9.js,/tb/_/ad_stats_008fc58.js,/tb/_/feed_inject_994d342.js,/tb/_/pad_overlay_c504049.js,/tb/_/search_handler_638443d.js,/tb/_/suggestion_1902cc7.js,/tb/_/search_bright_43d6f61.js"></script>
<script src="//tb1.bdstatic.com/??/tb/_/top_banner_a407e3f.js,/tb/_/couplet_44a34e5.js,/tb/_/data_handler_async_02038fe.js,/tb/_/loader_12269f6.js,/tb/_/slide_show_27c9ac6.js,/tb/_/carousel_area_v3_ca08742.js,/tb/_/interest_num_v2_def412c.js,/tb/_/tbskin_spage_83fcb9c.js,/tb/_/payment_dialog_title_a606194.js,/tb/_/show_dialog_1644928.js,/tb/_/qianbao_purchase_tdou_f7bef41.js,/tb/_/tdou_get_ed8eb55.js,/tb/_/tcharge_dialog_740d080.js,/tb/_/tool_async_fbb5605.js,/tb/_/loader_async_4df6f1e.js,/tb/_/like_tip_f90e312.js,/tb/_/member_api_722ea18.js,/tb/_/umoney_query_c9b7960.js,/tb/_/nameplate_d41d8cd.js,/tb/_/my_current_forum_97a6354.js,/tb/_/util_02ce566.js,/tb/_/dialog_dc42202.js,/tb/_/join_vip_dialog_e8b24ea.js,/tb/_/cont_sign_card_4b8b045.js,/tb/_/sign_mod_bright_4d57032.js,/tb/_/tb_spam_846204f.js,/tb/_/my_tieba_fc3f171.js,/tb/_/icon_tip_0e9f093.js,/tb/_/tdou_view_pay_166b2c6.js,/tb/_/popframe_cd135d9.js"></script>
<script src="//tb1.bdstatic.com/??/tb/_/scroll_panel_51b7780.js,/tb/_/often_visiting_forum_4e65277.js,/tb/_/onekey_sign_e4980db.js,/tb/_/spage_game_tab_d1d31b8.js,/tb/_/forum_directory_e748a6f.js,/tb/_/forum_rcmd_v2_be20584.js,/tb/_/spage_liveshow_slide_e85acf4.js,/tb/_/activity_carousel_d41d8cd.js,/tb/_/affairs_nav_c9a217a.js,/tb/_/feed_data_a6a0a69.js,/tb/_/dynamic_load_b34ab50.js,/tb/_/picture_rotation_a27cde5.js,/tb/_/affairs_7523160.js,/tb/_/audio_player_3ce73ee.js,/tb/_/voice_player_9a9b6dc.js,/tb/_/voice_0a018d7.js,/tb/_/tbcopy_a946019.js,/tb/_/tbshare_0f12fc7.js,/tb/_/aside_float_bar_fdcd8e0.js,/tb/_/topic_rank_dca38e6.js,/tb/_/aside_v2_a08ee35.js,/tb/_/feedback_22ddff1.js,/tb/_/common_footer_promote_0840273.js,/tb/_/new_footer_d41d8cd.js,/tb/_/stats_d5037da.js,/tb/_/tshow_out_date_warn_1e57655.js,/tb/_/ticket_warning_db7d890.js,/tb/_/member_upgrade_tip_e62bcc1.js,/tb/_/fixed_bar_163a2a0.js,/tb/_/new_2_index_49d35b7.js"></script>
<script src="//tb1.bdstatic.com/??/tb/_/tdou_view_pay_166b2c6.js,/tb/_/tpl_14_bb1002a.js,/tb/_/tpl_5_b1d3505.js"></script>
<script>window.modDiscardTemplate={};</script>
`
