package modules

import (
	"agent/config"
	"agent/kits"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"math/rand"
	"net/url"
	"os"
	"strconv"
	"time"
)

var dailCount int

type WebsocketClientManager struct {
	Conn    *websocket.Conn
	Addr    *string
	Path    string
	IsAlive bool
	Timeout int
}

// 构造函数
func NewWsClientManager() *WebsocketClientManager {
	defer func() {
		if r := recover(); r != nil {
			err := errors.New(fmt.Sprint(r))
			kits.WriteLog(err.Error())
		}
	}()
	var conn *websocket.Conn
	return &WebsocketClientManager{
		Addr:    &config.ServerAddr,
		Path:    config.WscPath,
		Conn:    conn,
		IsAlive: false,
		Timeout: 15,
	}
}

// 连接服务端
func (wsc *WebsocketClientManager) Dail() {
	var err error
	defer func() {
		if r := recover(); r != nil {
			err = errors.New(fmt.Sprint(r))
		}
	}()
	rand.Seed(time.Now().UnixNano())
	wsc.Addr = &config.ServerAddr
	u := url.URL{Scheme: "ws", Host: *wsc.Addr, Path: wsc.Path}
	wsc.Conn, _, err = websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		dailCount++
		if dailCount >= 3 {
			time.Sleep(time.Duration(dailCount) * time.Minute)
		}
		if dailCount >= 15 {
			dailCount = 0
		}
	} else {
		wsc.IsAlive = true
		config.WscAlive = true
		dailCount = 0
		kits.WriteLog("agent连接服务器成功")
	}
}

// 发送消息到服务端
func (wsc *WebsocketClientManager) sendMsgThread() {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				err := errors.New(fmt.Sprint(r))
				kits.WriteLog(err.Error())
				wsc.IsAlive = false
				config.WscAlive = false
			}
		}()
		for {
			if wsc.Conn != nil {
				rand.Seed(time.Now().UnixNano())
				msg := <-config.SendChan
				// websocket.TextMessage类型
				//kits.WriteLog("agent发送信息:" + msg)
				err := wsc.Conn.WriteMessage(websocket.TextMessage, []byte(msg))
				if err != nil {
					config.Qch <- "进程(" + strconv.Itoa(os.Getegid()) + ")发送消息失败:" + err.Error()
				} else {
					if dailCount >= 3 {
						err = wsc.Conn.WriteMessage(websocket.CloseMessage,
							websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
					}
				}
			} else {
				wsc.IsAlive = false
				config.WscAlive = false
				break
			}
		}
	}()
}

// 读取服务端消息
func (wsc *WebsocketClientManager) readMsgThread() {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				err := errors.New(fmt.Sprint(r))
				kits.WriteLog(err.Error())
				wsc.IsAlive = false
				config.WscAlive = false
			}
		}()
		for {
			if wsc.Conn != nil {
				_, message, err := wsc.Conn.ReadMessage()
				//kits.WriteLog("agent读取信息:" + string(message))
				if err != nil {
					config.Qch <- "进程(" + strconv.Itoa(os.Getegid()) + ")读取消息失败:" + err.Error()
				}
				config.RecvChan <- string(message)
				if dailCount >= 3 {
					err = wsc.Conn.WriteMessage(websocket.CloseMessage,
						websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
				}
			} else {
				wsc.IsAlive = false
				config.WscAlive = false
				break
			}
		}
	}()
}

func (wsc *WebsocketClientManager) Start() {
	for {
		func() {
			defer func() {
				if r := recover(); r != nil {
					err := errors.New(fmt.Sprint(r))
					kits.WriteLog(err.Error())
				}
			}()
			if wsc.IsAlive == false {
				wsc.Dail()
				wsc.sendMsgThread()
				wsc.readMsgThread()
			}
		}()
		time.Sleep(30 * time.Second)
	}
}
