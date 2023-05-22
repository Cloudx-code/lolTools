package service

/*
@Author:xy
@Desc: 主流程
*/

import (
	"MyLOLassisitant/consts"
	"MyLOLassisitant/models"
	"MyLOLassisitant/myLogs"
	"MyLOLassisitant/utils"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"sync"
	"time"
)

// ProphetInterface LOL助手接口
type ProphetInterface interface {
	// 获取port及token
}

type ProphetService struct {
	Port      int
	Token     string
	LolClient *models.LolClient
}

func NewProphet() *ProphetService {
	return &ProphetService{}
}

func (p *ProphetService) Run() error {
	//1.获取token,port
	port, token, err := p.getPortToken()
	if err != nil {
		myLogs.Println("fail to getPortToken,err:", err)
		return err
	}
	myLogs.Println("port,token after getPortToken :", port, token)

	lolClient := NewLolCilent(port, token)
	//2.连接lcu
	conn, err := lolClient.Conn2LCU()
	if err != nil {
		myLogs.Println("fail to ConnLcu,err:", err)
		return err
	}
	//3)接收lcu消息
	err = lolClient.Run(conn)
	if err != nil {
		myLogs.Println("fail to MonitorGame,err:", err)
		return err
	}
	return nil
}

// 根据lol进程获取端口和token
func (p *ProphetService) getPortToken() (port int, token string, err error) {
	utils.Retry(consts.TryTimes, 10*time.Millisecond, func() error {
		port, token, err = GetLolClientPortToken()
		if err != nil {
			if !errors.Is(consts.ErrLolProcessNotFound, err) {
				//如果不是因为没有找到lol进程的错误,那就记录下来
				myLogs.Println("fail to GetLolClientPortToken,获取port token 失败,err:", err)
			}
			myLogs.Println("获取port，token，尝试失败")
			return err
		}
		return nil
	})
	return
}

//这里的逻辑要好好想下
func (p *ProphetService) LolMonitor(conn *websocket.Conn) error {
	//下面就开始接收lcu发来的消息, 一切动作(开始匹配,回到大厅等消息都会发过来)
	myLogs.Println("start receive lol msg\n")
	for {
		msgType, msg, err := conn.ReadMessage()
		if err != nil {
			myLogs.Println("fail to conn.ReadMessage(),websocket接收消息失败,err = ", err)
			return err
		}
		myLogs.Printf("找到websocket类型为: %v\n找到websocket信息: %v\n", msgType, string(msg))
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
				err = p.ChampionSelectStart()
				if err != nil {
					myLogs.Println("fail to ChampionSelectStart,err:", err)
					return err
				}
			}
		}
	}
	return nil
}

//英雄选择阶段
func (p *ProphetService) ChampionSelectStart() error {
	time.Sleep(time.Second) //开始睡一会,因为服务端没那么快更新数据
	// 1、获取对战房间id
	roomId, err := p.LolClient.GetRoomId()
	if err != nil {
		myLogs.Println("fail to GetRoomId,err:", err)
		return err
	}
	fmt.Println("roomId:\t", roomId)
	//2、根据房间id获取5个召唤师id
	summonerIds, err := p.LolClient.GetSummonerInfoListByRoomId(roomId)
	myLogs.Println("召唤师名：", summonerIds)
	// 3.根据5个召唤师id,分别查出各自的召唤师信息存起来
	summonerInfos := make([]*models.SummonerInfo, 0)
	for _, summonerId := range summonerIds {
		summonerInfo, err := p.LolClient.GetSummonerInfoById(summonerId)
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
			score, err := p.LolClient.CalUserScoreById(id)
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
		err := p.LolClient.SendConversationMsg(msg, roomId)
		if err != nil {
			myLogs.Printf("发送消息时出现错误,err = %v\n", err)
		}
	}
	return nil
}
