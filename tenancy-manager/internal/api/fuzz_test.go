// SPDX-FileCopyrightText: 2026 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"fmt"
	"net/url"
	"strings"
	"testing"
)

// newFuzzHandler returns a Handler backed by a no-op mockStore.
// The mockStore has no mutable state, so this handler is safe to share across
// fuzz iterations.
func newFuzzHandler() *Handler {
	return newTestHandler(&mockStore{})
}

// assertValidStatus fails the test if code is outside the legal HTTP range.
func assertValidStatus(t *testing.T, code int, label string) {
	t.Helper()
	if code < 100 || code > 599 {
		t.Fatalf("invalid HTTP status %d for %s", code, label)
	}
}

// FuzzDecodeBody feeds arbitrary bytes as the JSON body of org and project
// mutation endpoints.  The invariant is that no input must cause a panic and
// every response must carry a legal HTTP status code.
func FuzzDecodeBody(f *testing.F) {
	f.Add(``)
	f.Add(`{}`)
	f.Add(`{"description":"hello"}`)
	f.Add(`{"description":` + strings.Repeat("x", 70000) + `}`) // exceeds 64 KB limit
	f.Add(`not json at all`)
	f.Add(`{"description":null}`)
	f.Add("\x00\x01\x02\x03")
	f.Add(`{"description":"` + strings.Repeat("\xff", 100) + `"}`)

	h := newFuzzHandler()
	f.Fuzz(func(t *testing.T, body string) {
		for _, path := range []string{
			"/v1/orgs/fuzz-org",
			"/v1/projects/fuzz-proj?org=fuzz-org",
		} {
			rr := do(h, "PUT", path, body)
			assertValidStatus(t, rr.Code, "PUT "+path)
		}
	})
}

// FuzzEventQueryParams feeds arbitrary values for the ?controller, ?after, and
// ?limit query parameters on the events endpoint, which parses and
// bounds-checks each one.  The invariant is that no combination may panic.
func FuzzEventQueryParams(f *testing.F) {
	// Valid cases
	f.Add("ctrl", int64(0), 100)
	f.Add("ctrl", int64(10), 50)
	// Boundary violations the handler rejects
	f.Add("ctrl", int64(-1), 50)
	f.Add("ctrl", int64(0), 0)
	f.Add("ctrl", int64(0), 1001)
	// Edge values
	f.Add("", int64(0), 1)
	f.Add("ctrl", int64(1<<62), 100)

	h := newFuzzHandler()
	f.Fuzz(func(t *testing.T, controller string, after int64, limit int) {
		path := fmt.Sprintf("/v1/events?controller=%s&after=%d&limit=%d",
			url.QueryEscape(controller), after, limit)
		rr := do(h, "GET", path, "")
		assertValidStatus(t, rr.Code, "GET "+path)
	})
}

// FuzzOrgRouting exercises the org endpoints (GET, PUT, DELETE) with arbitrary
// names in the URL path.  Names are percent-encoded so the fuzzer explores
// handler logic rather than URL-parsing edge cases.
func FuzzOrgRouting(f *testing.F) {
	f.Add("my-org")
	f.Add("")
	f.Add("org with spaces")
	f.Add("'; DROP TABLE orgs; --")
	f.Add(strings.Repeat("a", 256))
	f.Add("../../../etc/passwd")
	f.Add("\x00null\x00byte")

	h := newFuzzHandler()
	f.Fuzz(func(t *testing.T, name string) {
		encoded := url.PathEscape(name)
		for _, tc := range []struct{ method, body string }{
			{"GET", ""},
			{"PUT", `{"description":"fuzz"}`},
			{"DELETE", ""},
		} {
			path := "/v1/orgs/" + encoded
			rr := do(h, tc.method, path, tc.body)
			assertValidStatus(t, rr.Code, tc.method+" "+path)
		}
	})
}

// FuzzProjectRouting exercises the project endpoints with arbitrary name and
// org query-parameter values.
func FuzzProjectRouting(f *testing.F) {
	f.Add("my-proj", "my-org")
	f.Add("", "")
	f.Add("proj", "'; DROP TABLE orgs; --")
	f.Add(strings.Repeat("p", 256), "org")
	f.Add("proj", strings.Repeat("o", 256))

	h := newFuzzHandler()
	f.Fuzz(func(t *testing.T, name, org string) {
		encodedName := url.PathEscape(name)
		encodedOrg := url.QueryEscape(org)
		for _, tc := range []struct{ method, body string }{
			{"GET", ""},
			{"PUT", `{"description":"fuzz"}`},
			{"DELETE", ""},
		} {
			path := "/v1/projects/" + encodedName + "?org=" + encodedOrg
			rr := do(h, tc.method, path, tc.body)
			assertValidStatus(t, rr.Code, tc.method+" "+path)
		}
	})
}
