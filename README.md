# golang-heroku-log-drain [![GoDoc](https://godoc.org/github.com/tonimelisma/golang-heroku-log-drain?status.svg)](https://pkg.go.dev/mod/github.com/tonimelisma/golang-heroku-log-drain) [![Go Report Card](http://goreportcard.com/badge/tonimelisma/golang-heroku-log-drain)](http://goreportcard.com/report/tonimelisma/golang-heroku-log-drain) ![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/tonimelisma/golang-heroku-log-drain) ![License](https://img.shields.io/badge/license-MIT-blue.svg)
A super simple Heroku log drain written in Go. Store and view Heroku app logs on your local server.
It stores all received logs in a directory of your choosing, separating logs from heroku as well as from
each of your dynos in separate files, with ANSI coloring.

## Installation
1. Copy the repository
2. Copy ``.env_template`` to ``.env`` and change to your liking
3. Get an SSL certificate and template (e.g. from [https://letsencrypt.org](https://letsencrypt.org/))
4. Run the program on your server: ``go run main.go``
5. Ask Heroku to send copy of logs to drain: ``heroku drains:add -a herokuappname https://example.com:8443/log``


## Maintenance and development
The software works perfectly for my personal use cases so I don't expect there to be much activity in the repository
in the future. However, *the software is actively maintained* and I expect to answer any issues or PRs in a
reasonable timeframe.
## Caveats
- Fields such as the hostname and message facility are not stored in the logs as to my knowledge Heroku does not use these
- To view the coloring properly in ``less``, run it as ``less -r filename.log``
- Duplicate detection via the frame-ID header has not been implemented
- Directory and file permissions stored in ``.env`` are not used. The program defaults to 0755/0644 for directories/iles