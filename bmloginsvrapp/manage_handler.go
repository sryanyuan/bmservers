package main

import (
	"encoding/json"
)

var routerManagementItems = []RouterItem{
	{"/management/index", kPermission_SuperAdmin, webManagementIndexHandler},
	{"/management/user", kPermission_SuperAdmin, webManagementUserHandler},
	{"/management/finduser", kPermission_SuperAdmin, ajaxManagementFindUser},
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

//	user
var webManagementUserTpls = []string{
	"template/management/manage_nav.html",
	"template/management/user.html",
}

func webManagementUserHandler(ctx *RequestContext) {
	data := renderTemplate(ctx, webManagementUserTpls, nil)
	ctx.w.Write(data)
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
			resultSet := make([]*ExportUserAccountInfo, 1, 1)
			resultSet[0] = eAi
			data, _ := json.Marshal(resultSet)
			result.Msg = string(data)
			return
		}
	} else if len(name) != 0 {
		users, err := dbGetUserAccountInfoByName(g_DBUser, name)
		if nil != err {
			result.Msg = err.Error()
			return
		}
		data, _ := json.Marshal(users)
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
