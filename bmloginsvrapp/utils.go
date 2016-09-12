package main

import (
	"bytes"
	"encoding/binary"
	"os"

	"github.com/golang/protobuf/proto"
	"github.com/sryanyuan/bmservers/protocol"
	"github.com/sryanyuan/tcpnetwork"
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
