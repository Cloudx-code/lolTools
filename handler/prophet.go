package handler

import (
	"MyLOLassisitant/consts"
	"MyLOLassisitant/models"
	"MyLOLassisitant/myLogs"
	"MyLOLassisitant/service"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"
)

type Prophet struct {
	Port      int
	Token     string
	LolClient *models.LolClient
}

func NewProphet() *Prophet {
	return &Prophet{}
}

func (p *Prophet) Run() error {
	//1)初始化参数，获取token,port
	err := p.InitParams()
	if err != nil {
		myLogs.Println("fail to initParams,err:", err)
		return err
	}
	myLogs.Println("port,token after initParams :", p.Port, p.Token)
	//2)连接lcu
	conn, err := p.ConnLcu()
	if err != nil {
		myLogs.Println("fail to ConnLcu,err:", err)
		return err
	}
	//3)接收消息
	err = p.LolMonitor(conn)
	if err != nil {
		myLogs.Println("fail to MonitorGame,err:", err)
		return err
	}
	//todo 待删
	time.Sleep(time.Second)
	return nil
}

func (c *Prophet) ConnLcu() (*websocket.Conn, error) {
	//跳过证书检测
	dialer := websocket.DefaultDialer
	dialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	//构造header
	header := http.Header{}
	authToken := "Basic " + base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf(consts.WsTokenTemplate, c.Token)))
	myLogs.Printf("toekn = %v, authToken = %v, port = %v\n", c.Token, authToken, c.Port)
	header.Set("Authorization", authToken)
	//构造WsUrl
	urlStruct, _ := url.Parse(fmt.Sprintf(consts.WsUrlTemplate, c.Port))
	WsUrl := urlStruct.String()
	//连接到lcu
	conn, _, err := dialer.Dial(WsUrl, header)
	if err != nil {
		myLogs.Println("websocket连接到lcu失败, err = ", err)
		return nil, err
	}
	conn.WriteMessage(websocket.TextMessage, []byte("[5, \"OnJsonApiEvent\"]")) //lcu的规则,先发这条消息,后续才会发给我们消息
	return conn, nil
}

//这里的逻辑要好好想下
func (c *Prophet) LolMonitor(conn *websocket.Conn) error {
	//下面就开始接收lcu发来的消息, 一切动作(开始匹配,回到大厅等消息都会发过来)
	myLogs.Println("start receive lol msg\n")
	for {
		msgType, msg, err := conn.ReadMessage()
		if err != nil {
			myLogs.Println("websocket接收消息失败,err = ", err)
			return err
		}
		myLogs.Printf("找到websocket类型为: %v\n", msgType)
		myLogs.Printf("找到websocket信息: %v\n", string(msg))
		if msgType != websocket.TextMessage || len(msg) < consts.OnJsonApiEventPrefixLen+1 {
			myLogs.Println("strange msg@@@@@@@@@@@@@@@@@@@@@")
			continue
		}
		wsMsg := &models.WsMsg{}
		json.Unmarshal(msg[consts.OnJsonApiEventPrefixLen:len(msg)-1], wsMsg)
		if wsMsg.Uri == consts.GameFlowChangedEvt { //这是切换时间
			myLogs.Printf("切换状态为,%v\n", wsMsg.Data)
			if wsMsg.Data.(string) == consts.GameStatusChampionSelect { //如果当前时间是匹配成功进入英雄选择界面
				myLogs.Println("进入英雄选择阶段,正在计算队友分数")
				err = c.ChampionSelectStart()
				if err != nil {
					myLogs.Println("fail to ChampionSelectStart,err:", err)
					return err
				}
			}
		}
	}
	return nil
}

//根据lol进程获取端口和token
func (c *Prophet) InitParams() error {
	var port int
	var token string
	var err error
	for i := 0; i < consts.TryTimes; i++ {
		port, token, err = service.GetLolClientPortToken()
		if err != nil {
			if !errors.Is(consts.ErrLolProcessNotFound, err) {
				//如果不是因为没有找到lol进程的错误,那就记录下来
				myLogs.Println("fail to GetLolClientPortToken,获取lcu info 失败,err:", err)
				return err
			}
			myLogs.Println("获取port，token，第", i, "次尝试")
			time.Sleep(time.Second)
			continue
		}
		break
	}
	c.Port = port
	c.Token = token
	c.LolClient = models.NewLolCilent(port, token)
	return nil
}

//英雄选择阶段
func (c *Prophet) ChampionSelectStart() error {
	time.Sleep(time.Second) //开始睡一会,因为服务端没那么快更新数据
	// 1、获取对战房间id
	roomId, err := c.LolClient.GetRoomId()
	if err != nil {
		myLogs.Println("fail to GetRoomId,err:", err)
		return err
	}
	fmt.Println("roomId:\t", roomId)
	//2、根据房间id获取5个召唤师id
	summonerIds, err := c.LolClient.GetSummonerInfoListByRoomId(roomId)
	myLogs.Println("召唤师名：", summonerIds)
	// 3.根据5个召唤师id,分别查出各自的召唤师信息存起来
	summonerInfos := make([]*models.SummonerInfo, 0)
	for _, summonerId := range summonerIds {
		summonerInfo, err := c.LolClient.GetSummonerInfoById(summonerId)
		if err != nil {
			myLogs.Printf("fail to GetSummonerInfoById,id = %v的用户信息查询失败, err = %v\n", summonerId, err)
			continue
		}
		summonerInfos = append(summonerInfos, summonerInfo)
		myLogs.Printf("召唤师信息: %v\n", *summonerInfo)
	}
	// 4.根据5个召唤师信息,去计算各自的得分
	userScoreMap := map[int64]models.UserScore{}
	mu := sync.Mutex{}
	var wg = &sync.WaitGroup{}
	for _, summonerId := range summonerIds {
		wg.Add(1)
		go func(id int64) {
			defer wg.Done()
			score, err := c.LolClient.CalUserScoreById(id)
			if err != nil {
				log.Printf("查询用户%v分数出错\n", id)
				return
			}
			mu.Lock()
			userScoreMap[id] = score
			mu.Unlock()
		}(summonerId)
	}
	wg.Wait()
	// 6.把计算好的得分发送到聊天框内
	for _, msg := range userScoreMap {
		err := c.LolClient.SendConversationMsg(msg, roomId)
		if err != nil {
			myLogs.Printf("发送消息时出现错误,err = %v\n", err)
		}
	}
	return nil
}
