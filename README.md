# haminer

Library and program to parse and forward HAProxy logs.

Supported forwarder,

* Influxdb


## Requirements

* [[ https://golang.org | Go ]] for building from source code
* [[ https://github.com/alecthomas/gometalinter | gometalinter ]] (optional)
* [[ https://git-scm.com/ | git ]] for downloading source code
* [[ https://portal.influxdata.com/downloads | Influxdb ]] for storing
  HAProxy log.
* [[ https://portal.influxdata.com/downloads | Chronograf ]] for viewing
  influxdb data with graph.

## Building

This steps assume that you already installed `Go`, `git`, `gometalinter`, and
`influxdb`.

Get the source code using git,

	$ git clone git@github.com:shuLhan/haminer.git
	$ make

The binary will be installed on `$GOPATH/bin/haminer`.


## Configuration

`haminer` by default will load it's config from `/etc/haminer.conf`, if not
specified when running the program.

See `cmd/haminer/haminer.conf` for an example of possible configuration.


## Installation

(1) Copy configuration from `$SOURCE/cmd/haminer/haminer/conf` to
`/etc/haminer.conf`

(2) Update haminer configuration in `/etc/haminer.conf`

(3) Update HAProxy config to forward log to UDP port other than rsyslog, for
example,

```
global
	...
	log                       127.0.0.1:5140 haminer
	...
```

Then reload or restart HAProxy.

(4) Create user and database in Influxdb,

       $ influx
       > CREATE USER "haminer" WITH PASSWORD 'haminer'
       > CREATE DATABASE haminer
       > GRANT ALL ON haminer TO haminer

## Running

Run the haminer program manually,

	$ $GOPATH/bin/haminer
