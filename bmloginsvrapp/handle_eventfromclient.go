package main

import (
	"encoding/binary"

	"github.com/cihub/seelog"
	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	"github.com/sryanyuan/bmservers/protocol"
	"github.com/sryanyuan/tcpnetwork"
)

type ClientNode struct {
	clientId    int32
	gameConnId  int32
	connCode    int32
	accessToken string
	uid         uint32
}

var (
	clientNodeMap  map[int32]*ClientNode
	clientNodeSeed int
)

func init() {
	clientNodeSeed = 1
	clientNodeMap = make(map[int32]*ClientNode)
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
	evt.Conn.SetConnId(clientNodeSeed)
	var cn ClientNode
	cn.clientId = int32(clientNodeSeed)
	clientNodeSeed++
	evt.Conn.SetUserdata(&cn)

	//	send access ntf
	var ntf protocol.MLSAccessNtf
	uuidValue, _ := uuid.NewUUID()
	ntf.AccessToken = proto.String(uuidValue.String())
	ntf.LID = proto.Int32(cn.clientId)
	sendProto(evt.Conn, uint32(protocol.LSOp_LSAccessNtf), &ntf)
}

func onEventFromClientDisconnected(evt *tcpnetwork.ConnEvent) {
	evt.Conn.Free()
	evt.Conn.SetUserdata(nil)
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
		}
		return
	}
}

func verifyAccount(conn *tcpnetwork.Connection, client *ClientNode, pb *protocol.MVerifyAccountReq) int {
	var ret int = 0
	if !dbUserAccountExist(g_DBUser, pb.GetAccount()) {
		// non-exist account
		ret = 1
	}

	var info UserAccountInfo
	dbret, _ := dbGetUserAccountInfo(g_DBUser, pb.GetAccount(), &info)
	seelog.Info("Accout ", info.account, " Password ", info.password)
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
			serverListNtf.Servers = make([]*protocol.MServerListItem, len(serverNodeMap))
			for _, v := range serverNodeMap {
				var item protocol.MServerListItem
				item.ServerAddress = proto.String(v.exposeAddress)
				item.ServerName = proto.String(v.serverName)
			}
			sendProto(conn, uint32(protocol.LSOp_ServerListNtf), &serverListNtf)
		}
	} else {
		sendQuickMessage(conn, 8, 0)
	}

	return ret
}
