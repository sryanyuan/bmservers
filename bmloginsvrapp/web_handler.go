package main

import (
	"net/http"

	"github.com/cihub/seelog"
)

func startWebServer(addr string) {
	if len(addr) == 0 {
		return
	}
	http.HandleFunc("/web/index", webIndexHandler)

	seelog.Info("Start web server:", addr)
	go http.ListenAndServe(addr, nil)
}

func webIndexHandler(w http.ResponseWriter, r *http.Request) {

}
