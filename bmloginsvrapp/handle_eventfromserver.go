package main

//#include <stdlib.h>
import "C"

import (
	"encoding/binary"
	"fmt"

	"unsafe"

	"os"

	"github.com/cihub/seelog"
	"github.com/golang/protobuf/proto"
	"github.com/sryanyuan/bmservers/protocol"
	"github.com/sryanyuan/tcpnetwork"
)

type ServerNode struct {
	serverConnId  int
	serverId      int
	serverName    string
	exposeAddress string
	conn          *tcpnetwork.Connection
}

//	variables
var (
	serverNodeSeed int
	serverNodeMap  map[int]*ServerNode
)

func init() {
	serverNodeSeed = 1
	serverNodeMap = make(map[int]*ServerNode) // key:serverId
}

func processEventFromServer(evt *tcpnetwork.ConnEvent) {
	switch evt.EventType {
	case tcpnetwork.KConnEvent_Connected:
		{
			onEventFromServerConnected(evt)
		}
	case tcpnetwork.KConnEvent_Disconnected:
		{
			onEventFromServerDisconnected(evt)
		}
	case tcpnetwork.KConnEvent_Data:
		{
			onEventFromServerData(evt)
		}
	}
}

func onEventFromServerConnected(evt *tcpnetwork.ConnEvent) {
	evt.Conn.SetConnId(serverNodeSeed)
	var sn ServerNode
	sn.serverConnId = serverNodeSeed
	serverNodeSeed++
	evt.Conn.SetUserdata(&sn)
}

func onEventFromServerDisconnected(evt *tcpnetwork.ConnEvent) {
	evt.Conn.Free()
	sni := evt.Conn.GetUserdata()
	evt.Conn.SetUserdata(nil)

	if sni != nil {
		sn, ok := sni.(*ServerNode)
		if !ok {
			return
		}
		delete(serverNodeMap, sn.serverId)
		seelog.Info("GS ", sn.serverName, " disconnected , address : ", sn.exposeAddress)

		if sn.serverId != 0 {
			//	remove all online players
			dbRemoveOnlinePlayerByServerId(g_DBUser, sn.serverId)
		}
	}
}

func onEventFromServerData(evt *tcpnetwork.ConnEvent) {
	var user *ServerNode
	userData := evt.Conn.GetUserdata()
	if nil != userData {
		user = userData.(*ServerNode)
	}
	opcode := binary.LittleEndian.Uint32(evt.Data)
	op := protocol.LSOp(opcode)

	//	need registered
	if nil == user {
		seelog.Error("Invalid user")
		evt.Conn.Close()
		return
	}

	if user.serverId == 0 {
		if opcode == uint32(loginopstart+1) {
			//	old protocol compatible
			var protoNtf protocol.MProtoTypeNtf
			protoNtf.ProtoVersion = proto.Int(1)
			sendProto(evt.Conn, uint32(protocol.LSOp_ProtoTypeNtf), &protoNtf)
			return
		}
		//	register first
		if opcode != uint32(protocol.LSOp_RegisterServerReq) {
			seelog.Error("Invalid server register package")
			evt.Conn.Close()
			return
		}

		var pb protocol.MRegisterServerReq
		if err := proto.Unmarshal(evt.Data[4:], &pb); nil != err {
			seelog.Error("Failed to unmarshal protobuf : ", opcode)
			evt.Conn.Close()
			return
		}

		//	invalid server id ?
		if 0 == pb.GetServerID() {
			seelog.Error("Invalid server id")
			evt.Conn.Close()
			return
		}

		//	duplicated server id ?
		if _, ok := serverNodeMap[int(pb.GetServerID())]; ok {
			seelog.Error("Duplicated server id , reregisterd ? ", pb.GetServerID())
			evt.Conn.Close()
			return
		}

		//	done
		user.exposeAddress = pb.GetExposeAddress()
		user.serverId = int(pb.GetServerID())
		user.serverName = pb.GetServerName()
		user.conn = evt.Conn
		serverNodeMap[user.serverId] = user
		seelog.Info("Server [", user.serverId, "] register success")

		//	send rankInfo data
		//rankListData := getPlayerRankListV2(user.serverId)
		rankListData := getPlayerRankList()
		var rankNtf protocol.MSyncPlayerRankNtf
		rankNtf.Data = proto.String(rankListData)
		sendProto(evt.Conn, uint32(protocol.LSOp_SyncPlayerRankNtf), &rankNtf)

		//	create directory
		path := fmt.Sprintf("./login/gs_%d", user.serverId)
		if !PathExist(path) {
			err := os.Mkdir(path, os.ModeDir)
			if err != nil {
				seelog.Error("Cant't create user directory.Error:", err)
			}
		}

		return
	}

	//	registered server
	switch op {
	case protocol.LSOp_UserInternalVerifyReq:
		{
			//	set the user LS index
			var pb protocol.MUserInternalVerifyReq
			if err := proto.Unmarshal(evt.Data[4:], &pb); nil != err {
				seelog.Error("Failed to unmarshal protobuf : ", op)
				evt.Conn.Close()
				return
			}

			if client, ok := clientNodeMap[pb.GetLID()]; !ok {
				seelog.Error("Failed to save player data, cannot find player by LID :", pb.GetLID())
			} else {
				if client.uid == 0 {
					seelog.Error("Client not verified")
					return
				}

				if pb.GetAccessToken() != client.accessToken {
					seelog.Error("Client access token verify failed, UID : ", client.uid, " token:", pb.GetAccessToken(), " wanted:", client.accessToken)
				} else {
					if client.gsServerId != 0 {
						seelog.Error("Client already login ???")
						return
					}
					client.gsConnId = pb.GetGID()
					client.connCode = pb.GetConnCode()
					client.gsServerId = user.serverId
					seelog.Info("Verify access token of player : ", client.uid, " to gs ", user.serverId, " success")
				}

				//	sync humdata to client
				syncPlayerHumBaseData(evt.Conn, client)
			}
		}
	case protocol.LSOp_SavePlayerDataReq:
		{
			//	save player data
			var pb protocol.MSavePlayerDataReq
			if err := proto.Unmarshal(evt.Data[4:], &pb); nil != err {
				seelog.Error("Failed to unmarshal protobuf : ", op)
				evt.Conn.Close()
				return
			}

			SavePlayerData(&pb)
		}
	case protocol.LSOp_UpdatePlayerRankReq:
		{
			//	update player rank
			var pb protocol.MUpdatePlayerRankReq
			if err := proto.Unmarshal(evt.Data[4:], &pb); nil != err {
				seelog.Error("Failed to unmarshal protobuf : ", op)
				evt.Conn.Close()
				return
			}

			var rankInfo UserRankInfo
			rankInfo.Uid = pb.GetUID()
			rankInfo.Job = int(pb.GetJob())
			rankInfo.Level = int(pb.GetLevel())
			rankInfo.Name = pb.GetName()
			rankInfo.Power = int(pb.GetPower())
			rankInfo.ServerId = int(pb.GetServerID())
			if !dbUpdateUserRankInfoV2(g_DBUser, &rankInfo) {
				seelog.Error("Failed to insert player rank info")
			}
		}
	case protocol.LSOp_SavePlayerExtDataReq:
		{
			//	save player extend data
			var pb protocol.MSavePlayerExtDataReq
			if err := proto.Unmarshal(evt.Data[4:], &pb); nil != err {
				seelog.Error("Failed to unmarshal protobuf : ", op)
				evt.Conn.Close()
				return
			}

			SavePlayerExtData(&pb)
		}
	case protocol.LSOp_PlayerDisconnectedNtf:
		{
			var pb protocol.MPlayerDisconnectedNtf
			if err := proto.Unmarshal(evt.Data[4:], &pb); nil != err {
				seelog.Error("Failed to unmarshal protobuf : ", op)
				evt.Conn.Close()
				return
			}

			client, ok := clientNodeMap[pb.GetLID()]
			if !ok {
				return
			}
			if 0 == client.uid {
				return
			}
			//	remove online player info
			dbRemoveOnlinePlayerByUID(g_DBUser, client.uid)
		}
	default:
		{
			seelog.Warn("Unknown opcode : ", opcode)
		}
	}
}

func SavePlayerExtData(pb *protocol.MSavePlayerExtDataReq) {
	seelog.Debug(pb.GetName(), " request to save extend data.ext index:", pb.GetExtIndex())

	//	Create save file
	var filehandle uintptr = getSaveFileHandle(int(pb.GetServerID()), pb.GetUID())
	if 0 == filehandle {
		seelog.Error("Failed to get file handle : ", pb)
		return
	}
	//	Close
	defer g_procMap["CloseHumSave"].Call(filehandle)

	cname := C.CString(pb.GetName())
	//	no free!
	defer C.free(unsafe.Pointer(cname))

	r1, _, _ := g_procMap["WriteExtendData"].Call(filehandle, uintptr(unsafe.Pointer(cname)), uintptr(pb.GetExtIndex()), uintptr(unsafe.Pointer(&pb.Data[0])), uintptr(len(pb.Data)))
	if r1 != 0 {
		seelog.Error("Failed to write gamerole extend data")
	}
}

func SavePlayerData(pb *protocol.MSavePlayerDataReq) {
	seelog.Debug(pb.GetName(), " request to save data")
	var filehandle uintptr = getSaveFileHandle(int(pb.GetServerID()), pb.GetUID())
	if 0 == filehandle {
		seelog.Error("Failed to get file handle : ", pb)
		return
	}
	//	Close
	defer g_procMap["CloseHumSave"].Call(filehandle)

	cname := C.CString(pb.GetName())
	//	no free!
	defer C.free(unsafe.Pointer(cname))

	r1, _, _ := g_procMap["UpdateGameRoleInfo"].Call(filehandle, uintptr(unsafe.Pointer(cname)), uintptr(pb.GetLevel()))
	if r1 != 0 {
		seelog.Error("Failed to update gamerole head data")
	}

	r1, _, _ = g_procMap["WriteGameRoleData"].Call(filehandle, uintptr(unsafe.Pointer(cname)), uintptr(unsafe.Pointer(&pb.Data[0])), uintptr(len(pb.Data)))
	if r1 != 0 {
		seelog.Error("Failed to write gamerole data")
	}
}
