package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCheckURL(t *testing.T) {
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		timeout    time.Duration
		wantStatus int
		wantErr    bool
	}{
		{name: "OK status", handler: func(w http.ResponseWriter, r *http.Request) {}, timeout: 3 * time.Second, wantStatus: 200, wantErr: false},
		{name: "Fail status", handler: func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(404) }, timeout: 3 * time.Second, wantStatus: 404, wantErr: false},
		{name: "Fail status 500", handler: func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(500) }, timeout: 3 * time.Second, wantStatus: 500, wantErr: false},
		{name: "Error status", handler: func(w http.ResponseWriter, r *http.Request) { time.Sleep(300 * time.Millisecond) }, timeout: 50 * time.Millisecond, wantStatus: 0, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(tt.handler)
			defer server.Close()
			got := checkURL(context.Background(), &http.Client{}, server.URL, tt.timeout)

			if (got.err != nil) != tt.wantErr {
				t.Errorf("checkURL(%q) = %v, want %v", tt.name, got.err, tt.wantErr)
			}

			if got.status != tt.wantStatus {
				t.Errorf("checkURL(%q) = %v, want %v", tt.name, got.status, tt.wantStatus)
			}
		})
	}
}

func TestRunChecksOrder(t *testing.T) {
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(201)
	}))
	defer slow.Close()

	fast := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(202)
	}))
	defer fast.Close()

	urls := []string{
		slow.URL, fast.URL,
	}

	got := runChecks(context.Background(), &http.Client{}, urls, 2, 5*time.Second)

	if len(got) != 2 {
		t.Fatalf("error occurred: got %v results, want 2", len(got))
	}

	if got[0].status != 201 {
		t.Errorf("error: expected %v, got %v", 201, got[0].status)
	}

	if got[1].status != 202 {
		t.Errorf("error: expected %v, got %v", 202, got[1].status)
	}
}

func TestClassify(t *testing.T) {
	tests := []struct  {
		name string
		r result
		want string
	}{
		{name: "OK status 200", r: result{status: 200}, want: "OK"}, 
		{name: "OK status 299", r: result{status: 299}, want: "OK"}, 
		{name: "Fail status 199", r: result{status: 199}, want: "FAIL"}, 
		{name: "Fail status 300", r: result{status: 300}, want: "FAIL"}, 
		{name: "Fail status 404", r: result{status: 404}, want: "FAIL"}, 
		{name: "Error status", r: result{err: errors.New("boom")}, want: "ERROR"}, 
		{name: "Error with status 200", r: result{status: 200, err: errors.New("boom")}, want: "ERROR"}, 
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classify(tt.r)

			if got != tt.want {
				t.Errorf("classify(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}