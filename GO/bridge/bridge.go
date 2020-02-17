package bridge

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/astaxie/beego/logs"
)

var isCreate = false               //实例是否已被创建
var sendMsgMutex *sync.Mutex       //允许多线程下安全调用sendMsg()
type MsgHandler func(string) error //消息处理函数类型

/*
传输数据单元结构体
key="[fail]" => go client can't dial
key="[success]" => go client dial success
key="[disconnect]" => let go client close
*/
type Unit struct {
	Key     string
	Content string
}

type Bridge struct {
	conn      net.Conn
	isConnect bool         //socket 链接是否已建立
	mtu       int          //最大传输单元
	port      int          //监听端口
	buf       string       //消息缓存
	msgChan   *chan string //向外界提供的直接向qt端发送消息的管道
	handleMap map[string]MsgHandler
}

func init() {
	logs.EnableFuncCallDepth(true)
	logs.SetLogFuncCallDepth(3)
	sendMsgMutex = new(sync.Mutex)
}

/*
根据传入参数创建一个Bridge实例
mtu, 最大传输单元，单位是字节
port, 使用端口
*/
func GetBridge(mtu, port int) (*Bridge, error) {
	var err error
	if isCreate {
		err = errors.New("Already create one before!")
		logs.Error(err)
		return nil, err
	}
	var newBridge *Bridge
	newBridge = new(Bridge)
	newBridge.mtu = mtu
	newBridge.port = port
	newBridge.conn = nil
	newBridge.isConnect = false
	newBridge.handleMap = make(map[string]MsgHandler)
	newBridge.msgChan = new(chan string)
	*newBridge.msgChan = make(chan string, 50)
	isCreate = true
	return newBridge, nil
}

/*
注册一个消息处理函数
keyword: 关键字
handler: 消息处理函数
*/
func (b *Bridge) RegisterFunc(keyword string, handler MsgHandler) error {
	var err error
	if len(keyword) < 3 || strings.Contains(keyword, " ") {
		err = errors.New("keyword illeagle: keyword=" + keyword)
		logs.Error(err)
		return err
	}
	if handler == nil {
		err = errors.New("handler is null")
		logs.Error(err)
		return err
	}
	tmpHandle, success := b.handleMap[keyword]
	if success || tmpHandle != nil {
		err = errors.New("keyword already exist: keyword=" + keyword)
		logs.Error(err)
		return err
	}
	logs.Info("Register function success, keyword=%s", keyword)
	b.handleMap[keyword] = handler
	return nil
}

/*
开始监听，建立连接，等待消息并处理消息
注意若监听成功后本函数将会进入堵塞状态直至程序结束
*/
func (b *Bridge) ListenAndServer() error {
	var err error
	//建立链接
	if err = b.connect(); err != nil {
		logs.Error(err)
		return err
	}
	//等待接收消息,通过msgChan得到处理后的消息
	msgChan := make(chan *Unit, 100)
	go b.waitAndPreHandle(msgChan)
	//等待调用者发来的消息并向qt端转发
	go b.waitAndSendMessage()
	//处理接收到的消息
	for {
		newMsg, more := <-msgChan
		if more {
			go func() { //避免堵塞
				err = b.handleMessage(newMsg.Key, newMsg.Content)
				if err != nil {
					logs.Error("handler return a error: keyword=%s  err=%v", newMsg, err)
					b.SendMessage("error", fmt.Sprint(err))
				}
			}()
		} else {
			logs.Info("Go socket client close!")
			break
		}
	}
	return nil
}

/*
向qt端发送一条消息
key: 关键字
data: 任意格式的数据
*/
func (b *Bridge) SendMessage(key string, data interface{}) error {
	var err error
	if b.isConnect == false || b.conn == nil {
		err = errors.New("No connection have been created")
		logs.Error(err)
		return err
	}
	if len(key) < 3 || strings.Contains(key, " ") || strings.Contains(key, "@") {
		err = fmt.Errorf("keyword illeagle. key=%s", key)
		logs.Error(err)
		return err
	}
	if data == nil {
		err = fmt.Errorf("Receive data is null")
		logs.Error(err)
		return err
	}
	//处理data到字符串，若data不是字符串类型则需转json
	var tmpStr string
	if reflect.TypeOf(data).String() == "string" {
		tmpStr = data.(string)
	} else {
		tmpbytes, err := json.Marshal(data)
		if err != nil {
			return err
		}
		tmpStr = fmt.Sprintf("%s", tmpbytes)
	}
	//通过简单的传输协议来实现透明传输
	strings.ReplaceAll(tmpStr, "#", "^#")
	sendStr := fmt.Sprintf("%s@%s\\#", key, tmpStr)
	sendMsgMutex.Lock()
	fmt.Fprintf(b.conn, sendStr) //发出数据
	time.Sleep(15 * time.Millisecond)
	sendMsgMutex.Unlock()
	return nil
}

//向外界提供msgChan，提供直接向qt端发送数据的方式
func (b *Bridge) GetMsgChan() (*chan string, error) {
	if !isCreate {
		return nil, errors.New("Bridge object not been created")
	}
	if b.msgChan == nil {
		return nil, errors.New("msgChan is nil")
	}
	return b.msgChan, nil
}

//============== 私有函数 ===================

/*
等待消息并进行预处理，将处理后的消息通过msgChan通道返回
注意本函数非公用作且需要在新的携程中一直运行直至程序结束
为实现透明传输，qt端传输局时进行了一定处理，在循环里不断接受并逆向还原数据
传输协议：每条字符串以'#'结尾，‘\’为转意字符，通过‘@’来划分key和content
过程：
1.找到第一个#之前的字符串，不一定是条完整数据
2.若不完整则先追加到tmp中然后下个循环，否则成功读取得一条完整消息的末尾
*/
func (b *Bridge) waitAndPreHandle(msgChan chan<- *Unit) {
	var err error
	if !b.isConnect {
		err = errors.New("no connection have been created")
		logs.Error(err)
		close(msgChan)
		return
	}
	reader := bufio.NewReader(b.conn)
	for {
		//尝试完整读取一条数据
		tmp, err := reader.ReadString(byte('#'))
		if err != nil {
			logs.Error("ReadString fail: %s", err)
			break
		}
		b.buf += tmp
		if !strings.HasSuffix(tmp, `\#`) {
			logs.Info("Not end message: %s", b.buf)
			continue
		} else {
			reader.Reset(b.conn)
		}
		//简单处理以还原数据
		b.buf = b.buf[0 : len(b.buf)-2]
		b.buf = strings.ReplaceAll(b.buf, "^#", "#")
		msg := new(Unit)
		idx := strings.Index(b.buf, "@")
		if idx < 1 {
			logs.Error("Receive data do not found '@': %s", b.buf)
			b.buf = ""
			continue
		}
		msg.Key = b.buf[0:idx]
		msg.Content = b.buf[idx+1:]
		b.buf = ""
		//若qt端主动断开连接则结束循环
		if msg.Key == "[disconnect]" {
			logs.Info("Receive disconnect signal, going to shutdown...")
			close(msgChan)
			break
		}
		//得到qt端发来的完整一条消息，开始处理
		msgChan <- msg
	}
	//连接已断开，结束程序
	b.conn.Close()
	close(msgChan)
	os.Exit(0)
}

/*
不断等待 msgChan 里面发送来的字符串，将受到的消息直接向Qt端发生
不会造成堵塞，需要在单独一个协程里面运作
*/
func (b *Bridge) waitAndSendMessage() {
	if !isCreate {
		logs.Emergency("Bridge object not been created")
	}
	if b.msgChan == nil {
		logs.Emergency("msgChan not ready")
	}
	logs.Info("waitAndSendMessage() is running in background...")
	for {
		message, more := <-(*b.msgChan)
		if more {
			spiltRes := strings.Split(message, "@")
			if len(spiltRes) != 2 {
				logs.Warn("skip a worng message: message=%s", message)
				continue
			}
			b.SendMessage(spiltRes[0], spiltRes[1])
			continue
		}
		break
	}
	logs.Warn("waitAndSendMessage() exit because msgChan have been close")
	return
}

/*
分配消息到对应的消息处理函数进行处理
keyword: 消息的关键字
content: 消息的主题内容
*/
func (b *Bridge) handleMessage(keyword, content string) error {
	if len(keyword) < 3 || strings.Contains(keyword, " ") {
		return errors.New("keyword illeagle, keyword=" + keyword)
	}
	if content == "" {
		return errors.New("content is empty")
	}
	tmpHandler := b.handleMap[keyword]
	if tmpHandler == nil {
		return errors.New("keyword not register: keyword=" + keyword)
	}
	err := tmpHandler(content)
	if err != nil {
		logs.Warn("tmpHandle return a error: err=", err)
	}
	return err
}

/*
与qt端建立 socket 连接
注意本函数非公有函数
*/
func (b *Bridge) connect() error {
	var err error
	if b.isConnect {
		return errors.New("connection already created")
	}
	b.conn, err = net.Dial("tcp", fmt.Sprintf(":%d", b.port))
	if err != nil {
		logs.Error("Dial fail: %s", err)
		return err
	}
	b.isConnect = true
	logs.Info("Connect Qt Server success!")
	return nil
}
