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
    "testing"
    "time"
    "log"
)

func TestPopTimeout(t *testing.T) {
    maxActiveNum := 1
    pool := NewConnectionPool(maxActiveNum,
        0,
        time.Second*time.Duration(1),
        3,
        func() (interface{}, int, error) {
            return true, 0, nil
        },
        func(c interface{}, id int) {
        },
    )

    _, err := pool.Pop()
    if err != nil {
        log.Printf(err.Error())
    }

    log.Println("pop ok")
    _, err = pool.Pop()
    if err != nil {
        log.Printf(err.Error())
    }

}