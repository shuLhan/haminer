<!--
SPDX-FileCopyrightText: 2019 M. Shulhan <ms@kilabit.info>

SPDX-License-Identifier: GPL-3.0-or-later
-->

# haminer

`haminer` is a library and program to parse and forward HAProxy HTTP logs.

The HTTP logs is HTTP request that received by HAProxy frontend and forwarded
to backend.
In default format, it looks like these (split into multi lines, for
readability):

```
<158>Sep  4 17:08:47 haproxy[109530]: 185.83.144.103:46376
  [04/Sep/2022:17:08:47.264] www~ be_kilabit/kilabit-0.0/0/1/2/3 200 89 - -
  ---- 5/5/0/0/0 0/0 "GET / HTTP/1.1"
```

See
[HTTP log format documentation](https://www.haproxy.com/documentation/hapee/1-8r1/onepage/#8.2.3)
for more information.

Currently, there are supported database where haminer can forward the
parsed log: Influxdb, Questdb, and Postgresql.
Haminer support Influxdb v1 and v2.

```
 +---------+  UDP  +---------+      +-----------+
 | HAProxy |------>| haminer |----->| Influxdb  |
 +---------+       +---------+      | / Questdb |
                                    +-----------+
```

In Influxdb, the log are stored as measurement called `haproxy`.
In Questdb, the log are stored as table called `haproxy`.

The following fields are stored as tags (in Influxdb) or symbol (in Questdb):
host, server, backend, frontend, http_method, http_url, http_query,
http_proto, http_status, term_state, client_ip, client_port.

And the following fields are stored as fields (in Influxdb) or values (in
Questdb): time_req, time_wait, time_connect, time_rsp, time_all,
conn_active, conn_frontend, conn_backend, conn_server, conn_retries,
queue_server, queue_backend, bytes_read.

Once the log has been accumulated, we can query the data.
For example, with Questdb we can count each visited URL using the following
query,

```
select backend, http_url, count(*) as visit from 'haproxy'
group by backend, http_url
order by visit desc;
```

## Installation

### Building from source

Requirements,

- [Go](https://golang.org) for building the source code
- [git](https://git-scm.com) for downloading the source code

Get the source code using git,

```
$ git clone https://git.sr.ht/~shulhan/haminer
$ cd haminer
$ make
```

The binary name is `haminer` build in the current directory.


### Pre-build package

The Arch Linux package is available at build.kilabit.info.
Add the following repository to your `pacman.conf`,

```
[build.kilabit.info]
Server = https://build.kilabit.info/aur
```

To install it,

	$ sudo pacman -Sy --noconfirm haminer-git


## Configuration

haminer by default will load it's config from `/etc/haminer.conf`, if not
specified when running the program.

See
[haminer.conf](https://git.sr.ht/~shulhan/haminer/tree/main/item/cmd/haminer/haminer.conf)
for an example of possible configuration and their explanation.


### Forwarders

Currently, there are several database where haminer can forward the parsed
log: Influxdb,  Questdb, and Postgresql.
Haminer support Influxdb v1 and v2.

#### Influxdb v1

For v1, you need to create the user and database first,

```
$ influx
> CREATE USER "haminer" WITH PASSWORD 'haminer'
> CREATE DATABASE haminer
> GRANT ALL ON haminer TO haminer
```

Example of forwarder configuration,

```
[forwarder "influxd"]
version = v1
url = http://127.0.0.1:8086
bucket = haminer
user = haminer
password  = haminer
```

#### Influxdb v2

For v2,

```
$ sudo influx bucket create \
	--name haminer \
	--retention 30d
```

For v2, the example configuration is

```
[forwarder "influxd"]
version = v2
url = http://127.0.0.1:8086
org = $org
bucket = haminer
token = $token
```

#### Questdb

For Questdb the configuration is quite simple,

```
[forwarder "questdb"]
url = udp://127.0.0.1:9009
```

We did not need to create the table, Questdb will handled that automatically.

#### Postgresql

For Postgresql, you need to create the user and database first, for example,

```
postgres$ psql
postgres=> CREATE ROLE haminer PASSWORD 'haminer' CREATEDB INHERIT LOGIN;
postgres=> CREATE DATABASE haminer OWNER haminer;
postgres=> \q
```

The configuration only need the Data Source Name (DSN),

```
[forwarder "postgresql"]
url = postgres://<user>:<pass>@<host>/<database>?sslmode=<require|verify-full|verify-ca|disable>
```


## Deployment

Copy configuration from `$SOURCE/cmd/haminer/haminer/conf` to
`/etc/haminer.conf`

Update haminer configuration in `/etc/haminer.conf`.
For example,

```
[haminer]
listen = 127.0.0.1:5140

...
```

Add one or more provider to the configuration as the example above.

Update HAProxy config to forward log to UDP port other than rsyslog.
For example,

```
global
	...
	log 127.0.0.1:5140 local3
	...
```

Then reload or restart HAProxy.

Run the haminer program,

```
$ haminer
```

or use a
[systemd service](https://git.sr.ht/~shulhan/haminer/tree/main/item/cmd/haminer/haminer.service).

```
$ sudo systemctl enable haminer
$ sudo systemctl start  haminer
```


##  Development

<https://git.sr.ht/~shulhan/haminer>:: Link to the source code.

<https://lists.sr.ht/~shulhan/haminer>:: Link to development
and discussion.

<https://todo.sr.ht/~shulhan/haminer>:: Link to submit an issue,
feedback, or request for new feature.

[Changelog](https://kilabit.info/project/haminer/CHANGELOG.html):: History
of each release.


##  License

Copyright (C) 2018-2025 M. Shulhan &lt;ms@kilabit.info&gt;

This program is free software: you can redistribute it and/or modify it under
the terms of the GNU General Public License as published by the Free Software
Foundation, either version 3 of the License, or any later version.

This program is distributed in the hope that it will be useful, but WITHOUT
ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS
FOR A PARTICULAR PURPOSE.  See the GNU General Public License for more
details.

You should have received a copy of the GNU General Public License along with
this program.
If not, see <http://www.gnu.org/licenses/>.
