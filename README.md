# haminer

Library and program to parse and forward HAProxy logs.

Supported forwarder,

* Influxdb


## Requirements

* [Go](https://golang.org) for building from source code
* [git](https://git-scm.com/) for downloading source code
* [InfluxDB](https://portal.influxdata.com/downloads) for storing
  HAProxy log.
* [Chronograf](https://portal.influxdata.com/downloads) for viewing
  influxdb data with graph.


## Building

This steps assume that you already installed `Go`, `git`, and `influxdb`.

Get the source code using git,

	$ git clone git@github.com:shuLhan/haminer.git
	$ cd haminer
	$ make

The binary will be installed on `$GOPATH/bin/haminer`.


## Configuration

`haminer` by default will load it's config from `/etc/haminer.conf`, if not
specified when running the program.

See
`[haminer.conf](https://github.com/shuLhan/haminer/blob/master/cmd/haminer/haminer.conf)
for an example of possible configuration and their explanation.


## Installation

(1) Copy configuration from `$SOURCE/cmd/haminer/haminer/conf` to
`/etc/haminer.conf`

(2) Update haminer configuration in `/etc/haminer.conf`

(3) Update HAProxy config to forward log to UDP port other than rsyslog, for
example,

```
global
	...
	log                       127.0.0.1:5140 local3
	...
```

Then reload or restart HAProxy.

(4) Create user and database in Influxdb,

       $ influx
       > CREATE USER "haminer" WITH PASSWORD 'haminer'
       > CREATE DATABASE haminer
       > GRANT ALL ON haminer TO haminer

(5) Run the haminer program manually,

	$ $GOPATH/bin/haminer

or use a
[systemd service](https://github.com/shuLhan/haminer/blob/master/cmd/haminer/haminer.service).
