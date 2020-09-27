package main

import (
    "encoding/gob"
    "io"
    "os"
    "sync"
)

type Store struct {
    data record
    mu   sync.RWMutex
    file *os.File
}

type record struct {
    UUID string
    Jwt  string
}

func NewStore(filename string) *Store {
    s := &Store{data: record{}}
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

func (s *Store) GetUUID() string {
    s.mu.RLock()
    defer s.mu.RUnlock()
    return s.data.UUID
}

func (s *Store) SetUUID(uuid string) {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.data.UUID = uuid
}

func (s *Store) GetJWT() string {
    s.mu.RLock()
    defer s.mu.RUnlock()
    return s.data.Jwt
}

func (s *Store) SetJWT(jwt string) {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.data.Jwt = jwt
}

func (s *Store) load() error {
    if _, err := s.file.Seek(0, 0); err != nil {
        return err
    }
    d := gob.NewDecoder(s.file)
    var err error
    var r record
    if err = d.Decode(&r); err == nil {
        s.data.UUID = r.UUID
        s.data.Jwt = r.Jwt
    }
    if err == io.EOF {
        return nil
    }
    return err
}

func (s *Store) Save() error {
    e := gob.NewEncoder(s.file)
    return e.Encode(s.data)
}
