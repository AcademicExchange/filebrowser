package http

import (
    "sync"
)

var mtx sync.Mutex

const duration int64 = 60 * 60 // valid period: 1 hour
const cap int = 1              // capacity
var cache = NewExpiredMap(cap) // cache data
