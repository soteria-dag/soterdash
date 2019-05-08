// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"os/user"
	"path/filepath"
	"time"

	"github.com/soteria-dag/soterdash/census"
	"github.com/soteria-dag/soterdash/rand"
	"github.com/soteria-dag/soterd/chaincfg"
	"github.com/soteria-dag/soterd/rpcclient"
)

var (
	// Holds the directly-connected RPC clients
	clients []*rpcclient.Client
	// The census enumerator collects node connectivity info from participants in the p2p network
	e *census.Enumerator
)

// pickClient returns a randomly-chosen client
func pickClient(clients []*rpcclient.Client) (*rpcclient.Client, error) {
	if len(clients) == 0 {
		// rand.Int panics if the max value is <= 0, so we will bail out early if there's no clients to choose from.
		return nil, fmt.Errorf("clients slice is empty")
	}

	n, err := rand.RandInt(len(clients))
	if err != nil {
		return nil, err
	}

	return clients[n], nil
}

// soterdCertPath returns the default soterd RPC certificate path
func soterdCertPath() (string, error) {
	me, err := user.Current()
	if err != nil {
		return "", err
	}

	return filepath.Join(me.HomeDir, ".soterd", "rpc.cert"), nil
}

// soterdP2PAddrs returns p2p address info for the node, inbound and outbound peers
func soterdP2PAddrs(c *rpcclient.Client) ([]string, []string, error) {
	var me []string
	var addrs []string

	listenAddrs, err := c.GetListenAddrs()
	if err != nil {
		return me, addrs, err
	}
	me = listenAddrs.P2P

	cache, err := c.GetAddrCache()
	if err != nil {
		return me, addrs, err
	}
	addrs = cache.Addresses

	return me, addrs, nil
}

// TODO(cedric): Support connecting to more than one soterd node via RPC on startup
func main() {
	// Determine what default soterd RPC certificate path should be
	defaultSoterdCertPath, err := soterdCertPath()
	if err != nil {
		log.Fatalf("Couldn't determine what default soterd certificate path should be: %s", err)
	}

	// Parse cli flags
	var mainnet, testnet, regnet, simnet bool
	var addr, censusInterval, soterdAddr, soterdUser, soterdPass, soterdCertPath string
	var censusWorkers int

	flag.StringVar(&addr, "l", ":5072", "Which [ip]:port to listen on")
	flag.StringVar(&soterdAddr, "r", "", "Soterd RPC ip:port to connect to")
	flag.StringVar(&soterdUser, "u", "", "Soterd RPC username")
	flag.StringVar(&soterdPass, "p", "", "Soterd RPC password")
	flag.StringVar(&soterdCertPath, "c", defaultSoterdCertPath, "Soterd RPC certificate path")
	flag.IntVar(&censusWorkers, "w", 2, "Number of p2p network census workers to start")
	flag.StringVar(&censusInterval, "i", "15s", "Time interval for polling nodes")
	flag.BoolVar(&mainnet, "mainnet", false, "Use mainnet for soterd network census worker connections")
	flag.BoolVar(&testnet, "testnet", false, "Use testnet for soterd network census worker connections")
	flag.BoolVar(&regnet, "regnet", false, "Use regnet (regression test network) for soterd network census worker connections")
	flag.BoolVar(&simnet, "simnet", false, "Use simnet for soterd network census worker connections")

	flag.Parse()

	interval, err := time.ParseDuration(censusInterval)
	if err != nil {
		log.Fatalf("Failed to parse census interval '%s': %s", censusInterval, err)
	}

	// Pick soterd census worker net params
	var net chaincfg.Params
	netCount := 0
	if mainnet {
		net = chaincfg.MainNetParams
		netCount++
	}
	if testnet {
		net = chaincfg.TestNet1Params
		netCount++
	}
	if regnet {
		net = chaincfg.RegressionNetParams
		netCount++
	}
	if simnet {
		net = chaincfg.SimNetParams
		netCount++
	}
	if netCount == 0 {
		log.Fatalf("must choose p2p network for soterd census workers (-mainnet, -testnet, -regnet, -simnet)")
	}
	if netCount > 1 {
		log.Fatalf("must choose only one p2p network for soterd census workers (-mainnet, -testnet, -regnet, -simnet)")
	}

	// Read Soterd RPC certificate
	cert, err := ioutil.ReadFile(soterdCertPath)
	if err != nil {
		log.Fatalf("Failed to load certificate %s: %s", soterdCertPath, err)
	}

	// Connect to soterd node
	rpcCfg := rpcclient.ConnConfig{
		Host: soterdAddr,
		Endpoint: "ws",
		User: soterdUser,
		Pass: soterdPass,
		Certificates: cert,
	}
	client, err := rpcclient.New(&rpcCfg, nil)
	if err != nil {
		log.Fatalf("Failed to connect to soterd at %s: %s", soterdAddr, err)
	}
	clients = append(clients, client)

	// Determine listening p2p addresses of seed nodes
	listen, peers, err := soterdP2PAddrs(client)
	if err != nil {
		log.Fatalf("Failed to find soterd node listening interfaces: %s", err)
	}

	// Route requests for / (or anything that doesn't match another pattern) to handleRoot, in DefaultServeMux.
	// https://golang.org/pkg/net/http/#ServeMux
	http.HandleFunc("/", handleRoot)
	// Show block details
	// The trailing / allows us to route requests for URLs like /block/09d41fa to handleBlock
	http.HandleFunc("/block/", handleBlock)
	// Render dag with min, max height, and pagination support
	http.HandleFunc("/dag", handleDag)
	http.HandleFunc("/favicon.ico", handleFavicon)
	// Show census-enumerated node details
	http.HandleFunc("/node/", handleNode)
	// Graph census-enumerated node connectivity
	http.HandleFunc("/nodegraph", handleNodeGraph)
	// Show directly-connected RPC node details
	http.HandleFunc("/rpcnodes", handleRPCNodes)
	// Serve file contents from static folder
	http.HandleFunc("/static/", handleStatic)

	// Start http server in a goroutine, so that it doesn't block our flow for starting census
	// or other background activities. We'll use a channel to let us know if there was a problem encountered.
	httpSrvResult := make(chan error)
	startHttp := func() {
		// Use DefaultServeMux as the handler
		err := http.ListenAndServe(addr, nil)
		httpSrvResult <- err
	}

	go startHttp()

	// Start the soterd p2p network census
	log.Println("Starting soterd p2p network census")
	seedNodes := make([]*census.Node, 0)
	for _, a := range listen {
		cn := census.Node{
			Address: a,
		}
		seedNodes = append(seedNodes, &cn)
	}
	for _, a := range peers {
		cn := census.Node{
			Address: a,
		}
		seedNodes = append(seedNodes, &cn)
	}
	e = census.New(seedNodes, interval, censusWorkers, &net)
	e.Start()

	// Listen for signals telling us to shut down, or for http server to stop
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	select {
		case err := <-httpSrvResult:
			if err != nil {
				log.Printf("Failed to ListenAndServe for addr %s: %s")
			}
		case s := <-c:
			log.Println("Shutting down due to signal:", s)
	}

	// Stop census
	e.Stop()
}

