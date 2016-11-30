package main

import (
	"encoding/json"
	//	"log"
	"time"

	"github.com/cihub/seelog"
)

type PlayerRankList struct {
	RankLevel   []UserRankInfo `json:"rank_level"`
	RankZhanShi []UserRankInfo `json:"rank_zhanshi"`
	RankFaShi   []UserRankInfo `json:"rank_fashi"`
	RankDaoShi  []UserRankInfo `json:"rank_daoshi"`
}

var g_RankListCache string = ""
var g_LastRankListCacheTime int64 = 0

type RankListCacheItem struct {
	data           string
	lastUpdateTime int64
}

var (
	gRankListCacheMap map[int]RankListCacheItem
)

func init() {
	gRankListCacheMap = make(map[int]RankListCacheItem)
}

func getPlayerRankList() string {
	if time.Now().Unix()-g_LastRankListCacheTime < 30 {
		return g_RankListCache
	}
	g_LastRankListCacheTime = time.Now().Unix()
	rankList := ""

	var rankData PlayerRankList
	rankData.RankLevel = dbGetUserRankInfoOrderByLevel(g_DBUser, 10, -1)
	rankData.RankZhanShi = dbGetUserRankInfoOrderByPower(g_DBUser, 10, 0)
	rankData.RankFaShi = dbGetUserRankInfoOrderByPower(g_DBUser, 10, 1)
	rankData.RankDaoShi = dbGetUserRankInfoOrderByPower(g_DBUser, 10, 2)

	jsBytes, err := json.Marshal(&rankData)
	if err != nil {
		seelog.Error("Err:Failed to marshal rank data.err:", err)
		return ""
	} else {
		rankList = string(jsBytes)
		g_RankListCache = rankList
		return rankList
	}
}

func getPlayerRankListV2(serverId int) string {
	cache, ok := gRankListCacheMap[serverId]
	if ok {
		if time.Now().Unix()-cache.lastUpdateTime < 30 {
			return cache.data
		}
	}

	nt := time.Now().Unix()
	rankList := ""

	var rankData PlayerRankList
	rankData.RankLevel = dbGetUserRankInfoOrderByLevelV2(g_DBUser, 10, -1)
	rankData.RankZhanShi = dbGetUserRankInfoOrderByPowerV2(g_DBUser, 10, 0)
	rankData.RankFaShi = dbGetUserRankInfoOrderByPowerV2(g_DBUser, 10, 1)
	rankData.RankDaoShi = dbGetUserRankInfoOrderByPowerV2(g_DBUser, 10, 2)

	jsBytes, err := json.Marshal(&rankData)
	if err != nil {
		seelog.Error("Err:Failed to marshal rank data.err:", err)
		return ""
	} else {
		rankList = string(jsBytes)
		var item RankListCacheItem
		item.data = rankList
		item.lastUpdateTime = nt
		gRankListCacheMap[serverId] = item
		return rankList
	}
}
