package main

import (
	"encoding/json"
	"net/http"

	"github.com/cihub/seelog"
)

func sendHTTPJSON(w http.ResponseWriter, j interface{}) {
	data, err := json.Marshal(j)
	if nil != err {
		panic(err)
	}
	w.Write(data)
}

func startRPCServer(addr string) {
	if len(addr) == 0 {
		return
	}
	http.HandleFunc("/rpc/buyolshopitem", buyOlshopItemHandler)
	http.HandleFunc("/rpc/givereward", giveRewardHandler)

	seelog.Info("Start rpc server:", addr)
	go http.ListenAndServe(addr, nil)
}

type BuyOlshopItemResult struct {
	Result int
	Left   int
}

func buyOlshopItemHandler(w http.ResponseWriter, r *http.Request) {
	var result BuyOlshopItemResult
	result.Result = 1

	defer func() {
		sendHTTPJSON(w, &result)
	}()

	r.ParseForm()
	uid := getFormValueInt(r, "uid", 0)
	if uid <= 0 {
		return
	}
	cost := getFormValueInt(r, "cost", -1)
	if cost < 0 {
		return
	}
	itemId := getFormValueInt(r, "itemid", 0)
	if itemId <= 0 {
		return
	}
	name := getFormValueString(r, "name")
	if len(name) == 0 ||
		len(name) > 19 {
		return
	}

	ret, left := dbOnConsumeDonate(g_DBUser, uint32(uid), name, itemId, cost)
	if ret {
		result.Result = 0
		result.Left = left
	} else {
		result.Result = left
	}
}

func giveRewardHandler(w http.ResponseWriter, r *http.Request) {
}
