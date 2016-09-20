package main

import (
	"database/sql"
	"net/http"
	"time"

	"strconv"
	"strings"

	"github.com/cihub/seelog"
	"github.com/dchest/captcha"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
)

/*
	Permission
*/
const (
	kPermission_None       = iota // 默认权限，禁止访问
	kPermission_Guest             // 游客
	kPermission_User              // 注册用户
	kPermission_Admin             // 管理员
	kPermission_SuperAdmin        // 超级管理员
)

func checkPermission(perChecked uint32, want uint32) bool {
	if perChecked > kPermission_SuperAdmin ||
		want > kPermission_SuperAdmin {
		return false
	}

	if perChecked == kPermission_None ||
		want == kPermission_None {
		return false
	}

	if perChecked >= want {
		return true
	}

	return false
}

/*
	Http context
*/
type RequestContext struct {
	w         http.ResponseWriter
	r         *http.Request
	dbSession *sql.DB
	user      *WebUser
	tmRequest time.Time
}
type HttpHandler func(*RequestContext)

func (this *RequestContext) Redirect(url string, code int) {
	http.Redirect(this.w, this.r, url, code)
}

func (this *RequestContext) GetSession(name string) (*sessions.Session, error) {
	return store.Get(this.r, name)
}

func (this *RequestContext) GetWebUser() *WebUser {
	user := &WebUser{
		Permission: kPermission_Guest,
	}
	session, err := this.GetSession("user")
	if nil != err {
		seelog.Error(err)
		return user
	}

	userinfokey, ok := session.Values["login-key"].(string)
	if !ok {
		return user
	}

	//	parse info
	infoKeys := strings.Split(userinfokey, ":")
	if nil == infoKeys ||
		len(infoKeys) != 2 {
		seelog.Warn("invalid info keys:", infoKeys)
		return user
	}
	uid, err := strconv.Atoi(infoKeys[0])
	if nil != err ||
		0 == uid {
		seelog.Warn(err)
		return user
	}

	//	get user from db
	dbuser, err := dbWebUserGetByUid(g_DBUser, uint32(uid))
	if nil != err {
		seelog.Warn(err)
		return user
	}
	if dbuser.UserName != infoKeys[1] {
		seelog.Warn("Not equal : ", dbuser.UserName, " != ", infoKeys[1])
		return user
	}
	return dbuser
}

func (this *RequestContext) SaveWebUser(user *WebUser, saveDays int) {
	session, err := this.GetSession("user")
	if nil != err {
		seelog.Error(err)
		return
	}

	if 0 == user.Uid {
		return
	}

	userinfokey := strconv.Itoa(int(user.Uid)) + ":" + user.UserName
	session.Values["login-key"] = userinfokey
	if 0 != saveDays {
		session.Options = &sessions.Options{
			MaxAge: saveDays * 24 * 60 * 60,
		}
	}
	err = session.Save(this.r, this.w)
	if nil != err {
		seelog.Error(err)
	}
}

func (this *RequestContext) ClearWebUser() {
	session, err := this.GetSession("user")
	if nil != err {
		return
	}

	session.Options = &sessions.Options{MaxAge: -1}
	session.Save(this.r, this.w)
}

func (this *RequestContext) RenderJson(js interface{}) {
	renderJson(this, js)
}

/*
	Handler warper
*/
func responseWithAccessDenied(w http.ResponseWriter) {
	http.Error(w, "Access denied", http.StatusForbidden)
}

func wrapHandler(item *RouterItem) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		requestCtx := RequestContext{
			w:         w,
			r:         r,
			dbSession: nil,
			tmRequest: time.Now(),
		}

		user := requestCtx.GetWebUser()

		//	check permission
		if !checkPermission(user.Permission, item.Permission) {
			responseWithAccessDenied(w)
			return
		}

		seelog.Debug("Request url : ", r.URL, " user :", user.Uid)

		requestCtx.user = user
		item.Handler(&requestCtx)
	}
}

/*
	Router item
*/
type RouterItem struct {
	Url        string      // 路由的url
	Permission uint32      // url访问权限
	Handler    HttpHandler // 处理器
}

var routerItems = []RouterItem{
	{"/", kPermission_Guest, webIndexHandler},
	{"/account/regaccount", kPermission_Guest, webRegAccountHandler},
	{"/signin", kPermission_Guest, webSigninHandler},
	{"/signout", kPermission_User, webSignOutHandler},
	{"/account/signupsuccess", kPermission_Guest, signupSuccessHandler},
	{"/account/modifypassword", kPermission_Guest, webModifyPasswordHandler},
	{"/account/modifypasswordsuccess", kPermission_Guest, webModifyPasswordSuccessHandler},
	{"/about", kPermission_Guest, webAboutHandler},
	{"/servers", kPermission_Guest, webServersHandler},
}

func fileHandler(w http.ResponseWriter, r *http.Request) {
	filePath := r.URL.Path[1:]
	http.ServeFile(w, r, filePath)
}

func InitRouters(m *http.ServeMux, r *mux.Router) {
	//	handle func
	routersCount := len(routerItems)
	for i := 0; i < routersCount; i++ {
		seelog.Debug("Register url handler : ", routerItems[i].Url)
		r.HandleFunc(routerItems[i].Url, wrapHandler(&routerItems[i]))
	}
	for i := range routerManagementItems {
		seelog.Debug("Register url handler : ", routerManagementItems[i].Url)
		r.HandleFunc(routerManagementItems[i].Url, wrapHandler(&routerManagementItems[i]))
	}
	captchaStorage := captcha.NewMemoryStore(captcha.CollectNum, time.Minute*time.Duration(2))
	captcha.SetCustomStore(captchaStorage)
	m.Handle("/captcha/", captcha.Server(100, 40))

	//	static file
	m.Handle("/static/css/", http.FileServer(http.Dir(".")))
	m.Handle("/static/js/", http.FileServer(http.Dir(".")))
	m.Handle("/static/images/", http.FileServer(http.Dir(".")))
	m.Handle("/static/fonts/", http.FileServer(http.Dir(".")))
}
