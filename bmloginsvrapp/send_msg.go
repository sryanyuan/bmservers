package main

import (
	"bytes"
	"encoding/binary"

	"github.com/sryanyuan/tcpnetwork"
)

func WriteMsgLittleEndian(conn *tcpnetwork.Connection, opcode uint32, body []byte) error {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, &opcode)
	if body != nil {
		binary.Write(buf, binary.LittleEndian, body)
	}
	return conn.Send(buf.Bytes(), 0)
}
