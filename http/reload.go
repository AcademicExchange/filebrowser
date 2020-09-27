package http

import (
	"fmt"
    "bufio"
    "io"
    "log"
    "net/http"
    "os/exec"
    "path/filepath"
    "regexp"
    "strings"

    mapset "github.com/deckarep/golang-set"
)

type response struct {
    Status string   `json:"status"`
    Msg    []string `json:"msg"`
}

var reloadHandler = withUser(func(w http.ResponseWriter, r *http.Request, d *data) (int, error) {
    // filter requests without uuid or with inconsistent uuid
    uuid := r.URL.Query().Get("uuid")
    if uuid == "" {
        w.WriteHeader(http.StatusForbidden)
        w.Write([]byte("Please upload config first\n"))
        return http.StatusForbidden, nil
    }

    xmlFile := filepath.Join(d.user.Scope, "wedo/ClientConfig/CSCommon/DB/SvrLoadList.xml")
    cfgs := parseSvrloadXML(xmlFile)
    mtx.Lock()
    // fmt.Println(cache)
    found, vals := cache.Get(uuid)
    if !found {
        w.WriteHeader(http.StatusForbidden)
        w.Write([]byte("Someone is currently hot reloading, please try again later!\n"))
        mtx.Unlock()
        return http.StatusForbidden, nil
    }
    var str string
    var svrs []string
    for k, v := range vals {
        log.Println("dir: ", k)
        if v.svrs != nil && v.svrs.Cardinality() > 0 {
            str = "*.*.*.*"
            break
        }

        if v.xmls != nil && v.xmls.Cardinality() > 0 {
            str = "*.*.*.*"
            break
        }

        if v.dbs != nil && v.dbs.Cardinality() > 0 {
            dbs := interSliceToStrSlice(v.dbs.ToSlice())
            for i := 0; i < len(dbs); i++ {
                dbName := filepath.Base(dbs[i])
                dbName = strings.TrimRight(dbName, ".db")
                if arr, ok := cfgs[dbName]; ok {
                    svrs = append(svrs, arr...)
                }
            }
        }
    }

    if str != "*.*.*.*" {
        str = getSvrIDsFromSlice(svrs)
        log.Println("procs", str)
    }

    err, out := execReload(d, str)
    // Command executed, safely clear the cache and unlock
    cache.Clear()
    mtx.Unlock()

    w.WriteHeader(errToStatus(err))

    status := "OK"
    if errToStatus(err) != http.StatusOK {
        status = "Error"
    }

    rsp := &response{
        Status: status,
        Msg:    out,
    }

    if _, err := renderJSONIndent(w, r, rsp); err != nil {
        return errToStatus(err), err
    }

    return errToStatus(err), err
})

func execReload(d *data, proc string) (error, []string) {
    // root: /data/home/user00
    rootDir := d.user.FullPath("")
    tcmDir := filepath.Join(rootDir, "apps/tcm/bin")
    cmd := exec.Command("sh", "console_cmd.sh", "reload "+proc)
    cmd.Dir = tcmDir

    stdout, err := cmd.StdoutPipe()
    if err != nil {
        log.Printf("Get StdoutPipe failed for %s", err.Error())
        return err, []string{"Get StdoutPipe failed for " + err.Error()}
    }

    stderr, err := cmd.StderrPipe()
    if err != nil {
        log.Printf("Get StderrPipe failed for %s", err.Error())
        return err, []string{"Get StderrPipe failed for " + err.Error()}
    }

    if err := cmd.Start(); err != nil {
        log.Printf("Start command failed for %s", err.Error())
        return err, []string{"Start command failed for " + err.Error()}
    }

    var arr []string
    s := bufio.NewScanner(io.MultiReader(stdout, stderr))
    for s.Scan() {
        // match the strings with failed reloading
        regstr := "(?i)^.*\\[failed\\]$|(?i)^.*\\[succeed\\]$"
        line := s.Text()
        log.Println(line)
        if match, _ := regexp.MatchString(regstr, line); match {
            arr = append(arr, strings.TrimSpace(line))
        }
    }

    if err := cmd.Wait(); err != nil {
        log.Printf("Command execution failed for %s", err.Error())
        return err, arr
    }
    return nil, arr
}

func interSliceToStrSlice(inters []interface{}) []string {
    strs := make([]string, len(inters))
    for i, item := range inters {
        strs[i] = item.(string)
    }
    return strs
}

func getProcStr(strs []string) string {
    return "[" + strings.Join(strs, ",") + "]"
}

func getSvrIDsFromSlice(svrs []string) string {
    words := mapset.NewSet()
    zones := mapset.NewSet()
    procs := mapset.NewSet()
    insts := mapset.NewSet()
    for _, svr := range svrs {
        seps := strings.Split(svr, ".")
        if seps[0] == "*" {
            if !words.Contains(seps[0]) {
                words.Clear()
                words.Add(seps[0])
            }
        } else {
            words.Add(seps[0])
        }
        if seps[1] == "*" {
            if !zones.Contains(seps[1]) {
                zones.Clear()
                zones.Add(seps[1])
            }
        } else {
            zones.Add(seps[1])
        }
        if seps[2] == "*" {
            if !procs.Contains(seps[2]) {
                procs.Clear()
                procs.Add(seps[2])
            }
        } else {
            procs.Add(seps[2])
        }
        if seps[3] == "*" {
            if !insts.Contains(seps[3]) {
                insts.Clear()
                insts.Add(seps[3])
            }
        } else {
            insts.Add(seps[3])
        }
    }
    strs := []string{"*", "*", "*", "*"}
    if !words.Contains("*") {
        strs[0] = getProcStr(interSliceToStrSlice(words.ToSlice()))
    }
    if !zones.Contains("*") {
        strs[1] = getProcStr(interSliceToStrSlice(zones.ToSlice()))
    }
    if !procs.Contains("*") {
        strs[2] = getProcStr(interSliceToStrSlice(procs.ToSlice()))
    }
    if !insts.Contains("*") {
        strs[3] = getProcStr(interSliceToStrSlice(insts.ToSlice()))
    }
    return strings.Join(strs, ".")
}
