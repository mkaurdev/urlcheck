package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"
)

type job struct {
	index int
	url   string
}

type result struct {
	index    int
	status   int
	duration time.Duration
	err      error
}

func checkURL(ctx context.Context, client *http.Client, url string, timeout time.Duration) result {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return result{duration: 0, err: err}
	}

	start := time.Now()
	resp, err := client.Do(req)
	elapsed := time.Since(start)

	if err != nil {
		return result{duration: elapsed, err: err}
	}
	defer resp.Body.Close()

	return result{status: resp.StatusCode, duration: elapsed, err: err}
}

func worker(ctx context.Context, client *http.Client, jobs <-chan job, results chan<- result, wg *sync.WaitGroup, timeout time.Duration) {
	defer wg.Done()
	for j := range jobs {
		got := checkURL(ctx, client, j.url, timeout)
		results <- result{index: j.index, status: got.status, duration: got.duration, err: got.err}
	}
}

func classify(r result) string {
	if r.err != nil {
		return "ERROR"
	}
	if r.status < 200 || r.status > 299 {
		return "FAIL"
	}
	return "OK"
}

func runChecks(ctx context.Context, client *http.Client, urls []string, workers int, timeout time.Duration) []result {
	jobs := make(chan job)
	results := make(chan result)

	var wg sync.WaitGroup

	for range workers {
		wg.Add(1)
		go worker(ctx, client, jobs, results, &wg, timeout)
	}

	go func() {
		defer close(jobs)
		for i, u := range urls {
			jobs <- job{index: i, url: u}
		}
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	ordered := make([]result, len(urls))

	for r := range results {
		ordered[r.index] = r
	}

	return ordered
}

func main() {
	ctx := context.Background()

	workers := flag.Int("workers", 4, "how many checks run at the same time")
	timeout := flag.Duration("timeout", 5*time.Second, "time limit on every request")

	flag.Parse()

	if *workers < 1 || *timeout <= 0 {
		fmt.Fprintln(os.Stderr, "usage: urlcheck [-workers N] [-timeout 5s] <url>...")
		os.Exit(2)
	}

	urls := flag.Args()

	if len(urls) == 0 {
		fmt.Fprintln(os.Stderr, "usage: urlcheck [-workers N] [-timeout 5s] <url>...")
		os.Exit(2)
	}

	client := &http.Client{}

	ok := 0
	failed := 0
	errs := 0

	ordered := runChecks(ctx, client, urls, *workers, *timeout)

	for i, r := range ordered {
		switch classify(r) {
		case "ERROR":
			fmt.Println("ERROR", urls[i], r.err)
			errs++

		case "FAIL":
			fmt.Println("FAIL", r.status, urls[i], r.duration)
			failed++

		case "OK":
			fmt.Println("OK", r.status, urls[i], r.duration)
			ok++
		}
	}

	fmt.Printf("%v checked, %v OK, %v failed, %v error\n", len(urls), ok, failed, errs)

	if failed > 0 || errs > 0 {
		os.Exit(1)
	}
}
