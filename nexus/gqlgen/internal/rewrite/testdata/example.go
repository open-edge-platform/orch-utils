// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package testdata

import (
	"fmt"

	lol "bytes"
)

type Foo struct {
	Field int
}

func (m *Foo) Method(arg int) {
	// leading comment

	// field comment
	m.Field++

	// trailing comment
}

func (m *Foo) String() string {
	var buf lol.Buffer
	buf.WriteString(fmt.Sprintf("%d", m.Field))
	return buf.String()
}
