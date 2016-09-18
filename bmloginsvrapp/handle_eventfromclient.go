package main

//#include <stdlib.h>
import "C"

import (
	"encoding/binary"
	"encoding/json"

	"unsafe"

	"container/list"

	"github.com/cihub/seelog"
	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	"github.com/sryanyuan/bmservers/protocol"
	"github.com/sryanyuan/tcpnetwork"
)

type ClientNode struct {
	clientConnId int32
	gsServerId   int
	gsConnId     int32
	connCode     int32
	accessToken  string
	uid          uint32
	conn         *tcpnetwork.Connection
}

var (
	clientNodeMap  map[int32]*ClientNode // key : clientConnId , for all connected peers
	clientUserMap  map[uint32]*list.List // key : uid , for all verified user
	clientNodeSeed int
)

func init() {
	clientNodeSeed = 1
	clientNodeMap = make(map[int32]*ClientNode)
	clientUserMap = make(map[uint32]*list.List)
}

func addClientToClientUserMap(client *ClientNode) {
	clients, ok := clientUserMap[client.uid]
	if !ok {
		clients = list.New()
		clientUserMap[client.uid] = clients
	}

	clients.PushBack(client)
}

func removeClientFromClientUserMap(client *ClientNode) bool {
	clients, ok := clientUserMap[client.uid]
	if !ok {
		return false
	}

	headElement := clients.Front()

	for nil != headElement {
		cn, ok := headElement.Value.(*ClientNode)
		if !ok {
			continue
		}

		if cn == client {
			clients.Remove(headElement)
			return true
		}
		headElement = headElement.Next()
	}

	return false
}

func getClientsFromClientUserMap(uid uint32) *list.List {
	clients, ok := clientUserMap[uid]
	if !ok {
		return nil
	}
	return clients
}

func processEventFromClient(evt *tcpnetwork.ConnEvent) {
	switch evt.EventType {
	case tcpnetwork.KConnEvent_Connected:
		{
			onEventFromClientConnected(evt)
		}
	case tcpnetwork.KConnEvent_Disconnected:
		{
			onEventFromClientDisconnected(evt)
		}
	case tcpnetwork.KConnEvent_Data:
		{
			onEventFromClientData(evt)
		}
	}
}

func onEventFromClientConnected(evt *tcpnetwork.ConnEvent) {
	seelog.Info("Client ", evt.Conn.GetRemoteAddress(), " connected")
	evt.Conn.SetConnId(clientNodeSeed)
	var cn ClientNode
	cn.clientConnId = int32(clientNodeSeed)
	cn.conn = evt.Conn
	clientNodeSeed++
	evt.Conn.SetUserdata(&cn)
	clientNodeMap[cn.clientConnId] = &cn

	//	save access token
	uuidValue, _ := uuid.NewUUID()
	cn.accessToken = uuidValue.String()

	//	send access ntf
	var ntf protocol.MLSAccessNtf
	ntf.AccessToken = proto.String(cn.accessToken)
	ntf.LID = proto.Int32(cn.clientConnId)
	ntf.GameType = proto.Int(2)
	sendProto(evt.Conn, uint32(protocol.LSOp_LSAccessNtf), &ntf)
}

func onEventFromClientDisconnected(evt *tcpnetwork.ConnEvent) {
	cni := evt.Conn.GetUserdata()
	evt.Conn.Free()
	evt.Conn.SetUserdata(nil)

	if nil != cni {
		cn, ok := cni.(*ClientNode)
		if !ok {
			return
		}
		delete(clientNodeMap, cn.clientConnId)
		removeClientFromClientUserMap(cn)
	}
}

func onEventFromClientData(evt *tcpnetwork.ConnEvent) {
	if nil == evt.Conn.GetUserdata() {
		seelog.Error("Invalid client user")
		evt.Conn.Close()
		return
	}

	user, ok := evt.Conn.GetUserdata().(*ClientNode)
	if !ok {
		seelog.Error("Invalid client user")
		evt.Conn.Close()
		return
	}

	opcode := binary.LittleEndian.Uint32(evt.Data)
	op := protocol.LSOp(opcode)

	//	must verify first
	if 0 == user.uid {
		if op == protocol.LSOp_VerifyAccountReq {
			var pb protocol.MVerifyAccountReq
			if err := proto.Unmarshal(evt.Data[4:], &pb); nil != err {
				seelog.Error("Failed to unmarshal protobuf : ", op)
				evt.Conn.Close()
				return
			}

			var ret int = verifyAccount(evt.Conn, user, &pb)
			var rsp protocol.MVerifyAccountRsp
			rsp.Result = proto.Int(ret)
			sendProto(evt.Conn, uint32(protocol.LSOp_VerifyAccountRsp), &rsp)

			//	verify ok
			if 0 != ret {
				return
			}

			addClientToClientUserMap(user)
			syncPlayerHumBaseData(evt.Conn, user)
		}
		return
	}

	//	normal packets
	switch op {
	case protocol.LSOp_CreateHumReq:
		{
			var pb protocol.MCreateHumReq
			if err := proto.Unmarshal(evt.Data[4:], &pb); nil != err {
				seelog.Error("Failed to unmarshal protobuf : ", op)
				evt.Conn.Close()
				return
			}
			onCreateHumReq(evt.Conn, user, &pb)
		}
	case protocol.LSOp_DelHumReq:
		{
			var pb protocol.MDelHumReq
			if err := proto.Unmarshal(evt.Data[4:], &pb); nil != err {
				seelog.Error("Failed to unmarshal protobuf : ", op)
				evt.Conn.Close()
				return
			}
			onDelHumReq(evt.Conn, user, &pb)
		}
	case protocol.LSOp_LoginGameReq:
		{
			var pb protocol.MLoginGameReq
			if err := proto.Unmarshal(evt.Data[4:], &pb); nil != err {
				seelog.Error("Failed to unmarshal protobuf : ", op)
				evt.Conn.Close()
				return
			}
			onLoginGame(evt.Conn, user, &pb)
		}
	case protocol.LSOp_HeartBeatNtf:
		{
			//	nothing
		}
	}
}

func onCreateHumReq(conn *tcpnetwork.Connection, client *ClientNode, pb *protocol.MCreateHumReq) {
	filehandle := getPlayerSaveFileHandle(client)
	if 0 == filehandle {
		seelog.Error("Failed to open save file :", client)
		return
	}
	//	Close
	defer g_procMap["CloseHumSave"].Call(filehandle)
	// role name exists ?
	//if dbUserNameExistV2(g_DBUser, int(client.gsServerId), pb.GetName()) {
	if dbUserNameExist(g_DBUser, pb.GetName()) {
		sendQuickMessage(conn, 4, 0)
		return
	}

	//	add role
	cname := C.CString(pb.GetName())
	defer C.free(unsafe.Pointer(cname))
	r1, _, _ := g_procMap["AddGameRole"].Call(filehandle,
		uintptr(unsafe.Pointer(cname)),
		uintptr(pb.GetJob()),
		uintptr(pb.GetSex()))
	var ntf protocol.MCreateHumRsp
	ntf.Job = pb.Job
	ntf.Name = pb.Name
	ntf.Sex = pb.Sex
	if r1 == 0 {
		//	Success
		//	add role result ret 1byte;namelen 1byte;name namelen
		ntf.Result = proto.Int(1)
		sendProto(conn, uint32(protocol.LSOp_CreateHumRsp), &ntf)

		//	Add user name
		//if !dbAddUserNameV2(g_DBUser, client.uid, int(client.gsServerId), pb.GetName()) {
		if !dbAddUserNameByUid(g_DBUser, client.uid, pb.GetName()) {
			sendQuickMessage(conn, 6, 0)
		}
	} else {
		//	failed
		ntf.Result = proto.Int(0)
		sendProto(conn, uint32(protocol.LSOp_CreateHumRsp), &ntf)
	}
}

func onDelHumReq(conn *tcpnetwork.Connection, client *ClientNode, pb *protocol.MDelHumReq) {
	filehandle := getPlayerSaveFileHandle(client)
	if 0 == filehandle {
		seelog.Error("Failed to open save file :", client)
		return
	}
	//	Close
	defer g_procMap["CloseHumSave"].Call(filehandle)

	cname := C.CString(pb.GetName())
	defer C.free(unsafe.Pointer(cname))

	r1, _, _ := g_procMap["DelGameRole"].Call(filehandle,
		uintptr(unsafe.Pointer(cname)))
	if r1 != 0 {
		seelog.Error("Can't remove gamerole ", pb.GetName())
		return
	}

	//	remove name from db
	//if !dbRemoveUserNameV2(g_DBUser, client.uid, int(client.gsServerId), pb.GetName()) {
	if !dbRemoveUserNameByUid(g_DBUser, client.uid, pb.GetName()) {
		//sendQuickMessage(conn, 5, 0)
		seelog.Error("Failed to remove user name from role_name")
	}
	//if !dbRemoveUserRankInfoV2(g_DBUser, client.gsServerId, pb.GetName()) {
	if !dbRemoveUserRankInfo(g_DBUser, pb.GetName()) {
		seelog.Error("Failed to remove user name from rank_info")
	}
	var rsp protocol.MDelHumRsp
	rsp.Name = pb.Name
	sendProto(conn, uint32(protocol.LSOp_DelHumRsp), &rsp)
}

func onLoginGame(conn *tcpnetwork.Connection, client *ClientNode, pb *protocol.MLoginGameReq) {
	if 0 == client.gsServerId {
		//	invalid server id
		sendQuickMessage(conn, 7, 0)
		return
	}

	//	already in other servers
	prevServerId := dbGetPlayerOnlineServerId(g_DBUser, client.uid)
	if 0 != prevServerId {
		if prevServerId != client.gsServerId {
			seelog.Warn("Relogin, prev serverid:", prevServerId, " serverid:", client.gsServerId)
			return
		}
	}

	gs, ok := serverNodeMap[client.gsServerId]
	if !ok {
		//	invalid server id
		sendQuickMessage(conn, 7, 0)
		return
	}

	filehandle := getPlayerSaveFileHandle(client)
	if 0 == filehandle {
		seelog.Error("Failed to open save file :", client)
		sendQuickMessage(conn, 2, 0)
		return
	}
	//	Close
	defer g_procMap["CloseHumSave"].Call(filehandle)

	var newhum bool = false
	var datasize uint32 = 0
	cname := C.CString(pb.GetName())
	defer C.free(unsafe.Pointer(cname))
	//	Get data size
	r1, _, _ := g_procMap["ReadGameRoleSize"].Call(filehandle,
		uintptr(unsafe.Pointer(cname)))
	if r1 == 0 {
		newhum = true
	} else {
		datasize = uint32(r1)
	}

	var ntf protocol.MPlayerLoginHumDataNtf

	//	read head
	ntf.GID = proto.Int32(client.gsConnId)
	ntf.ConnID = proto.Int32(client.connCode)
	ntf.LID = proto.Int32(client.clientConnId)
	ntf.UID = proto.Uint32(client.uid)

	r1, _, _ = g_procMap["GetGameRoleIndex"].Call(filehandle, uintptr(unsafe.Pointer(cname)))
	if r1 < 0 || r1 > 2 {
		sendQuickMessage(conn, 5, 0)
		seelog.Error("Can't get role index, name :", pb.GetName())
		return
	}

	var heroidx int = int(r1)
	var job, sex uint8
	var level uint16
	r1, _, _ = g_procMap["GetGameRoleInfo_Value"].Call(filehandle, uintptr(heroidx), uintptr(unsafe.Pointer(&job)),
		uintptr(unsafe.Pointer(&sex)), uintptr(unsafe.Pointer(&level)))

	ntf.Name = pb.Name
	ntf.Sex = proto.Int(int(sex))
	ntf.Job = proto.Int(int(job))
	ntf.Level = proto.Int(int(level))

	if !newhum {
		seelog.Info("Not new hum, read size ", datasize, " role name:", pb.GetName())

		humdata := make([]byte, datasize)
		r1, _, _ = g_procMap["ReadGameRoleData"].Call(filehandle,
			uintptr(unsafe.Pointer(cname)),
			uintptr(unsafe.Pointer(&humdata[0])))
		if r1 != 0 {
			seelog.Error("Uid :", client.uid, " save not valid")
			sendQuickMessage(conn, kQM_SaveNotValid, 0)
			return
		}

		//	send gamerole data to server
		ntf.Data = humdata
	}

	//	发送登录扩展信息，json格式
	extInfo := &UserLoginExtendInfo{}
	donateInfo := &UserDonateInfo{}
	if dbGetUserDonateInfo(g_DBUser, client.uid, donateInfo) {
		//	nothing
		extInfo.DonateLeft = int32(dbGetUserDonateLeft(g_DBUser, client.uid))

		seelog.Info("player[", client.uid, "] donate money:", donateInfo.donate, "donate left:", extInfo.DonateLeft)
	}

	extInfo.DonateMoney = donateInfo.donate
	extInfo.SystemGift = dbGetSystemGiftIdByUid(g_DBUser, client.uid)
	binaryExtInfo, jsErr := json.Marshal(extInfo)
	if jsErr != nil {
		seelog.Error("failed to marshal user extend login information:", jsErr)
	} else {
		//	发送扩展信息
		if 0 != len(binaryExtInfo) {
			//	写入json数据
			ntf.JsonData = proto.String(string(binaryExtInfo))
		}
	}

	sendProto(gs.conn, uint32(protocol.LSOp_PlayerLoginHumDataNtf), &ntf)

	//	发送额外的人物数据
	var extNtf protocol.MPlayerLoginExtHumDataNtf
	extNtf.GID = proto.Int32(client.gsConnId)
	extNtf.ConnID = proto.Int32(client.connCode)
	extNtf.UID = proto.Uint32(client.uid)
	extNtf.ExtIndex = proto.Int(0)

	r1, _, _ = g_procMap["ReadExtendDataSize"].Call(filehandle,
		uintptr(unsafe.Pointer(cname)),
		0)
	if 0 != r1 {
		datasize = uint32(r1)
		humextdata := make([]byte, datasize)
		r1, _, _ = g_procMap["ReadExtendData"].Call(filehandle,
			uintptr(unsafe.Pointer(cname)),
			0,
			uintptr(unsafe.Pointer(&humextdata[0])))
		if r1 != 0 {
			sendQuickMessage(conn, kQM_SaveNotValid, 0)
		} else {
			//	send gamerole data to server
			extNtf.Data = humextdata
		}
	} else {
		//	just an empty hum data
	}

	sendProto(gs.conn, uint32(protocol.LSOp_PlayerLoginExtHumDataNtf), &extNtf)

	//	set login status
	err := dbAddOnlinePlayer(g_DBUser, client.uid, client.gsServerId)
	if nil != err {
		seelog.Error(err)
	}
}

func syncPlayerHumBaseData(conn *tcpnetwork.Connection, client *ClientNode) {
	filehandle := getPlayerSaveFileHandle(client)
	if 0 == filehandle {
		seelog.Error("Failed to open save file :", client)
		return
	}
	//	Close
	defer g_procMap["CloseHumSave"].Call(filehandle)

	//	Send head info
	var ntf protocol.MPlayerHumBaseDataNtf
	ntf.Roles = make([]*protocol.MPlayerHumBaseData, 0, 3)
	for i := 0; i < 3; i++ {
		databuf := make([]byte, 100)
		var datalen uintptr
		datalen, _, _ = g_procMap["ReadGameRoleHeadInfo"].Call(filehandle, uintptr(i), uintptr(unsafe.Pointer(&databuf[0])))
		if datalen != 0 {
			var role protocol.MPlayerHumBaseData
			role.RoleIndex = proto.Int(i)
			role.RoleData = databuf
			ntf.Roles = append(ntf.Roles, &role)
		}
	}
	sendProto(conn, uint32(protocol.LSOp_PlayerHumBaseDataNtf), &ntf)
}

func verifyAccount(conn *tcpnetwork.Connection, client *ClientNode, pb *protocol.MVerifyAccountReq) int {
	var ret int = 0
	if !dbUserAccountExist(g_DBUser, pb.GetAccount()) {
		// non-exist account
		ret = 1
	}

	var info UserAccountInfo
	dbret, _ := dbGetUserAccountInfo(g_DBUser, pb.GetAccount(), &info)
	if !dbret {
		ret = 1
	} else {
		if pb.GetPassword() != info.password {
			ret = 2
		} else {
			//	pass
			client.uid = info.uid
		}
	}

	if 0 == ret {
		//	send server information
		if len(serverNodeMap) == 0 {
			sendQuickMessage(conn, 7, 0)
		} else {
			var serverListNtf protocol.MServerListNtf
			serverListNtf.Servers = make([]*protocol.MServerListItem, 0, len(serverNodeMap))
			for _, v := range serverNodeMap {
				var item protocol.MServerListItem
				item.ServerAddress = proto.String(v.exposeAddress)
				item.ServerName = proto.String(v.serverName)
				item.ServerID = proto.Int(v.serverId)
				serverListNtf.Servers = append(serverListNtf.Servers, &item)
			}
			sendProto(conn, uint32(protocol.LSOp_ServerListNtf), &serverListNtf)
		}
	} else {
		sendQuickMessage(conn, 8, 0)
	}

	return ret
}
