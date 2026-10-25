# urlcheck

A small command-line tool that checks whether URLs are reachable.
Checks run concurrently with a fixed-size worker pool, every request has
a time limit, and the results are printed in the order the URLs were typed.

## Build

```
go build -o urlcheck.exe
```

## Usage

```
urlcheck [-workers N] [-timeout 5s] <url>...
```

| Option | Default | Meaning |
|---|---|---|
| `-workers` | `4` | How many checks run at the same time |
| `-timeout` | `5s` | Time limit for each request |

Example:

```
urlcheck.exe -timeout 2s https://example.com https://example.com/this-page-does-not-exist https://no-such-host.invalid https://httpbin.org/delay/10
OK 200 https://example.com 171.2445ms
FAIL 404 https://example.com/this-page-does-not-exist 168.4213ms
ERROR https://no-such-host.invalid Get "https://no-such-host.invalid": context deadline exceeded
ERROR https://httpbin.org/delay/10 Get "https://httpbin.org/delay/10": context deadline exceeded
4 checked, 1 OK, 1 failed, 2 error
```

Timings will differ on your machine.

## What the results mean

- **OK**: the server replied with a 2xx status.
- **FAIL**: the server replied, but the status was not 2xx (for example 404 or 500).
- **ERROR**: no reply arrived (unknown host, connection refused, or the time limit ran out), so there is no status, only a reason.

A short `-timeout` can turn a slow name lookup into a timeout error, as in the
example above.

## Exit codes

- `0`: every URL was OK.
- `1`: at least one URL was FAIL or ERROR. All URLs are still checked.
- `2`: usage error (no URLs, or `-workers` below 1, or `-timeout` of zero or less).

```
urlcheck.exe -workers 0 https://example.com
usage: urlcheck [-workers N] [-timeout 5s] <url>...
```

## How it works

- A feeder goroutine puts one job (index and URL) per URL on a channel.
- A fixed number of workers take jobs, check them, and send results back.
- Each request gets its own timeout through the context, so one dead server
  cannot hang the tool.
- Each result carries its index, so `main` places it in the right slot and
  prints in the typed order, however the checks finish.
- Only `main` collects the results, so there is no shared state or mutex.

## Tests

```
go test -v -race
```

- `classify` is tested on its boundaries (199, 200, 299, 300) and on the case
  where an error and a 200 status appear together.
- `checkURL` is tested against fake servers (`httptest`) for 200, 404, 500,
  and a timeout.
- `runChecks` is tested with a slow URL first and a fast one second, to check
  the output order.

## Limitations

- A URL typed without `https://` gives an unsupported-scheme ERROR.
- There is no graceful stop on Ctrl+C.