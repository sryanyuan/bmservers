package main

//#include <stdlib.h>
import "C"

//	Go
import (
	"database/sql"
	//	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime/debug"
	"time"

	"github.com/cihub/seelog"
	"github.com/sryanyuan/tcpnetwork"
)

var (
	kDefaultLogSetting = `
	<seelog minlevel="debug">
    	<outputs formatid="main">
			<rollingfile namemode="postfix" type="date" filename="log/app.log" datepattern="060102" maxrolls="30"/>
       		<console />
    	</outputs>
    	<formats>
        	<format id="main" format="%Date/%Time [%LEV] %Msg (%File:%Line %FuncShort)%n"/>
    	</formats>
	</seelog>
	`
)

var (
	g_ServerS        *tcpnetwork.TCPNetwork
	g_ServerC        *tcpnetwork.TCPNetwork
	g_ServerSSeed    int
	g_ServerCSeed    int
	g_UserList       *UserInfoList
	g_ServerList     *UserInfoList
	g_CtrlCh         chan uint8
	g_DBUser         *sql.DB
	g_DBCrashReport  *sql.DB
	g_Redis          *RedisOperator
	g_AvaliableGS    uint32
	g_strVersionInfo string = "1.0.1"
	g_ControlAddr    []string
)

func exceptionDetails() {
	if err := recover(); err != nil {
		seelog.Error("Exception!error:", err, "stack:")
		debug.PrintStack()
	}
}

func main() {
	defer func() {
		seelog.Info("Server terminated.")
		exceptionDetails()
		var input string
		fmt.Scanln(&input)
	}()

	//	init log module
	var err error
	logFilePath := "./conf/log.conf"
	var logger seelog.LoggerInterface
	logFileExist, _ := pathExists(logFilePath)

	if !logFileExist {
		//	using the default setting
		log.Printf("[WRN] Can't open %s, using the default log setting: %s", logFilePath, kDefaultLogSetting)
		logger, err = seelog.LoggerFromConfigAsString(kDefaultLogSetting)
		if nil != err {
			panic(err)
		}
	} else {
		logger, err = seelog.LoggerFromConfigAsFile(logFilePath)
		panic(err)
	}
	seelog.ReplaceLogger(logger)

	g_ControlAddr = make([]string, 0, 10)
	ReadControlAddr("./login/gmlist.txt")

	//	Load config
	ipaddrclient := flag.String("tcp-client-addr", "", "Listen clients")
	ipaddrserver := flag.String("tcp-gs-addr", "", "Listen gameserver")
	redisAddress := flag.String("redis-addr", "", "Redis address")
	httpAddr := flag.String("http-addr", "", "http listen address (for game room)")
	rpcHttpAddr := flag.String("http-rpc-addr", "", "rpc address")
	webHttpAddr := flag.String("http-web-addr", "", "web http address")
	flag.Parse()
	if len(*ipaddrclient) == 0 || len(*ipaddrserver) == 0 {
		log.Println("invalid input parameters.")
		flag.PrintDefaults()
		return
	}

	seelog.Info("BackMIR Login Server started.")
	//	Initialize directory
	if !PathExist("./login") {
		os.Mkdir("./login", os.ModeDir)
	}

	//	Initialize the database
	g_DBUser = initDatabaseUserV2("./login/users.db")
	if nil == g_DBUser {
		seelog.Error("Initialize database failed.")
		return
	}
	defer g_DBUser.Close()

	//	Initialize bug report database
	g_DBCrashReport = initDatabaseCrashReport("./login/crashreport.db")

	//	Initialize redis
	g_Redis = NewRedisOperator()
	if len(*redisAddress) != 0 {
		g_Redis.Run(*redisAddress, "bmevent")
	}

	//	for server
	g_ServerList = &UserInfoList{
		allusers: make(map[uint32]IUserInterface),
	}
	g_ServerS = tcpnetwork.NewTCPNetwork(1024, tcpnetwork.NewStreamProtocol4())

	//	for client
	g_ServerC = tcpnetwork.NewTCPNetwork(1024, tcpnetwork.NewStreamProtocol4())
	g_ServerC.SetReadTimeoutSec(60)
	g_UserList = &UserInfoList{
		allusers: make(map[uint32]IUserInterface),
	}

	//	http server
	if httpAddr != nil &&
		len(*httpAddr) != 0 {
		startHttpServer(*httpAddr)
	}

	//	rpc server
	if rpcHttpAddr != nil &&
		len(*rpcHttpAddr) != 0 {
		startRPCServer(*rpcHttpAddr)
	}

	//	web server
	if len(*webHttpAddr) != 0 {
		startWebServer(*webHttpAddr)
	}

	//	Initialize dll module
	if !initDllModule("./login/BMHumSaveControl.dll") {
		seelog.Error("Can't load the save control module.")
	}

	//	main thread message handler
	MainThreadInit()

	//	start scheduler
	g_scheduleManager.Start()

	timerTick := time.Tick(time.Duration(5) * time.Second)

	//	start servers
	if err = g_ServerS.Listen(*ipaddrserver); nil != err {
		seelog.Error(err)
		return
	}
	if err = g_ServerC.Listen(*ipaddrclient); nil != err {
		seelog.Error(err)
		return
	}

	seelog.Info("Start process event.listen server:", *ipaddrserver, " listen client:", *ipaddrclient)

	for {
		select {
		case evt := <-g_ServerS.GetEventQueue():
			{
				//ProcessServerSEvent(evt)
				processEventFromServer(evt)
			}
		case evt := <-g_ServerC.GetEventQueue():
			{
				ProcessServerCEvent(evt)
			}
		case evt := <-g_Redis.outputChan:
			{
				ProcessRedisEvent(evt)
			}
		case <-time.After(time.Duration(5) * time.Minute):
			{
				ReadControlAddr("./login/gmlist.txt")
			}
		case evt := <-g_chanMainThread:
			{
				ProcessMThreadMsg(evt)
			}
		case <-time.After(time.Duration(30) * time.Second):
			{
				UpdateMThreadMsg()
			}
		case <-timerTick:
			{
				UpdateTimerEvent()
			}
		}
	}

	seelog.Info("Quit process event...")
	releaseDllModule()
}

func ProcessServerCEvent(evt *tcpnetwork.ConnEvent) {
	switch evt.EventType {
	case tcpnetwork.KConnEvent_Connected:
		{
			HandleCConnect(evt)
		}
	case tcpnetwork.KConnEvent_Data:
		{
			HandleCMsg(evt)
		}
	case tcpnetwork.KConnEvent_Disconnected:
		{
			HandleCDisconnect(evt)
		}
	default:
		{
			seelog.Warn("Unsolved ConnEvent[evtid:", evt.EventType, "]")
		}
	}
}

func ProcessServerSEvent(evt *tcpnetwork.ConnEvent) {
	switch evt.EventType {
	case tcpnetwork.KConnEvent_Connected:
		{
			HandleSConnect(evt)
		}
	case tcpnetwork.KConnEvent_Data:
		{
			HandleSMsg(evt)
		}
	case tcpnetwork.KConnEvent_Disconnected:
		{
			HandleSDisconnect(evt)
		}
	default:
		{
			seelog.Warn("Unsolved ConnEvent[evtid:", evt.EventType, "]")
		}
	}
}

func ProcessRedisEvent(evt *RedisEvent) {
	switch evt.CommandType {
	case RedisEvent_SavePlayerData:
		{
			OfflineSaveUserData(evt.BinaryData)
		}
	}
}
