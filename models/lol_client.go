package models

import (
	"MyLOLassisitant/consts"
	"MyLOLassisitant/myLogs"
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
)

type LolClient struct {
	client  *http.Client
	baseUrl string
}

func NewLolCilent(port int, token string) *LolClient {
	return &LolClient{
		client: &http.Client{
			Transport: &http.Transport{
				ForceAttemptHTTP2: true,                                  //使用http2.0
				TLSClientConfig:   &tls.Config{InsecureSkipVerify: true}, //不去验证服务端的数字证书
			},
		},
		baseUrl: fmt.Sprintf(consts.BaseUrlTokenPortTemplate, token, port),
	}
}

type datatest struct { //发送消息时,服务端指定格式的数据
	Body string `json:"body"`
	Type string `json:"type"`
}

// SendConversationMsg 根据房间id发送消息
func (l *LolClient) SendConversationMsg(msg interface{}, roomId string) error {
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
func (l *LolClient) CalUserScoreById(summonerId int64) (UserScore, error) {
	userScoreInfo := &UserScore{
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
func CalSocre(gameList []GameInfo) float64 {
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

func (l *LolClient) listGameHistory(summonerId int64) ([]GameInfo, error) {
	fmtList := make([]GameInfo, 0, 7)
	//超过7把战绩就别存了
	resp, err := l.ListGamesBySummonerID(summonerId, 0, 7)
	//fmt.Printf("用户id为%v\n", summonerId )
	//fmt.Printf("用户战绩为%+v\n", resp )
	if err != nil {
		log.Printf("查询用户战绩失败,id = %v, err = %v\n", summonerId, err)
		return nil, err
	}
	for _, gameItem := range resp.Games.Games { //遍历每一局游戏
		if gameItem.QueueId != NormalQueueID &&
			gameItem.QueueId != RankSoleQueueID &&
			gameItem.QueueId != ARAMQueueID &&
			gameItem.QueueId != RankFlexQueueID {
			continue
		}
		fmtList = append(fmtList, gameItem)
	}
	return fmtList, nil
}

// ListGamesBySummonerID 根据召唤师id,查询最近[begin,begin+limit-1]的游戏战绩
func (l *LolClient) ListGamesBySummonerID(summonerId int64, begin, limit int) (*GameListResp, error) {
	bts, err := l.GetReq(fmt.Sprintf("/lol-match-history/v3/matchlist/account/%d?begIndex=%d&endIndex=%d",
		summonerId, begin, begin+limit))
	if err != nil {
		return nil, err
	}
	data := &GameListResp{}
	json.Unmarshal(bts, data)
	return data, nil
}

// GetSummonerInfoById 根据召唤师id查找召唤师的完整信息
func (l *LolClient) GetSummonerInfoById(id int64) (*SummonerInfo, error) {
	summonerInfo := make([]*SummonerInfo, 1)
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
func (l *LolClient) GetSummonerInfoListByRoomId(roomId string) ([]int64, error) {
	msg, err := l.GetReq(fmt.Sprintf(consts.GetSummonerInfoByRoomIdTemplate, roomId)) //得到这个房间内的所有消息
	if err != nil {
		myLogs.Println("fail to getSummonerInfoListByRoomId getReq,err:", err)
		return nil, err
	}
	lolMsgs := make([]LolMessage, 10)
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
func (l *LolClient) GetRoomId() (string, error) {
	msg, err := l.GetReq(consts.GetRoomId)
	if err != nil {
		myLogs.Println("fail to GetRoomId,err:", err)
		return "", err
	}
	myLogs.Println("getRoomId info:", string(msg))
	conversations := make([]Conversation, 10)
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

func (l *LolClient) GetReq(url string) ([]byte, error) {
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

func (l *LolClient) PostReq(url string, data interface{}) ([]byte, error) {
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
