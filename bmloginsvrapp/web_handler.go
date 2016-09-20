package main

import (
	"net/http"
	"time"

	"crypto/md5"

	"encoding/hex"

	"regexp"

	"strings"

	"github.com/cihub/seelog"
	"github.com/dchest/captcha"
	"github.com/gorilla/mux"
	"github.com/sryanyuan/tcpnetwork"
)

func startWebServer(addr string) {
	if len(addr) == 0 {
		return
	}

	seelog.Info("Start web server:", addr)

	//	initialize routers
	r := mux.NewRouter()
	httpMux := http.NewServeMux()
	InitRouters(httpMux, r)
	httpMux.Handle("/", r)

	//	run the server
	go http.ListenAndServe(addr, httpMux)
}

var webIndexTpls = []string{
	"template/home.html",
}

func webIndexHandler(ctx *RequestContext) {
	data := renderTemplate(ctx, webIndexTpls, nil)
	ctx.w.Write(data)
}

//	sign in
type SignInResult struct {
	Result    int
	Msg       string
	CaptchaId string
}

var webSigninTpls = []string{
	"template/account/signin.html",
}

func webSigninHandler(ctx *RequestContext) {
	tplData := make(map[string]interface{})
	if ctx.r.Method == "GET" {
		tplData["captchaid"] = captcha.NewLen(4)
		data := renderTemplate(ctx, webSigninTpls, tplData)
		ctx.w.Write(data)
	} else {
		ctx.r.ParseForm()

		var result = SignInResult{
			Result: 1,
		}

		username := ctx.r.Form.Get("user[login]")
		password := ctx.r.Form.Get("user[password]")
		rememberMe := "0"
		url := ctx.r.Form.Get("url")
		if len(url) == 0 {
			url = "/"
		}

		//	check
		failedMsg := ""

		for {
			if !captcha.VerifyString(ctx.r.Form.Get("captchaid"), ctx.r.Form.Get("captchaSolution")) {
				failedMsg = "验证码错误"
				break
			}

			if len(username) == 0 {
				failedMsg = "用户名不能为空"
				break
			}
			if len(password) == 0 {
				failedMsg = "密码不能为空"
				break
			}
			if len(password) > 20 {
				failedMsg = "密码太长"
				break
			}

			// get user from db
			user, err := dbWebUserGet(g_DBUser, username)
			if nil != err {
				failedMsg = err.Error()
				break
			}
			if nil == user {
				failedMsg = "用户名不存在"
				break
			}
			md5calc := md5.New()
			md5calc.Write([]byte(password))
			md5Psw := hex.EncodeToString(md5calc.Sum(nil))
			if md5Psw != user.Password {
				failedMsg = "密码错误"
				break
			}

			//	now ok
			if "0" != rememberMe {
				ctx.SaveWebUser(user, 5)
			} else {
				ctx.SaveWebUser(user, 0)
			}
			break
		}

		if 0 != len(failedMsg) {
			result.CaptchaId = captcha.NewLen(4)
			result.Msg = failedMsg
			ctx.RenderJson(&result)
		} else {
			//	login ok
			result.Msg = url
			result.Result = 0
			ctx.RenderJson(&result)
			seelog.Debug("User ", username, " login success")
		}
	}
}

//	sign out
func webSignOutHandler(ctx *RequestContext) {
	//	already login
	if ctx.user.Uid == 0 {
		ctx.Redirect("/", http.StatusFound)
		return
	}

	//	clear user in session
	ctx.ClearWebUser()
	ctx.Redirect("/signin", http.StatusFound)
}

//	sign up
type SignUpResult struct {
	Result    int    `json:Result`
	Msg       string `json:Msg`
	CaptchaId string `json:CaptchaId`
}

var signupRenderTpls = []string{
	"template/account/signup.html",
}

func webRegAccountHandler(ctx *RequestContext) {
	tplData := make(map[string]interface{})

	//	render signup page
	if ctx.r.Method == "GET" {
		tplData["captchaid"] = captcha.NewLen(4)
		data := renderTemplate(ctx, signupRenderTpls, tplData)
		ctx.w.Write(data)
	} else if ctx.r.Method == "POST" {
		//	post register message
		ctx.r.ParseForm()
		failedMsg := ""
		userName := ctx.r.Form.Get("user[login]")
		password := ctx.r.Form.Get("user[password]")
		email := ctx.r.Form.Get("user[email]")

		//	validate input
		for {
			if !captcha.VerifyString(ctx.r.Form.Get("captchaid"), ctx.r.Form.Get("captchaSolution")) {
				failedMsg = "验证码错误"
				break
			}
			if matched, _ := regexp.Match("^[a-zA-Z0-9_]{5,20}$", []byte(userName)); !matched {
				failedMsg = "非法的用户名"
				break
			}
			if strings.ToLower(userName) == "guest" {
				failedMsg = "非法的用户名"
				break
			}

			if matched, _ := regexp.Match("^\\s*\\w+(?:\\.{0,1}[\\w-]+)*@[a-zA-Z0-9]+(?:[-.][a-zA-Z0-9]+)*\\.[a-zA-Z]+\\s*$", []byte(email)); !matched {
				failedMsg = "非法的邮件地址"
				break
			}
			if matched, _ := regexp.Match("^[0-9a-zA-Z~!@$#%^]{5,20}$", []byte(password)); matched {
				if ctx.r.Form.Get("user[password_confirm]") != password {
					failedMsg = "两次输入密码不相同"
					break
				}
			} else {
				failedMsg = "非法的密码"
				break
			}

			//	already exists?
			if userExists := dbUserAccountExist(g_DBUser, userName); userExists {
				failedMsg = "用户名已存在"
				break
			}
			break
		}

		signUpResult := SignUpResult{
			Result: 1,
			Msg:    failedMsg,
		}
		if len(failedMsg) != 0 {
			//	echo error message
			signUpResult.CaptchaId = captcha.NewLen(4)
			renderJson(ctx, &signUpResult)
			return
		}

		//	new user
		var newuser UserAccountInfo
		newuser.account = userName
		newuser.password = password
		newuser.mail = email
		newuserList := make([]UserAccountInfo, 1)
		newuserList[0] = newuser

		if !dbInsertUserAccountInfo(g_DBUser, newuserList) {
			signUpResult.CaptchaId = captcha.NewLen(4)
			signUpResult.Msg = "Internal server error"
			renderJson(ctx, &signUpResult)
			return
		}

		//	all ok, redirect to signin page
		signUpResult.Result = 0
		signUpResult.Msg = "/account/signupsuccess?account=" + userName
		renderJson(ctx, &signUpResult)
	} else {
		http.Redirect(ctx.w, ctx.r, "/", http.StatusFound)
	}
}

var signupSuccessRenderTpls = []string{
	"template/account/signupsuccess.html",
}

func signupSuccessHandler(ctx *RequestContext) {
	ctx.r.ParseForm()
	tplData := make(map[string]interface{})
	username := ctx.r.Form.Get("account")
	tplData["account"] = username
	data := renderTemplate(ctx, signupSuccessRenderTpls, tplData)
	ctx.w.Write(data)
}

//	modify password
var modifyPasswordRenderTpls = []string{
	"template/account/modifypassword.html",
}

func webModifyPasswordHandler(ctx *RequestContext) {
	tplData := make(map[string]interface{})

	//	render modify page
	if ctx.r.Method == "GET" {
		tplData["captchaid"] = captcha.NewLen(4)
		data := renderTemplate(ctx, modifyPasswordRenderTpls, tplData)
		ctx.w.Write(data)
	} else if ctx.r.Method == "POST" {
		//	post register message
		ctx.r.ParseForm()
		failedMsg := ""
		userName := ctx.r.Form.Get("user[login]")
		password := ctx.r.Form.Get("user[password]")
		email := ctx.r.Form.Get("user[email]")

		//	validate input
		for {
			if !captcha.VerifyString(ctx.r.Form.Get("captchaid"), ctx.r.Form.Get("captchaSolution")) {
				failedMsg = "验证码错误"
				break
			}
			if matched, _ := regexp.Match("^[a-zA-Z0-9_]{5,20}$", []byte(userName)); !matched {
				failedMsg = "非法的用户名"
				break
			}
			if strings.ToLower(userName) == "guest" {
				failedMsg = "非法的用户名"
				break
			}

			if matched, _ := regexp.Match("^\\s*\\w+(?:\\.{0,1}[\\w-]+)*@[a-zA-Z0-9]+(?:[-.][a-zA-Z0-9]+)*\\.[a-zA-Z]+\\s*$", []byte(email)); !matched {
				failedMsg = "非法的邮件地址"
				break
			}
			if matched, _ := regexp.Match("^[0-9a-zA-Z~!@$#%^]{5,20}$", []byte(password)); matched {
				if ctx.r.Form.Get("user[password_confirm]") != password {
					failedMsg = "两次输入密码不相同"
					break
				}
			} else {
				failedMsg = "非法的密码"
				break
			}

			break
		}

		signUpResult := SignUpResult{
			Result: 1,
			Msg:    failedMsg,
		}
		if len(failedMsg) != 0 {
			//	echo error message
			signUpResult.CaptchaId = captcha.NewLen(4)
			renderJson(ctx, &signUpResult)
			return
		}

		//	new user
		var account UserAccountInfo
		got, err := dbGetUserAccountInfo(g_DBUser, userName, &account)
		if !got ||
			nil != err {
			signUpResult.CaptchaId = captcha.NewLen(4)
			if nil != err {
				signUpResult.Msg = err.Error()
			} else {
				signUpResult.Msg = "不存在的用户"
			}
			renderJson(ctx, &signUpResult)
			return
		}
		//	check email
		if account.mail != email {
			signUpResult.CaptchaId = captcha.NewLen(4)
			signUpResult.Msg = "邮箱错误"
			renderJson(ctx, &signUpResult)
			return
		}

		if !dbUpdateUserAccountPassword(g_DBUser, userName, password) {
			signUpResult.CaptchaId = captcha.NewLen(4)
			signUpResult.Msg = "Internal server error"
			renderJson(ctx, &signUpResult)
			return
		}

		//	all ok, redirect to signin page
		signUpResult.Result = 0
		signUpResult.Msg = "/account/modifypasswordsuccess?account=" + userName
		renderJson(ctx, &signUpResult)
	} else {
		http.Redirect(ctx.w, ctx.r, "/", http.StatusFound)
	}
}

var modifyPasswordSuccessRenderTpls = []string{
	"template/account/modifypasswordsuccess.html",
}

func webModifyPasswordSuccessHandler(ctx *RequestContext) {
	ctx.r.ParseForm()
	tplData := make(map[string]interface{})
	username := ctx.r.Form.Get("account")
	tplData["account"] = username
	data := renderTemplate(ctx, modifyPasswordSuccessRenderTpls, tplData)
	ctx.w.Write(data)
}

//	about
var webAboutRenderTpls = []string{
	"template/about.html",
}

func webAboutHandler(ctx *RequestContext) {
	data := renderTemplate(ctx, webAboutRenderTpls, nil)
	ctx.w.Write(data)
}

//	servers
var webServersTpls = []string{
	"template/servers.html",
}

func webServersHandler(ctx *RequestContext) {
	renderData := make(map[string]interface{})
	//	get server list from tcp routine
	var event internalEventFetchServerList
	event.done = make(chan struct{}, 1)
	var tcpEvent tcpnetwork.ConnEvent
	tcpEvent.EventType = kTCPInternalEvent_FetchServerList
	tcpEvent.Userdata = &event

	g_ServerS.Push(&tcpEvent)

	select {
	case <-event.done:
		{
			//	nothing
		}
	case <-time.After(time.Millisecond * 100):
		{
			//	timeout
			seelog.Error("Timeout on internalEventFetchServerList")
			event.servers = make([]*ServerNodeExpose, 0, 0)
		}
	}
	renderData["ServerList"] = event.servers

	data := renderTemplate(ctx, webServersTpls, renderData)
	ctx.w.Write(data)
}
