package http

import (
	"bytes"
	"fmt"
	"sync"
	"time"

	mapset "github.com/deckarep/golang-set"
	libErrors "github.com/filebrowser/filebrowser/v2/errors"
)

type CacheData struct {
	bakdir string
	xmls   mapset.Set
	dbs    mapset.Set
	svrs   mapset.Set
}

type ConfigType int

const (
	ConfigDB  ConfigType = 0
	ConfigXML ConfigType = 1
	ConfigSVR ConfigType = 2
)

func newCacheData() *CacheData {
	c := CacheData{
		bakdir: "",
		xmls:   mapset.NewSet(),
		dbs:    mapset.NewSet(),
		svrs:   mapset.NewSet(),
	}
	return &c
}

// only for debug
func (c *CacheData) String() string {
	return fmt.Sprintf("{ bakdir:%s, xml_config:%v, dbs_config:%v, svr_config:%v }", c.bakdir, c.xmls, c.dbs, c.svrs)
}

type val struct {
	data        map[string]*CacheData
	expiredTime int64
}

type ExpiredMap struct {
	m       map[string]*val
	cap     int
	timeMap map[int64][]string
	mtx     *sync.Mutex
	stop    chan struct{}
}

func NewExpiredMap(c int) *ExpiredMap {
	e := ExpiredMap{
		m:       make(map[string]*val),
		cap:     c,
		mtx:     new(sync.Mutex),
		timeMap: make(map[int64][]string),
		stop:    make(chan struct{}),
	}
	go e.run(time.Now().Unix())
	return &e
}

type delMsg struct {
	keys []string
	t    int64
}

const delChannelCap = 100

func (e *ExpiredMap) run(now int64) {
	// trigger per second
	t := time.NewTicker(time.Second * 1)
	delCh := make(chan *delMsg, delChannelCap)
	go func() {
		for v := range delCh {
			e.MDel(v.keys, v.t)
		}
	}()
	for {
		select {
		case <-t.C:
			now++
			if keys, found := e.timeMap[now]; found {
				delCh <- &delMsg{keys: keys, t: now}
			}
		case <-e.stop:
			close(delCh)
			return
		}
	}
}

func (e *ExpiredMap) Set(key string, value map[string]*CacheData, ttl int64) bool {
	if ttl <= 0 {
		return false
	}
	e.mtx.Lock()
	defer e.mtx.Unlock()
	if len(e.m) >= e.cap {
		return false
	}
	expiredTime := time.Now().Unix() + ttl
	e.m[key] = &val{
		data:        value,
		expiredTime: expiredTime,
	}
	e.timeMap[expiredTime] = append(e.timeMap[expiredTime], key)
	return true
}

func (e *ExpiredMap) Get(key string) (found bool, value map[string]*CacheData) {
	e.mtx.Lock()
	defer e.mtx.Unlock()
	if found = e.isKeyExisted(key); !found {
		return
	}
	value = e.m[key].data
	return
}

func (e *ExpiredMap) Del(key string) {
	e.mtx.Lock()
	delete(e.m, key)
	e.mtx.Unlock()
}

func (e *ExpiredMap) MDel(keys []string, t int64) {
	e.mtx.Lock()
	defer e.mtx.Unlock()
	delete(e.timeMap, t)
	for _, key := range keys {
		delete(e.m, key)
	}
}

func (e *ExpiredMap) Size() int {
	e.mtx.Lock()
	defer e.mtx.Unlock()
	return len(e.m)
}

func (e *ExpiredMap) TTL(key string) int64 {
	e.mtx.Lock()
	defer e.mtx.Unlock()
	if !e.isKeyExisted(key) {
		return -1
	}
	return e.m[key].expiredTime - time.Now().Unix()
}

func (e *ExpiredMap) Clear() {
	e.mtx.Lock()
	defer e.mtx.Unlock()
	e.m = make(map[string]*val)
	e.timeMap = make(map[int64][]string)
}

func (e *ExpiredMap) Close() {
	e.mtx.Lock()
	defer e.mtx.Unlock()
	e.stop <- struct{}{}
}

func (e *ExpiredMap) IsDirCacheExisted(key string, dir string) bool {
	if key == "" || dir == "" {
		return false
	}
	e.mtx.Lock()
	defer e.mtx.Unlock()
	value, found := e.m[key]
	if !found {
		return false
	}
	if _, found := value.data[dir]; !found {
		return false
	}
	return true
}

func (e *ExpiredMap) SetCacheData(key string, dir string, c *CacheData) bool {
	if key == "" || dir == "" || c == nil {
		return false
	}
	e.mtx.Lock()
	defer e.mtx.Unlock()
	value, found := e.m[key]
	if !found {
		return false
	}
	if _, found := value.data[dir]; found {
		return false
	}
	value.data[dir] = c
	return true
}

func (e *ExpiredMap) SetBakDir(key string, dir string, bak string) bool {
	if key == "" || dir == "" || bak == "" {
		return false
	}
	e.mtx.Lock()
	defer e.mtx.Unlock()
	value, found := e.m[key]
	if !found {
		return false
	}
	cd, found := value.data[dir]
	if !found {
		return false
	}

	if cd == nil {
		return false
	}
	cd.bakdir = bak
	return true
}

func (e *ExpiredMap) GetBakDir(key string, dir string) string {
	if key == "" || dir == "" {
		return ""
	}
	e.mtx.Lock()
	defer e.mtx.Unlock()
	value, found := e.m[key]
	if !found {
		return ""
	}
	cd, found := value.data[dir]
	if !found {
		return ""
	}
	if cd == nil {
		return ""
	}
	return cd.bakdir
}

func (e *ExpiredMap) AddConfig(key string, dir string, cfg string, t ConfigType) error {
	if key == "" || dir == "" || cfg == "" {
		return libErrors.ErrCacheFailed
	}
	e.mtx.Lock()
	defer e.mtx.Unlock()
	value, found := e.m[key]
	if !found {
		return libErrors.ErrCacheFailed
	}
	cd, found := value.data[dir]
	if !found {
		return libErrors.ErrCacheFailed
	}
	switch t {
	case ConfigDB:
		cd.dbs.Add(cfg)
	case ConfigXML:
		cd.xmls.Add(cfg)
	case ConfigSVR:
		cd.svrs.Add(cfg)
	default:
		return libErrors.ErrCacheFailed
	}
	return nil
}

func (e *ExpiredMap) RemoveConfig(key string, dir string, cfg string, t ConfigType) error {
	if key == "" || dir == "" || cfg == "" {
		return libErrors.ErrCacheFailed
	}
	e.mtx.Lock()
	defer e.mtx.Unlock()
	value, found := e.m[key]
	if !found {
		return libErrors.ErrCacheFailed
	}
	cd, found := value.data[dir]
	if !found {
		return libErrors.ErrCacheFailed
	}
	switch t {
	case ConfigDB:
		if cd.dbs.Contains(cfg) {
			cd.dbs.Remove(cfg)
		}
	case ConfigXML:
		if cd.xmls.Contains(cfg) {
			cd.xmls.Remove(cfg)
		}
	case ConfigSVR:
		if cd.svrs.Contains(cfg) {
			cd.svrs.Remove(cfg)
		}
	default:
		return libErrors.ErrCacheFailed
	}
	return nil
}

// Not locked, only for internal use
func (e *ExpiredMap) isKeyExisted(key string) bool {
	if val, found := e.m[key]; found {
		if val.expiredTime <= time.Now().Unix() {
			delete(e.m, key)
			delete(e.timeMap, val.expiredTime)
			return false
		}
		return true
	}
	return false
}

// External use, need to be locked
func (e *ExpiredMap) IsKeyExisted(key string) bool {
	e.mtx.Lock()
	defer e.mtx.Unlock()
	if val, found := e.m[key]; found {
		if val.expiredTime <= time.Now().Unix() {
			delete(e.m, key)
			delete(e.timeMap, val.expiredTime)
			return false
		}
		return true
	}
	return false
}

// only for debug
func (e *ExpiredMap) String() string {
	e.mtx.Lock()
    defer e.mtx.Unlock()
    var buf bytes.Buffer
    buf.WriteString("{ ")
	for k, v := range e.m {
		if !e.isKeyExisted(k) {
			continue
        }
        buf.WriteString(fmt.Sprintf("{ %s:{ ", k))
		for k1, v1 := range v.data {
            buf.WriteString(fmt.Sprintf("{ dir:%s, cache:%v },", k1, v1))
        }
        buf.WriteString(" }")
    }
    buf.WriteString(" }")
    return buf.String()
}
