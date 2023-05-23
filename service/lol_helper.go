package service

import (
	"MyLOLassisitant/consts"
	"MyLOLassisitant/models"
	"MyLOLassisitant/myLogs"
	"MyLOLassisitant/utils"
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

type LolHelper struct {
	port  int
	token string

	client  *http.Client
	baseUrl string
}

func NewLolHelper() *LolHelper {
	return &LolHelper{}
}

func (l *LolHelper) Run() error {
	// 1.获取token,port
	err := l.initParams()
	if err != nil {
		myLogs.Println("fail to initParams,err:", err)
		return err
	}
	myLogs.Println("port,token after initParams :", l.port, l.token)

	// 2.连接lcu
	conn, err := l.Conn2LCU()
	if err != nil {
		myLogs.Println("fail to ConnLcu,err:", err)
		return err
	}

	// 3.接收lcu消息
	err = l.Start(conn)
	if err != nil {
		myLogs.Println("fail to MonitorGame,err:", err)
		return err
	}
	return nil
}

// 根据lol进程获取端口和token
func (l *LolHelper) initParams() error {
	var port int
	var token string
	var err error
	// 1.获取token,port信息
	err = utils.Retry(consts.TryTimes, 10*time.Millisecond, func() error {
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
	if err != nil {
		return err
	}
	// 2.LolHelper成员赋值
	l.token = token
	l.port = port
	l.client = &http.Client{
		Transport: &http.Transport{
			ForceAttemptHTTP2: true,                                  //使用http2.0
			TLSClientConfig:   &tls.Config{InsecureSkipVerify: true}, //不去验证服务端的数字证书
		},
	}
	l.baseUrl = fmt.Sprintf(consts.BaseUrlTokenPortTemplate, token, port)
	return nil
}

// Conn2LCU 连接到lcu
func (l *LolHelper) Conn2LCU() (*websocket.Conn, error) {
	//跳过证书检测
	dialer := websocket.DefaultDialer
	dialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	//构造header
	header := http.Header{}
	authToken := "Basic " + base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf(consts.WsTokenTemplate, l.token)))
	header.Set("Authorization", authToken)

	myLogs.Printf("success to toekn = %v, authToken = %v, port = %v\n", l.token, authToken, l.port)

	//构造WsUrl
	WsUrl, _ := url.Parse(fmt.Sprintf(consts.WsUrlTemplate, l.port))

	//连接到lcu
	conn, _, err := dialer.Dial(WsUrl.String(), header)
	if err != nil {
		myLogs.Println("fail to websocket连接到lcu失败, err = ", err)
		return nil, err
	}
	//lcu的规则,先发这条消息,后续才会发给我们消息
	conn.WriteMessage(websocket.TextMessage, []byte("[5, \"OnJsonApiEvent\"]"))
	return conn, nil
}

// Start 监听Lcu消息,下面就开始接收lcu发来的消息, 一切动作(开始匹配,回到大厅等消息都会发过来)
func (l *LolHelper) Start(conn *websocket.Conn) error {
	myLogs.Println("start receive lol msg\n")
	for {
		// 监听消息
		msgType, msg, err := conn.ReadMessage()
		if err != nil {
			myLogs.Println("fail to conn.ReadMessage(),websocket接收消息失败,err = ", err)
			return err
		}
		myLogs.Printf("success to get msg,websocket类型为: %v.\n找到websocket信息: %v\n", msgType, string(msg))
		if msgType != websocket.TextMessage || len(msg) < consts.OnJsonApiEventPrefixLen+1 {
			myLogs.Println("strange msg@@@@@@@@@@@@@@@@@@@@@")
			continue
		}

		wsMsg := &models.WsMsg{}
		// todo 此处待改进
		err = json.Unmarshal(msg[consts.OnJsonApiEventPrefixLen:len(msg)-1], wsMsg)
		if err != nil {
			myLogs.Printf("fail to json.Unmarshal(msg[consts.OnJsonApiEventPrefixLen:len(msg)-1], wsMsg),err:%v", err)
			continue
		}
		if wsMsg.Uri == consts.GameFlowChangedEvt { //这是切换时间
			myLogs.Printf("切换状态为,%v\n", wsMsg.Data)
			if wsMsg.Data.(string) == consts.GameStatusChampionSelect { //如果当前时间是匹配成功进入英雄选择界面
				myLogs.Println("进入英雄选择阶段,正在计算队友分数")
				err = l.ChampionSelectStart()
				if err != nil {
					myLogs.Println("fail to ChampionSelectStart,err:", err)
					return err
				}
			}
		}
	}
	return nil
}

//ChampionSelectStart 英雄选择阶段
func (l *LolHelper) ChampionSelectStart() error {
	time.Sleep(time.Second) //开始睡一会,因为服务端没那么快更新数据
	// 1、获取对战房间id
	roomId, err := l.GetRoomId()
	if err != nil {
		myLogs.Println("fail to GetRoomId,err:", err)
		return err
	}
	fmt.Println("roomId:\t", roomId)
	//2、根据房间id获取5个召唤师id
	summonerIds, err := l.GetSummonerInfoListByRoomId(roomId)
	myLogs.Println("召唤师名：", summonerIds)
	// 3.根据5个召唤师id,分别查出各自的召唤师信息存起来
	summonerInfos := make([]*models.SummonerInfo, 0)
	for _, summonerId := range summonerIds {
		summonerInfo, err := l.GetSummonerInfoById(summonerId)
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
			score, err := l.CalUserScoreById(id)
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
		err := l.SendConversationMsg(msg, roomId)
		if err != nil {
			myLogs.Printf("发送消息时出现错误,err = %v\n", err)
		}
	}
	return nil
}

type datatest struct { //发送消息时,服务端指定格式的数据
	Body string `json:"body"`
	Type string `json:"type"`
}

// SendConversationMsg 根据房间id发送消息
func (l *LolHelper) SendConversationMsg(msg interface{}, roomId string) error {
	TempByte, _ := json.Marshal(msg)
	data := datatest{
		Body: string(TempByte),
		Type: "chat",
	}
	fmt.Println("\n\n\n$$$$$$$$$$$$$$\n", data.Body)
	//json str 转map
	var dat map[string]interface{}
	if err := json.Unmarshal([]byte(data.Body), &dat); err == nil {
		fmt.Println("==============json str 转map=======================")
		fmt.Println(dat)
		fmt.Printf("type：%t\n", dat["currKDA"])
		//switch dat["currKDA"].(type) {
		//case string:
		//	fmt.Println(1111111)
		//case [][3]int:
		//	fmt.Println(2222222)
		//}
	} else {
		fmt.Println(err)
	}
	kda := strings.Split(data.Body, "currKDA\":")[1]
	kda = kda[:len(kda)-1]
	fmt.Println("kda:", kda)
	name := dat["summonerName"].(string)
	fmt.Println("name", name)
	score := dat["score"].(float64)
	fmt.Println("score", score)
	horseType := ""
	if score >= 10 {
		horseType = "上等马"
	} else if score >= 5 {
		horseType = "中等马"
	} else if score >= 3 {
		horseType = "下等马"
	} else {
		horseType = "牛马"
	}
	data.Body = horseType + ":" + name + ",近7场比赛KDA:" + strconv.FormatFloat(score, 'f', 3, 64) + "最近三把战绩为:" + kda
	if name == "皮皮九逗比小青年" || name == "Just随便一个名字" || name == "只有我干饭用盆么" {
		data.Body = "恭喜你匹配到了传说中的半人马（偶然能当个人）:+" + name + "，希望本场比赛他能够发光发热，不要搞事！半人马的最近三把战绩为：" + kda
		data.Body = "恭喜你匹配到了传说中的:+" + name + "他最近三把的战绩为：" + kda
	}
	if name == "第一把位" {
		if horseType == "下等马" || horseType == "牛马" {
			data.Body = "恭喜你匹配到了神秘的:+" + name + "这个人很神秘" + kda
		}
	}
	if name == "好机油鳄鱼" || name == "寒带火熊" {
		if horseType == "下等马" || horseType == "牛马" {
			data.Body = "匹配到的是作者本人:" + name + ",战况保密！"
		}
	}
	if name == "盈盈一水間" {
		if horseType == "下等马" || horseType == "牛马" {
			data.Body = "匹配到的是作者开黑的小伙伴:" + name + ",战况保密！"
		}
	}
	myLogs.Println("data.body:", string(data.Body))
	mess, err := l.PostReq(fmt.Sprintf("/lol-chat/v1/conversations/%s/messages", roomId), data)
	fmt.Println("响应为", string(mess))
	myLogs.Println("响应为", string(mess))
	return err
}

// 根据召唤师id计算得分
func (l *LolHelper) CalUserScoreById(summonerId int64) (models.UserScore, error) {
	userScoreInfo := &models.UserScore{
		SummonerID: summonerId,
		Score:      10,
	}
	// 获取用户信息
	summoner, err := l.GetSummonerInfoById(summonerId)
	if err != nil {
		return *userScoreInfo, err
	}
	userScoreInfo.SummonerName = summoner.DisplayName
	// 接下来只需要知道用户得分就行了.于是我们查询该召唤师的战绩列表
	gameList, err := l.listGameHistory(summonerId)
	if err != nil {
		log.Println("获取用户战绩失败,summonerId = %v, err = %v\n", summonerId, err)
		return *userScoreInfo, nil
	}

	for i, game := range gameList {
		if i < len(gameList)-3 {
			continue
		}
		var kda [3]int
		kda[0] = game.Participants[0].Stats.Kills
		kda[1] = game.Participants[0].Stats.Deaths
		kda[2] = game.Participants[0].Stats.Assists
		userScoreInfo.AvgKDA = append(userScoreInfo.AvgKDA, kda)
		if len(userScoreInfo.AvgKDA) == 5 {
			break
		}
	}
	userScoreInfo.Score = CalSocre(gameList)
	return *userScoreInfo, nil
}
func CalSocre(gameList []models.GameInfo) float64 {
	//var sum int64 = 0
	//for _,game := range gameList{
	//	sum = sum+game.GameId
	//}
	//return float64( sum )
	killNums, deathNums, assistNums := 0, 0, 0
	for _, game := range gameList {
		killNums += game.Participants[0].Stats.Kills
		deathNums += game.Participants[0].Stats.Deaths
		assistNums += game.Participants[0].Stats.Assists
	}

	return (float64(killNums) + float64(assistNums)) / float64(deathNums) * 3
}

func (l *LolHelper) listGameHistory(summonerId int64) ([]models.GameInfo, error) {
	fmtList := make([]models.GameInfo, 0, 7)
	//超过7把战绩就别存了
	resp, err := l.ListGamesBySummonerID(summonerId, 0, 7)
	//fmt.Printf("用户id为%v\n", summonerId )
	//fmt.Printf("用户战绩为%+v\n", resp )
	if err != nil {
		log.Printf("查询用户战绩失败,id = %v, err = %v\n", summonerId, err)
		return nil, err
	}
	for _, gameItem := range resp.Games.Games { //遍历每一局游戏
		if gameItem.QueueId != models.NormalQueueID &&
			gameItem.QueueId != models.RankSoleQueueID &&
			gameItem.QueueId != models.ARAMQueueID &&
			gameItem.QueueId != models.RankFlexQueueID {
			continue
		}
		fmtList = append(fmtList, gameItem)
	}
	return fmtList, nil
}

// ListGamesBySummonerID 根据召唤师id,查询最近[begin,begin+limit-1]的游戏战绩
func (l *LolHelper) ListGamesBySummonerID(summonerId int64, begin, limit int) (*models.GameListResp, error) {
	bts, err := l.GetReq(fmt.Sprintf("/lol-match-history/v3/matchlist/account/%d?begIndex=%d&endIndex=%d",
		summonerId, begin, begin+limit))
	if err != nil {
		return nil, err
	}
	data := &models.GameListResp{}
	json.Unmarshal(bts, data)
	return data, nil
}

// GetSummonerInfoById 根据召唤师id查找召唤师的完整信息
func (l *LolHelper) GetSummonerInfoById(id int64) (*models.SummonerInfo, error) {
	summonerInfo := make([]*models.SummonerInfo, 1)
	msg, err := l.GetReq(fmt.Sprintf(consts.GetSummonerInfoTemplate, id))
	if err != nil {
		myLogs.Println("fail to GetSummonerInfoById GetReq,err:", err)
		return nil, err
	}

	err = json.Unmarshal(msg, &summonerInfo)
	if err != nil {
		myLogs.Println("fail to GetSummonerInfoById unmarshal,err:", err)
		return nil, err
	}
	return summonerInfo[0], nil
}

// GetSummonerListByRoomId 通过聊天房间ID,查出聊天记录,从中找到5个召唤师的id值
func (l *LolHelper) GetSummonerInfoListByRoomId(roomId string) ([]int64, error) {
	msg, err := l.GetReq(fmt.Sprintf(consts.GetSummonerInfoByRoomIdTemplate, roomId)) //得到这个房间内的所有消息
	if err != nil {
		myLogs.Println("fail to getSummonerInfoListByRoomId getReq,err:", err)
		return nil, err
	}
	lolMsgs := make([]models.LolMessage, 10)
	err = json.Unmarshal(msg, &lolMsgs)
	if err != nil {
		myLogs.Println("fail to getSummonerInfoListByRoomId Unmarshal,err:", err)
		return nil, err
	}
	summonerIds := make([]int64, 0)
	for _, lolmsg := range lolMsgs {
		myLogs.Printf("房间消息为%v\n", lolmsg)
		if lolmsg.Type == "system" { //系统发出的消息,形如xxx进入房间
			summonerIds = append(summonerIds, lolmsg.FromSummonerID)
		}
	}
	return summonerIds, nil
}

// GetRoomId 获取当前对战聊天房间的ID
func (l *LolHelper) GetRoomId() (string, error) {
	msg, err := l.GetReq(consts.GetRoomId)
	if err != nil {
		myLogs.Println("fail to GetRoomId,err:", err)
		return "", err
	}
	myLogs.Println("getRoomId info:", string(msg))
	conversations := make([]models.Conversation, 10)
	err = json.Unmarshal(msg, &conversations)
	if err != nil {
		myLogs.Println("fail to unmarshal getRoomId,err:", err)
		return "", err
	}
	for _, conversation := range conversations {
		myLogs.Printf("conversation: %+v\n", conversation)
		if conversation.Type == "championSelect" {
			return conversation.Id, nil
		}
	}
	return "", nil
}

func (l *LolHelper) GetReq(url string) ([]byte, error) {
	fmt.Printf("http,get : cli+url = %v\n", l.baseUrl+url)
	myLogs.Printf("http,get : cli+url = %v\n", l.baseUrl+url)
	req, _ := http.NewRequest(http.MethodGet, l.baseUrl+url, nil)
	if req.Body != nil {
		req.Header.Add("ContentType", "application/json")
	}
	resp, err := l.client.Do(req)
	if err != nil {
		myLogs.Println("fail to req client.Do,err:", err)
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func (l *LolHelper) PostReq(url string, data interface{}) ([]byte, error) {
	fmt.Printf("http,post : cli+url = %v\n", l.baseUrl+url)
	fmt.Printf("data.body %v\n", data.(datatest).Body)
	fmt.Printf("data.type %v\n", data.(datatest).Type)
	myLogs.Printf("http,post : cli+url = %v\n", l.baseUrl+url)
	bts, err := json.Marshal(data) //转为二进制
	if err != nil {
		return nil, err
	}
	body := bytes.NewReader(bts)
	req, _ := http.NewRequest(http.MethodPost, l.baseUrl+url, body)
	if req.Body != nil {
		req.Header.Add("ContentType", "application/json")
	}
	resp, err := l.client.Do(req)
	if err != nil {
		myLogs.Println("fail to req client.Do,err:", err)
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}
