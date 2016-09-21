package main

import (
	"os"

	"strings"

	"github.com/cihub/seelog"
)

func ControlValid(addr string) bool {
	if len(g_ControlAddr) == 0 {
		return false
	}

	for _, v := range g_ControlAddr {
		if v == addr {
			return true
		}
	}

	return false
}

func ReadControlAddr(path string) bool {
	//	read control addr
	file, err := os.Open(path)
	if err != nil {
		seelog.Error(err)
		return false
	}

	buf := make([]byte, 512)
	defer file.Close()
	readbytes, readerr := file.Read(buf)
	if readerr != nil {
		seelog.Error(err)
		return false
	}

	content := string(buf[0:readbytes])
	g_ControlAddr = strings.Split(content, "\r\n")

	for _, v := range g_ControlAddr {
		seelog.Info("Controller: ", v, " length", len(v))
	}
	return true
}
