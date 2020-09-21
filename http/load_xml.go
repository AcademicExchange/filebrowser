package http

import (
	"encoding/xml"
	"io/ioutil"
	"log"
	"strings"
)

type XMLExcelCfg struct {
	XMLName   xml.Name `xml:"excel"`
	ExcelName string   `xml:"name,attr"`
}

type XMLSvrCfg struct {
	XMLName   xml.Name      `xml:"server"`
	SvrName   string        `xml:"name,attr"`
	ExcelList []XMLExcelCfg `xml:"excel"`
}

type XMLSvrLoadResult struct {
	XMLName   xml.Name    `xml:"root"`
	ServerCfg []XMLSvrCfg `xml:"server"`
}

type SvrLoadCfg struct {
	SvrName  string
	LoadList []string
}

var svrNameToIDMap = map[string]string{
	"Common":         "*.*.*.*",
	"GameSvr":        "*.*.13.*",
	"MatchSvr":       "*.*.17.*",
	"PvpAgentSvr":    "*.*.16.*",
	"ChatSvr":        "*.*.20.*",
	"ViewSvr":        "*.*.34.*",
	"TeamSvr":        "*.*.19.*",
	"ActivitySvr":    "*.*.39.*",
	"WeeklyFubenSvr": "*.*.40.*"}

// Parse which server the db file belongs to
func parseSvrloadXML(xmlFile string) map[string][]string {
	content, err := ioutil.ReadFile(xmlFile)
	if err != nil {
		log.Println(err.Error())
	}

	var result XMLSvrLoadResult
	err = xml.Unmarshal(content, &result)
	if err != nil {
		log.Println(err.Error())
	}

	var loadCfg []SvrLoadCfg
	for _, serverCfg := range result.ServerCfg {
		var svrLoadCfg SvrLoadCfg
		svrLoadCfg.SvrName = serverCfg.SvrName

		for _, excelCfg := range serverCfg.ExcelList {
			svrLoadCfg.LoadList = append(svrLoadCfg.LoadList, strings.Title(excelCfg.ExcelName))
		}
		loadCfg = append(loadCfg, svrLoadCfg)
	}

	res := make(map[string][]string)
	for _, cfg := range loadCfg {
		svr := cfg.SvrName
		if svr == "MonitorSvr" {
			continue
		}
		for _, db := range cfg.LoadList {
			res[db] = append(res[db], svrNameToIDMap[svr])
		}
	}
	return res
}
