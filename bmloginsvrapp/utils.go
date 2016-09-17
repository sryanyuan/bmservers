package main

//#include <stdlib.h>
import "C"

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"unsafe"

	"github.com/cihub/seelog"
	"github.com/golang/protobuf/proto"
	"github.com/sryanyuan/bmservers/protocol"
	"github.com/sryanyuan/tcpnetwork"
)

const (
	kQM_None                = 1
	kQM_HumSaveNotExist     = 2
	kQM_SaveNotValid        = 3
	kQM_NameAlreadyExist    = 4
	kQM_HumDataNotExist     = 5
	kQM_CannotCreateRole    = 6
	kQM_NoGSAvailable       = 7
	kQM_AccountVerifyFailed = 8
	kQM_SaveHumDataFailed   = 9
)

func pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func readUint32FromSlice(sl []byte) uint32 {
	return binary.LittleEndian.Uint32(sl)
}

func sendProto(conn *tcpnetwork.Connection, op uint32, pb proto.Message) error {
	buf := new(bytes.Buffer)
	var opBuf [4]byte
	binary.LittleEndian.PutUint32(opBuf[0:], op)
	buf.Write(opBuf[0:])

	data, err := proto.Marshal(pb)
	if nil != err {
		seelog.Error("Failed to send proto : ", err)
		return err
	}
	buf.Write(data)

	return conn.Send(buf.Bytes(), 0)
}

func sendQuickMessage(conn *tcpnetwork.Connection, msgId int32, param int32) error {
	var pb protocol.MQuickMessageNtf
	pb.MsgId = proto.Int32(msgId)
	pb.Param = proto.Int32(param)

	return sendProto(conn, uint32(protocol.LSOp_QuickMessageNtf), &pb)
}

func getPlayerSaveFileHandle(client *ClientNode) uintptr {
	return getSaveFileHandle(client.gsServerId, client.uid)
}

func getSaveFileHandle(serverId int, uid uint32) uintptr {
	if 0 == uid {
		return 0
	}
	if 0 == serverId {
		return 0
	}

	path := fmt.Sprintf("./login/gs_%d", serverId)
	if !PathExist(path) {
		err := os.Mkdir(path, os.ModeDir)
		if err != nil {
			seelog.Error("Cant't create user directory.Error:", err)
			return 0
		}
	}
	path += fmt.Sprintf("/%d", uid)
	if !PathExist(path) {
		err := os.Mkdir(path, os.ModeDir)
		if err != nil {
			seelog.Error("Cant't create user directory.Error:", err)
			return 0
		}
	}

	//	create new save file if not exists
	userfile := path + "/hum.sav"
	cuserfile := C.CString(userfile)
	defer C.free(unsafe.Pointer(cuserfile))
	g_procMap["CreateHumSave"].Call(uintptr(unsafe.Pointer(cuserfile)))

	//	Open it
	r1, _, _ := g_procMap["OpenHumSave"].Call(uintptr(unsafe.Pointer(cuserfile)))
	if r1 == 0 {
		seelog.Error("Can't open hum save.Err:", r1)
		return 0
	}

	return r1
}

func getFormValueInt(r *http.Request, key string, failed int) int {
	v := strings.TrimSpace(r.Form.Get(key))
	if len(v) == 0 {
		return failed
	}

	value, err := strconv.Atoi(v)
	if nil != err {
		return failed
	}
	return value
}

func getFormValueString(r *http.Request, key string) string {
	return strings.TrimSpace(r.Form.Get(key))
}
