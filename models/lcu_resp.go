package models

import "time"

// 聊天组
type Conversation struct {
	GameName           string      `json:"gameName"`
	GameTag            string      `json:"gameTag"`
	Id                 string      `json:"id"`
	InviterId          string      `json:"inviterId"`
	IsMuted            bool        `json:"isMuted"`
	LastMessage        interface{} `json:"lastMessage"`
	Name               string      `json:"name"`
	Password           string      `json:"password"`
	Pid                string      `json:"pid"`
	TargetRegion       string      `json:"targetRegion"`
	Type               string      `json:"type"`
	UnreadMessageCount int         `json:"unreadMessageCount"`
}

//聊天信息
type LolMessage struct {
	Body           string    `json:"body"`
	FromID         string    `json:"fromId"`
	FromPid        string    `json:"fromPid"`
	FromSummonerID int64     `json:"fromSummonerId"`
	ID             string    `json:"id"`
	IsHistorical   bool      `json:"isHistorical"`
	Timestamp      time.Time `json:"timestamp"`
	Type           string    `json:"type"`
}

//召唤师信息
type SummonerInfo struct {
	AccountID                   int64  `json:"accountId"`
	DisplayName                 string `json:"displayName"`
	InternalName                string `json:"internalName"`
	NameChangeFlag              bool   `json:"nameChangeFlag"`
	PercentCompleteForNextLevel int    `json:"percentCompleteForNextLevel"`
	Privacy                     string `json:"privacy"`
	ProfileIconID               int    `json:"profileIconId"`
	Puuid                       string `json:"puuid"`
	RerollPoints                struct {
		CurrentPoints    int `json:"currentPoints"`
		MaxRolls         int `json:"maxRolls"`
		NumberOfRolls    int `json:"numberOfRolls"`
		PointsCostToRoll int `json:"pointsCostToRoll"`
		PointsToReroll   int `json:"pointsToReroll"`
	} `json:"rerollPoints"`
	SummonerID       int64 `json:"summonerId"`
	SummonerLevel    int   `json:"summonerLevel"`
	Unnamed          bool  `json:"unnamed"`
	XpSinceLastLevel int   `json:"xpSinceLastLevel"`
	XpUntilNextLevel int   `json:"xpUntilNextLevel"`
}

// 玩家分数
type UserScore struct {
	SummonerID   int64    `json:"summonerID"`
	SummonerName string   `json:"summonerName"`
	Score        float64  `json:"score"`
	AvgKDA       [][3]int `json:"currKDA"`
}

type CommonResp struct {
	ErrorCode  string `json:"errorCode"`
	HttpStatus int    `json:"httpStatus"`
	Message    string `json:"message"`
}
type GameListResp struct {
	CommonResp
	AccountID int64    `json:"accountId"`
	Games     GameList `json:"games"`
}
type GameList struct {
	GameBeginDate  string     `json:"gameBeginDate"`
	GameCount      int        `json:"gameCount"`
	GameEndDate    string     `json:"gameEndDate"`
	GameIndexBegin int        `json:"gameIndexBegin"`
	GameIndexEnd   int        `json:"gameIndexEnd"`
	Games          []GameInfo `json:"games"`
}
