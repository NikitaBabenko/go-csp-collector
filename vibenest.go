package main

import (
	"io"
	"log"
	"net"
	"net/http"
	"sync"
	"time"
)

const (
	reportBodyLimit   = 256 << 10
	reportConcurrency = 4
	reportConnections = 64
	reportBurst       = 8
	reportInterval    = 500 * time.Millisecond
	reportLogLimit    = 16 << 20
)

func configureVibeNestServer(server *http.Server) {
	server.Handler = limitReportRequests(server.Handler)
	server.MaxHeaderBytes = 16 << 10
	server.ReadHeaderTimeout = 3 * time.Second
	server.ReadTimeout = 10 * time.Second
	server.WriteTimeout = 15 * time.Second
	server.IdleTimeout = 15 * time.Second
	// The standard server error logger can include remote addresses or panic details.
	server.ErrorLog = log.New(io.Discard, "", 0)
}

// Bound accepted connections before net/http creates per-connection goroutines.
type reportListener struct {
	net.Listener
	inFlight chan struct{}
}

func (l *reportListener) Accept() (net.Conn, error) {
	for {
		conn, err := l.Listener.Accept()
		if err != nil {
			return nil, err
		}
		select {
		case l.inFlight <- struct{}{}:
			return &reportConn{Conn: conn, inFlight: l.inFlight}, nil
		default:
			_ = conn.Close()
		}
	}
}

type reportConn struct {
	net.Conn
	inFlight chan struct{}
	release  sync.Once
}

func (c *reportConn) Close() error {
	err := c.Conn.Close()
	c.release.Do(func() { <-c.inFlight })
	return err
}

func limitReportRequests(next http.Handler) http.Handler {
	inFlight := make(chan struct{}, reportConcurrency)
	var rate reportRate
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if (r.URL.Path == defaultHealthCheckPath || r.URL.Path == "/") && (r.Method == http.MethodGet || r.Method == http.MethodHead) {
			next.ServeHTTP(w, r)
			return
		}

		select {
		case inFlight <- struct{}{}:
			defer func() { <-inFlight }()
		default:
			rejectReportRequest(w, http.StatusTooManyRequests)
			return
		}

		if !rate.allow(time.Now()) {
			rejectReportRequest(w, http.StatusTooManyRequests)
			return
		}
		if r.Method == http.MethodPost {
			if r.ContentLength > reportBodyLimit {
				rejectReportRequest(w, http.StatusRequestEntityTooLarge)
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, reportBodyLimit)
		}

		next.ServeHTTP(w, r)
	})
}

func rejectReportRequest(w http.ResponseWriter, status int) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if status == http.StatusTooManyRequests {
		w.Header().Set("Retry-After", "1")
	}
	w.WriteHeader(status)
}

// A shared budget keeps admission independent of forwarded IP or other client inputs.
type reportRate struct {
	mu   sync.Mutex
	next time.Time
}

func (r *reportRate) allow(now time.Time) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.next.After(now.Add((reportBurst - 1) * reportInterval)) {
		return false
	}
	if r.next.Before(now) {
		r.next = now
	}
	r.next = r.next.Add(reportInterval)
	return true
}

type reportLogBudget struct {
	mu        sync.Mutex
	output    io.Writer
	written   int
	exhausted bool
}

func (b *reportLogBudget) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	const notice = "{\"level\":\"warning\",\"message\":\"report log budget exhausted; accepted reports continue to count in metrics\"}\n"
	if b.exhausted {
		return len(p), nil
	}
	if len(p) > reportLogLimit-len(notice)-b.written {
		b.exhausted = true
		_, err := io.WriteString(b.output, notice)
		return len(p), err
	}

	n, err := b.output.Write(p)
	b.written += n
	return n, err
}
