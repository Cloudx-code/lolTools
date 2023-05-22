package consts

const (
	//解析url时的前缀固定长度 todo
	OnJsonApiEventPrefixLen = len(`[8,"OnJsonApiEvent",`)
	//游戏内切换状态
	GameFlowChangedEvt = "/lol-gameflow/v1/gameflow-phase"
	//后续的请求都要在此路径后添加	%s:Token,	%d:Port
	BaseUrlTokenPortTemplate = "https://riot:%s@127.0.0.1:%d"
	//wsUrl模板:端口号Port
	WsUrlTemplate = "wss://127.0.0.1:%d/"
	//wsToken模板:Token
	WsTokenTemplate = "riot:%s"
)

//lcu的url相关
const (
	GetRoomId                       = "/lol-chat/v1/conversations"
	GetSummonerInfoByRoomIdTemplate = GetRoomId + "/%v/messages"

	GetSummonerInfoTemplate = "/lol-summoner/v2/summoners?ids=[%v]"
)

//模式选择
const (
	GameStatusChampionSelect = "ChampSelect"
)
