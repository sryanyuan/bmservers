package main

//#include <stdlib.h>
import "C"

import (
	"encoding/binary"

	"unsafe"

	"strconv"

	"github.com/cihub/seelog"
	"github.com/golang/protobuf/proto"
	"github.com/sryanyuan/bmservers/protocol"
	"github.com/sryanyuan/tcpnetwork"
)

type ServerNode struct {
	serverId      int
	serverName    string
	exposeAddress string
}

//	variables
var (
	serverNodeSeed int
	serverNodeMap  map[string]*ServerNode
)

func init() {
	serverNodeSeed = 1
	serverNodeMap = make(map[string]*ServerNode)
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
	sn.serverId = serverNodeSeed
	serverNodeSeed++
	evt.Userdata = &sn
}

func onEventFromServerDisconnected(evt *tcpnetwork.ConnEvent) {
	evt.Conn.Free()
	evt.Conn.SetUserdata(nil)
}

func onEventFromServerData(evt *tcpnetwork.ConnEvent) {
	var user *ServerNode
	userData := evt.Userdata
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

	if len(user.exposeAddress) == 0 {
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

		//	duplicated server name ?
		if _, ok := serverNodeMap[pb.GetExposeAddress()]; ok {
			seelog.Error("Duplicated server expose address , reregisterd ? ", pb.GetExposeAddress())
			evt.Conn.Close()
		}

		//	done
		user.exposeAddress = pb.GetExposeAddress()
		serverNodeMap[user.exposeAddress] = user
		seelog.Info("Server [", user.exposeAddress, "] register success")
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
					seelog.Error("Client access token verify failed, UID : ", client.uid)
				} else {
					client.gameConnId = pb.GetGID()
					client.connCode = pb.GetConnCode()
					seelog.Info("Verify access token of player : ", client.uid, " success")
				}
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
			if !dbUpdateUserRankInfo(g_DBUser, &rankInfo) {
				seelog.Error("Failed to insert player rank info")
			}
		}
	case protocol.LSOp_CheckCanBuyOlshopItemReq:
		{
			//	check can buy online shop item
			var pb protocol.MCheckCanBuyOlshopItemReq
			if err := proto.Unmarshal(evt.Data[4:], &pb); nil != err {
				seelog.Error("Failed to unmarshal protobuf : ", op)
				evt.Conn.Close()
				return
			}

			ret := dbCheckConsumeDonate(g_DBUser, pb.GetUID(), int(pb.GetCost()))
			retInt32 := int32(0)
			if ret {
				retInt32 = 1
			}
			//	send response to gs
			var rsp protocol.MCheckCanBuyOlshopItemRsp
			rsp.GID = pb.GID
			rsp.ItemId = pb.ItemId
			rsp.QueryId = pb.QueryId
			rsp.Result = &retInt32
			rsp.UID = pb.UID
			sendProto(evt.Conn, opcode, &rsp)
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
	default:
		{
			seelog.Warn("Unknown opcode : ", opcode)
		}
	}
}

func SavePlayerExtData(pb *protocol.MSavePlayerExtDataReq) {
	seelog.Debug(pb.GetName(), " request to save extend data.ext index:", pb.GetExtIndex())

	//	Create save file
	userfile := "./login/" + strconv.FormatUint(uint64(pb.GetUID()), 10) + "/hum.sav"
	cuserfile := C.CString(userfile)
	defer C.free(unsafe.Pointer(cuserfile))
	r1, _, _ := g_procMap["CreateHumSave"].Call(uintptr(unsafe.Pointer(cuserfile)))
	//	Open it
	r1, _, _ = g_procMap["OpenHumSave"].Call(uintptr(unsafe.Pointer(cuserfile)))
	if r1 == 0 {
		seelog.Error("Can't open hum save.Err:", r1)
		return
	}
	var filehandle uintptr = r1
	//	Close
	defer g_procMap["CloseHumSave"].Call(filehandle)

	cname := C.CString(pb.GetName())
	//	no free!
	defer C.free(unsafe.Pointer(cname))

	r1, _, _ = g_procMap["WriteExtendData"].Call(filehandle, uintptr(unsafe.Pointer(cname)), uintptr(pb.GetExtIndex()), uintptr(unsafe.Pointer(&pb.Data[0])), uintptr(len(pb.Data)))
	if r1 != 0 {
		seelog.Error("Failed to write gamerole extend data")
	}
}

func SavePlayerData(pb *protocol.MSavePlayerDataReq) {
	//	Create save file
	userfile := "./login/" + strconv.FormatUint(uint64(pb.GetUID()), 10) + "/hum.sav"
	cuserfile := C.CString(userfile)
	defer C.free(unsafe.Pointer(cuserfile))
	r1, _, _ := g_procMap["CreateHumSave"].Call(uintptr(unsafe.Pointer(cuserfile)))
	//	Open it
	r1, _, _ = g_procMap["OpenHumSave"].Call(uintptr(unsafe.Pointer(cuserfile)))
	if r1 == 0 {
		seelog.Error("Can't open hum save.Err:", r1)
		return
	}
	var filehandle uintptr = r1
	//	Close
	defer g_procMap["CloseHumSave"].Call(filehandle)

	cname := C.CString(pb.GetName())
	//	no free!
	defer C.free(unsafe.Pointer(cname))

	r1, _, _ = g_procMap["UpdateGameRoleInfo"].Call(filehandle, uintptr(unsafe.Pointer(cname)), uintptr(pb.GetLevel()))
	if r1 != 0 {
		seelog.Error("Failed to update gamerole head data")
	}

	r1, _, _ = g_procMap["WriteGameRoleData"].Call(filehandle, uintptr(unsafe.Pointer(cname)), uintptr(unsafe.Pointer(&pb.Data[0])), uintptr(len(pb.Data)))
	if r1 != 0 {
		seelog.Error("Failed to write gamerole data")
	}
}
