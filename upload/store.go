package main

import (
    "encoding/gob"
    "io"
    "os"
    "sync"
)

type Store struct {
    data map[string]map[string]*meta
    mu   sync.RWMutex
    file *os.File
}

type meta struct {
    Url  string
    Uuid string
    Jwt  string
}

type record struct {
    M map[string]map[string]*meta
}

func NewStore(filename string) *Store {
    s := &Store{data: make(map[string]map[string]*meta)}
    f, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
    if err != nil {
        log.Fatalf("open store failed:", err)
    }
    s.file = f
    if err := s.load(); err != nil {
        log.Fatalf("load store failed:", err)
    }
    os.Truncate(filename, 0)
    return s
}

func (s *Store) GetUuid(env, url string) string {
    s.mu.RLock()
    defer s.mu.RUnlock()
    if _, found := s.data[env]; !found {
        s.data[env] = make(map[string]*meta)
    }
    if _, found := s.data[env][url]; !found {
        s.data[env][url] = &meta{}
    }
    return s.data[env][url].Uuid
}

func (s *Store) SetUuid(env, url, uuid string) {
    s.mu.Lock()
    defer s.mu.Unlock()
    if _, found := s.data[env]; !found {
        s.data[env] = make(map[string]*meta)
    }
    if _, found := s.data[env][url]; !found {
        s.data[env][url] = &meta{}
    }
    s.data[env][url].Uuid = uuid
}

func (s *Store) GetJwt(env, url string) string {
    s.mu.RLock()
    defer s.mu.RUnlock()
    if _, found := s.data[env]; !found {
        s.data[env] = make(map[string]*meta)
    }
    if _, found := s.data[env][url]; !found {
        s.data[env][url] = &meta{}
    }
    return s.data[env][url].Jwt
}

func (s *Store) SetJwt(env, url, jwt string) {
    s.mu.Lock()
    defer s.mu.Unlock()
    if _, found := s.data[env]; !found {
        s.data[env] = make(map[string]*meta)
    }
    if _, found := s.data[env][url]; !found {
        s.data[env][url] = &meta{}
    }
    s.data[env][url].Jwt = jwt
}

func (s *Store) load() error {
    if _, err := s.file.Seek(0, 0); err != nil {
        return err
    }
    d := gob.NewDecoder(s.file)
    if err := d.Decode(&s.data); err != nil {
        if err == io.EOF {
            return nil
        }
        return err
    }
    return nil
}

func (s *Store) Save() error {
    e := gob.NewEncoder(s.file)
    return e.Encode(s.data)
}
