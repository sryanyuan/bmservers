package main

import (
	"encoding/json"

	"github.com/cihub/seelog"
)

var routerManagementItems = []RouterItem{
	{"/management/index", kPermission_SuperAdmin, webManagementIndexHandler},
	{"/management/user", kPermission_SuperAdmin, webManagementUserHandler},
	{"/management/finduser", kPermission_SuperAdmin, ajaxManagementFindUser},
	{"/management/adddonate", kPermission_SuperAdmin, webManagementAddDonateHandler},
	{"/management/doadddonate", kPermission_SuperAdmin, ajaxManagementAddDonate},
	{"/management/viewdonate", kPermission_SuperAdmin, webManagementViewDonateHandler},
}

//	index
var webManagementIndexTpls = []string{
	"template/management/manage_nav.html",
	"template/management/index.html",
}

func webManagementIndexHandler(ctx *RequestContext) {
	data := renderTemplate(ctx, webManagementIndexTpls, nil)
	ctx.w.Write(data)
}

//	view donate
var webManagementViewDonateTpls = []string{
	"template/management/manage_nav.html",
	"template/management/viewdonate.html",
}

func webManagementViewDonateHandler(ctx *RequestContext) {
	ctx.r.ParseForm()
	uid := getFormValueInt(ctx.r, "uid", 0)
	renderData := make(map[string]interface{})
	renderData["UID"] = uid

	var donateHistoryList []*UserDonateHistoryExpose
	if 0 == uid {
		donateHistoryList = make([]*UserDonateHistoryExpose, 0, 1)
	} else {
		var err error
		donateHistoryList, err = dbGetUserDonateHistoryList(g_DBUser, uint32(uid))
		if nil != err {
			seelog.Error(err)
		}
	}
	renderData["DonateList"] = donateHistoryList

	data := renderTemplate(ctx, webManagementViewDonateTpls, renderData)
	ctx.w.Write(data)
}

//	adddonate
var webManagementAddDonateTpls = []string{
	"template/management/manage_nav.html",
	"template/management/adddonate.html",
}

func webManagementAddDonateHandler(ctx *RequestContext) {
	ctx.r.ParseForm()
	uid := getFormValueInt(ctx.r, "uid", 0)
	renderData := make(map[string]interface{})
	renderData["UID"] = uid
	data := renderTemplate(ctx, webManagementAddDonateTpls, renderData)
	ctx.w.Write(data)
}

//	user
var webManagementUserTpls = []string{
	"template/management/manage_nav.html",
	"template/management/user.html",
}

func webManagementUserHandler(ctx *RequestContext) {
	data := renderTemplate(ctx, webManagementUserTpls, nil)
	ctx.w.Write(data)
}

func ajaxManagementAddDonate(ctx *RequestContext) {
	result := SignUpResult{
		Result: 1,
		Msg:    "No operation",
	}
	defer func() {
		ctx.RenderJson(&result)
	}()

	if ctx.r.Method != "POST" {
		result.Msg = "Invalid method"
		return
	}

	ctx.r.ParseForm()
	uid := getFormValueInt(ctx.r, "user[uid]", 0)
	if 0 == uid {
		result.Msg = "Invalid uid"
		return
	}
	orderId := getFormValueString(ctx.r, "user[orderid]")
	if 0 == len(orderId) {
		result.Msg = "Invalid orderid"
		return
	}
	donateCount := getFormValueInt(ctx.r, "user[donate]", 0)
	if 0 == donateCount {
		result.Msg = "Invalid donate"
		return
	}

	//	get user
	var donateUser UserAccountInfo
	findUser := dbGetUserAccountInfoByUID(g_DBUser, uint32(uid), &donateUser)
	if !findUser {
		result.Msg = "Can't find user"
		return
	}

	//	insert a record
	if err := dbIncUserDonateInfoEx(g_DBUser, uint32(uid), donateCount, orderId); nil != err {
		result.Msg = err.Error()
		return
	}

	result.Result = 0

	//	push tcp event
}

func ajaxManagementFindUser(ctx *RequestContext) {
	result := SignUpResult{
		Result: 1,
		Msg:    "No operation",
	}
	defer func() {
		ctx.RenderJson(&result)
	}()

	if ctx.r.Method != "POST" {
		result.Msg = "Invalid method"
		return
	}

	ctx.r.ParseForm()
	uid := getFormValueInt(ctx.r, "user[uid]", 0)
	name := getFormValueString(ctx.r, "user[name]")
	account := getFormValueString(ctx.r, "user[account]")

	if 0 == uid &&
		len(name) == 0 &&
		len(account) == 0 {
		return
	}

	if 0 != uid {
		var accountInfo UserAccountInfo
		if !dbGetUserAccountInfoByUID(g_DBUser, uint32(uid), &accountInfo) {
			result.Msg = "找不到该用户"
			return
		} else {
			result.Result = 0
			eAi := &ExportUserAccountInfo{
				Account:  accountInfo.account,
				Uid:      accountInfo.uid,
				Password: accountInfo.password,
				Mail:     accountInfo.mail,
			}
			data, _ := json.Marshal(eAi)
			result.Msg = string(data)
			return
		}
	} else if len(name) != 0 {
		var userInfo ExportUserAccountInfo
		err := dbGetUserAccountInfoByName(g_DBUser, name, &userInfo)
		if nil != err {
			result.Msg = err.Error()
			return
		}
		result.Result = 0
		data, _ := json.Marshal(&userInfo)
		result.Msg = string(data)
		return
	} else if len(account) != 0 {
		var accountInfo UserAccountInfo
		ok, err := dbGetUserAccountInfo(g_DBUser, account, &accountInfo)
		if !ok ||
			nil != err {
			result.Msg = "找不到该用户"
			if nil != err {
				result.Msg = err.Error()
			}
			return
		} else {
			result.Result = 0
			eAi := &ExportUserAccountInfo{
				Account:  accountInfo.account,
				Uid:      accountInfo.uid,
				Password: accountInfo.password,
				Mail:     accountInfo.mail,
			}
			data, _ := json.Marshal(eAi)
			result.Msg = string(data)
			return
		}
	}
}
