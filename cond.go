/**
 * Copyright 2017  authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License"): you may
 * not use this file except in compliance with the License. You may obtain
 * a copy of the License at
 *
 *     http: *www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
 * WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
 * License for the specific language governing permissions and limitations
 * under the License.
 */

// Created by xuning on 2017/2/17

package connpool

import (
    "sync"
    "time"
)

type TMOCond struct {
    L    sync.Locker
    ch      chan bool
}

func NewTMOCond(l sync.Locker) *TMOCond {
    return &TMOCond{ ch: make(chan bool), L: l }
}

func (t *TMOCond) Wait() {
    t.L.Unlock()
    <-t.ch
    t.L.Lock()
}

func (t *TMOCond) WaitOrTimeout(d time.Duration) bool {
    tmo := time.NewTimer(d)
    t.L.Unlock()
    var r bool
    select {
    case <-tmo.C:
        r = false
    case <-t.ch:
        r = true
    }
    if !tmo.Stop() {
        select {
        case <- tmo.C:
        default:
        }
    }
    t.L.Lock()
    return r
}

func (t *TMOCond) Signal() {
    t.signal()
}

func (t *TMOCond) Broadcast() {
    for {
        // Stop when we run out of waiters
        //
        if !t.signal() {
            return
        }
    }
}

func (t *TMOCond) signal() bool {
    select {
    case t.ch <- true:
        return true
    default:
        return false
    }
}
