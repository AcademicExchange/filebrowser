package main

import (
    "encoding/json"
    "fmt"
    "io/ioutil"
)

type Socket struct {
    IP       string `json:"ip"`
    Port     int    `json:"port"`
    Username string `json:"username`
    Password string `json:"password"`
}

func (s *Socket) GetUrl() string {
    return fmt.Sprintf("http://%s:%d", s.IP, s.Port)
}

func (s *Socket) String() string {
    return fmt.Sprintf("ip: %s|port: %d", s.IP, s.Port)
}

type Server struct {
    Desc  string   `json:"desc"`
    Tcm   Socket   `json:"tcm"`
    Nodes []Socket `json:"nodes"`
}

func (s *Server) String() string {
    return fmt.Sprintf("desc: %s|tcm: %v|nodes: %v", s.Desc, s.Tcm, s.Nodes)
}

var configFile = "config.json"

func loadConfig() (map[string]Server, error) {
    var svrMap map[string]Server
    buffer, err := ioutil.ReadFile(configFile)
    if err != nil {
        log.Errorf("read config file %s failed for %v", configFile, err)
        return svrMap, err
    }
    if err := json.Unmarshal(buffer, &svrMap); err != nil {
        log.Errorf("JSON unmarshal failed for %v", err)
        return svrMap, err
    }
    return svrMap, nil
}
