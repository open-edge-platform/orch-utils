// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package testdata

import (
	a "fmt"
	"time"
	_ "underscore"
)

type foo struct {
	Time time.Time `json:"text"`
}

func fn() {
	a.Println("hello")
}

type Message struct {
	ID        string    `json:"id"`
	Text      string    `json:"text"`
	CreatedBy string    `json:"createdBy"`
	CreatedAt time.Time `json:"createdAt"`
}
