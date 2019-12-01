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

//connect to Qt TCP server and constanly wait for the message,
//note that it function need to run in a gorounting
func (b *Bridge) Start(msgChan chan<- *Unit) {
	var err error
	b.conn, err = net.Dial("tcp", fmt.Sprintf(":%d", b.port))
	if err != nil {
		logs.Error("Dial fail: %s", err)
		msg := new(Unit)
		msg.Key = "[fail]"
		msg.Content = fmt.Sprintf("%s", err)
		msgChan <- msg
		return
	} else {
		msg := new(Unit)
		msg.Key = "[success]"
		msg.Content = "success"
		logs.Info("Connect Qt Server success!")
		msgChan <- msg
	}
	//transparent transmission
	var buf = ""
	reader := bufio.NewReader(b.conn)
	for {
		tmp, err := reader.ReadString(byte('#'))
		if err != nil {
			logs.Error("ReadString fail: %s", err)
			break
		}
		buf += tmp
		if !strings.HasSuffix(tmp, `\#`) { //not end
			logs.Info("Not end message: %s", buf)
			continue
		} else {
			reader.Reset(b.conn)
		}
		buf = buf[0 : len(buf)-2] //remove '\#'
		buf = strings.ReplaceAll(buf, "^#", "#")
		msg := new(Unit)
		idx := strings.Index(buf, "@")
		if idx < 1 { //discard a Unrecognized message
			logs.Error("Receive data do not found '@': %s", buf)
			buf = ""
			continue
		}
		msg.Key = buf[0:idx]
		msg.Content = buf[idx+1:]
		buf = ""
		if msg.Key == "[disconnect]" { //qt server initiative close the connection
			logs.Info("Receive disconnect signal, going to shutdown...")
			close(msgChan)
			break
		}
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
