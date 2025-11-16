package main

import (
	"bufio"
	"errors"
	"net"
	"net/http"
	"testing"
)

type mockResponseWriter struct {
	header   http.Header
	hijacked bool
	flushed  bool
}

func newMockResponseWriter() *mockResponseWriter {
	return &mockResponseWriter{header: make(http.Header)}
}

func (m *mockResponseWriter) Header() http.Header         { return m.header }
func (m *mockResponseWriter) Write(b []byte) (int, error) { return len(b), nil }
func (m *mockResponseWriter) WriteHeader(statusCode int)  {}
func (m *mockResponseWriter) Flush()                      { m.flushed = true }
func (m *mockResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	m.hijacked = true
	return nil, nil, nil
}

func TestStatusWriterHijackDelegates(t *testing.T) {
	rw := newMockResponseWriter()
	sw := &statusWriter{ResponseWriter: rw}

	if _, _, err := sw.Hijack(); err != nil {
		t.Fatalf("Hijack returned error: %v", err)
	}
	if !rw.hijacked {
		t.Fatalf("expected underlying hijacker to be invoked")
	}
}

func TestStatusWriterFlushDelegates(t *testing.T) {
	rw := newMockResponseWriter()
	sw := &statusWriter{ResponseWriter: rw}

	sw.Flush()
	if !rw.flushed {
		t.Fatalf("expected Flush to delegate")
	}
}

func TestStatusWriterPushWithoutSupport(t *testing.T) {
	recorder := basicResponseWriter{}
	sw := &statusWriter{ResponseWriter: recorder}

	if err := sw.Push("/", nil); !errors.Is(err, http.ErrNotSupported) {
		t.Fatalf("expected ErrNotSupported, got %v", err)
	}
}

type basicResponseWriter struct{}

func (basicResponseWriter) Header() http.Header       { return http.Header{} }
func (basicResponseWriter) Write([]byte) (int, error) { return 0, nil }
func (basicResponseWriter) WriteHeader(int)           {}
