package main

import (
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/sryanyuan/bmservers/protocol"
)

//	timer player rank
var g_lastSendPlayerTime int64 = 0

func UpdateTimerEvent() {
	nowTime := time.Now().Unix()

	if nowTime-g_lastSendPlayerTime > 30 {
		g_lastSendPlayerTime = nowTime
		//	update player rank
		if nil != g_DBUser {
			var rankNtf protocol.MSyncPlayerRankNtf
			for _, gs := range serverNodeMap {
				//rankNtf.Data = proto.String(getPlayerRankListV2(gs.serverId))
				rankNtf.Data = proto.String(getPlayerRankList())
				sendProto(gs.conn, uint32(protocol.LSOp_SyncPlayerRankNtf), &rankNtf)
			}
		}
	}
}
