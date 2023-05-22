package main

import (
	"MyLOLassisitant/models"
	"encoding/json"
	"fmt"
)

type testMsg struct {
	a int
	b string
	c models.WsMsg
}

func main() {
	msg := `[8,"OnJsonApiEvent",{"data":{"availability":"chat","gameName":"","gameTag":"","icon":-1,"id":"2945136420","lastSeenOnlineTimestamp":null,"lol":{"challengeCrystalLevel":"SILVER","challengeTitleSelected":"81256cdb-5f7a-ef5b-141b-9caf49fa2345","challengeTokensSelected":"505006,505007,505000","championId":"","companionId":"18","damageSkinId":"1","gameId":"6571884820","gameMode":"ARAM","gameQueueType":"","gameStatus":"outOfGame","iconOverride":"summonerIcon","initSummoner":"0","isObservable":"ALL","mapId":"","mapSkinId":"1","queueId":"450","regalia":"{\"bannerType\":2,\"crestType\":1,\"selectedPrestigeCrest\":1}","skinVariant":"","skinname":"","timeStamp":"1662992732407"},"name":"好机油鳄鱼","patchline":"","pid":"","platformId":"","product":"league_of_legends","productName":"","puuid":"","statusMessage":"再玩腕豪我是🐶","summary":"","summonerId":2945136420,"time":0},"eventType":"Update","uri":"/lol-chat/v1/conversations/4017184151/participants/2945136420"}]`
	t := make([]interface{}, 0)
	err := json.Unmarshal([]byte(msg), &t)
	if err != nil {
		fmt.Println(err)
	}
	//for k, v := range t {
	//	fmt.Printf("%v,%T,%v\n\n", k, v, v)
	//
	//}
	x, _ := json.Marshal(t[2])
	fmt.Println(string(x))
	wx := &models.WsMsg{}
	json.Unmarshal(x, wx)
	fmt.Println(wx.Uri)

}
