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

var isCreate = false

/* Notice
key="[fail]" => go client can't dial
key="[success]" => go client dial success
key="[disconnect]" => let go client close
*/

type Unit struct {
	Key     string
	Content string
}

type Bridge struct {
	conn net.Conn
	mtu  int //max translate unite
	port int // listen at port
	mu   sync.Mutex
}

func init() {
	logs.EnableFuncCallDepth(true)
	logs.SetLogFuncCallDepth(3)
}

//Create a Bridge by given config
func GetBridge(mtu, port int) (*Bridge, error) {
	if isCreate {
		return nil, errors.New("Already create one before!")
	}
	var newBridge *Bridge
	newBridge = new(Bridge)
	newBridge.mtu = mtu
	newBridge.port = port
	newBridge.conn = nil
	isCreate = true
	return newBridge, nil
}

//发出连接请求并等待qt端回复，注意本函数在独立的协程中运行
//msgChan有于与主进程进行数据交换
func (b *Bridge) Start(msgChan chan<- *Unit) {
	var err error
	msg := new(Unit) //用户暂存解析得到的数据
	//开始尝试连接
	b.conn, err = net.Dial("tcp", fmt.Sprintf(":%d", b.port))
	if err != nil {
		logs.Error("Dial fail: %s", err)
		msg.Key = "[fail]"
		msg.Content = fmt.Sprintf("%s", err)
		msgChan <- msg
		return
	} else {
		msg.Key = "[success]"
		msg.Content = "success"
		logs.Info("Connect Qt Server success!")
		msgChan <- msg
	}
	var buf = ""
	reader := bufio.NewReader(b.conn)
	/*
		已连接成功，等待数据来到并处理
		为实现透明传输，qt端传输局时进行了一定处理，在循环里不断接受并逆向还原数据
		传输协议：每条字符串以'#'结尾，‘\’为转意字符，通过‘@’来划分key和content
		过程：
		1.找到第一个#之前的字符串，不一定是条完整数据
		2.若不完整则先追加到tmp中然后下个循环，否则成功读取得一条完整消息的末尾
	*/
	for {
		//尝试完整读取一条数据
		tmp, err := reader.ReadString(byte('#'))
		if err != nil {
			logs.Error("ReadString fail: %s", err)
			break
		}
		buf += tmp
		if !strings.HasSuffix(tmp, `\#`) {
			logs.Info("Not end message: %s", buf)
			continue
		} else {
			reader.Reset(b.conn)
		}
		//简单处理以还原数据
		buf = buf[0 : len(buf)-2]
		buf = strings.ReplaceAll(buf, "^#", "#")
		msg := new(Unit)
		idx := strings.Index(buf, "@")
		if idx < 1 {
			logs.Error("Receive data do not found '@': %s", buf)
			buf = ""
			continue
		}
		msg.Key = buf[0:idx]
		msg.Content = buf[idx+1:]
		buf = ""
		//若qt端主动断开连接则结束循环
		if msg.Key == "[disconnect]" {
			logs.Info("Receive disconnect signal, going to shutdown...")
			close(msgChan)
			break
		}
		//得到qt端发来的完整一条消息，开始处理
		msgChan <- msg
	}
	b.conn.Close()
	os.Exit(0)
}

//send some data to Qt TCP server
func (b *Bridge) SendMessage(key string, data interface{}) error {
	if b.conn == nil {
		err := errors.New("Socket not ready!")
		return err
	}
	if len(key) < 3 {
		err := fmt.Errorf("key is too short: '%s'", key)
		return err
	}
	if data == nil {
		err := fmt.Errorf("Receive null pointer to data!")
		return err
	}
	if strings.Index(key, "@") >= 0 {
		err := fmt.Errorf("Found char '@' in given key: '%s'", key)
		return err
	}
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
	strings.ReplaceAll(tmpStr, "#", "^#")
	sendStr := fmt.Sprintf("%s@%s\\#", key, tmpStr)
	b.mu.Lock()
	fmt.Fprintf(b.conn, sendStr)
	time.Sleep(15 * time.Millisecond)
	b.mu.Unlock()
	return nil
}
