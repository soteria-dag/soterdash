soterdash
===

[![ISC License](http://img.shields.io/badge/license-ISC-blue.svg)](http://copyfree.org)

Soterdash is a web app that provides a dashboard for a set of soterd nodes. It's intended for use with Soteria's testnet.

## Requirements

[Go](http://golang.org) 1.11.1 or newer

[graphviz](http://graphviz.org/), for DAG rendering functionality

## Installation

### Build from source

* Install Go according to the [installation instructions](http://golang.org/doc/install).
* Ensure Go was installed properly and is a supported version

```bash
go version
go env GOROOT GOPATH GO111MODULE 
```

NOTE: The `GOROOT` and `GOPATH` above must not be the same path.  It is
recommended that `GOPATH` is set to a directory in your home directory such as
`~/goprojects` to avoid write permission issues.  It is also recommended to add
`$GOPATH/bin` to your `PATH` at this point.

* Obtain a copy of `soterdash`, and build it

```bash
git clone ssh://github.com/soteria-dag/soterdash $GOPATH/src/github.com/soteria-dag/soterdash
cd $GOPATH/src/github.com/soteria-dag/soterdash
export GO111MODULE=on
go build && echo "build ok" && go install . && echo "install ok"
```

* `soterdash` will now be installed in `$GOPATH/bin`

## Running `soterdash`

Use CLI parameters to configure `soterdash` and connect it to soterd node(s).

```
$ soterdash -h
  Usage of soterdash:
    -c string
      	Soterd RPC certificate path (default "/home/me/.soterd/rpc.cert")
    -l string
      	Which [ip]:port to listen on (default ":5072")
    -mainnet
          Use mainnet for soterd network census worker connections
    -p string
      	Soterd RPC password
    -r string
      	Soterd RPC ip:port to connect to
    -regnet
          Use regnet (regression test network) for soterd network census worker connections
    -simnet
          Use simnet for soterd network census worker connections
    -testnet
          Use testnet for soterd network census worker connections
    -u string
      	Soterd RPC username
```

### Example run

An example run of `soterdash` could be:

```bash
soterdash -simnet -r 127.0.0.1:18556 -u USER -p PASS
```

Here it's connecting to the `soterd` at `127.0.0.1:18556`, using the username `USER` and password `PASS`. The web ui for `soterdash` is available at `localhost:5072`