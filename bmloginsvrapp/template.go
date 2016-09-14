package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"strconv"
	"time"
)

//	templates
var (
	tplBinaryDataMap map[string][]byte
	layoutFiles      = []string{
		"template/layout.html",
		"template/component/navbar_v2.html",
		"template/component/footer.html",
	}
)

var tplFuncMap = template.FuncMap{
	"getProcessTime":    tplfn_getprocesstime,
	"getUnixTimeString": tplfn_getUnixTimeString,
	"getTimeGapString":  tplfn_getTimeGapString,
	"convertToHtml":     tplfn_convertToHtml,
	"minusInt":          tplfn_minusInt,
	"addInt":            tplfn_addInt,
	"formatDate":        tplfn_formatDate,
}

func tplfn_getprocesstime(tm time.Time) string {
	return strconv.Itoa((time.Now().Nanosecond()-tm.Nanosecond())/1e6) + " ms"
}

func tplfn_getUnixTimeString(utm int64) string {
	tm := time.Unix(utm, 0)
	return tm.Format("2006-01-02")
}

func tplfn_getTimeGapString(tm int64) string {
	t := time.Unix(tm, 0)
	gap := time.Now().Sub(t)
	if gap.Seconds() < 60 {
		return "刚刚"
	} else if gap.Minutes() < 60 {
		return fmt.Sprintf("%.0f 分钟前", gap.Minutes())
	} else if gap.Hours() < 24 {
		return fmt.Sprintf("%.0f 小时前", gap.Hours())
	} else {
		hours := int(gap.Hours())
		days := hours / 24
		if days < 30 {
			return fmt.Sprintf("%d 天前", days)
		} else {
			return t.Format("2006-01-02 15:04")
		}
	}
}

func tplfn_convertToHtml(str string) template.HTML {
	return template.HTML(str)
}

func tplfn_minusInt(val int, step int) int {
	return val - step
}

func tplfn_addInt(val int, step int) int {
	return val + step
}

func tplfn_formatDate(tm int64) string {
	timeVal := time.Unix(tm, 0)
	return timeVal.Format("2006-01-02")
}

func init() {
	tplBinaryDataMap = make(map[string][]byte)
}

func getTplBinaryData(file string) []byte {
	_, ok := tplBinaryDataMap[file]
	if ok {
		//	directy return data
	}

	layoutData, err := ioutil.ReadFile(file)
	if nil != err {
		panic(err)
	}

	tplBinaryDataMap[file] = layoutData

	return layoutData
}

func parseTemplate(fileNames []string, layoutFiles []string, data map[string]interface{}) []byte {
	var err error
	var buffer bytes.Buffer
	t := template.New("layout").Funcs(tplFuncMap)

	//	parse layout
	for _, v := range layoutFiles {
		tplContent := string(getTplBinaryData(v))

		if t, err = t.Parse(tplContent); nil != err {
			panic(err)
		}
	}

	//	parse files
	if nil != fileNames &&
		len(fileNames) != 0 {
		if t, err = t.ParseFiles(fileNames...); nil != err {
			panic(err)
		}
	}

	//	execute
	if err = t.Execute(&buffer, data); nil != err {
		panic(err)
	}

	return buffer.Bytes()
}

func renderTemplate(ctx *RequestContext, fileNames []string, data map[string]interface{}) []byte {
	//	input some common variables
	if nil == data {
		data = make(map[string]interface{})
	}

	data["url"] = ctx.r.URL.String()
	data["user"] = ctx.user

	//	get render data
	return parseTemplate(fileNames, layoutFiles, data)
}

func renderJson(ctx *RequestContext, js interface{}) {
	if data, err := json.Marshal(js); nil != err {
		panic(err)
	} else {
		ctx.w.Header().Set("Content-Type", "application/json")
		ctx.w.Write(data)
	}
}
