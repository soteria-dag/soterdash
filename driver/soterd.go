// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package driver

import (
	"net"
	"strings"

	"github.com/soteria-dag/soterd/chaincfg"
	"github.com/soteria-dag/soterd/integration/rpctest"
	"github.com/soteria-dag/soterd/rpcclient"
)

// Soterd provides a bit of an abstraction from the soterd rpctest.Harness interface.
// (at some point we may want to pull soterd-driver-functionality out of rpctest package)
type Soterd struct {
	process *rpctest.Harness
}

// NewSoterd returns a new soterd process
func NewSoterd(net *chaincfg.Params) (*Soterd, error) {
	//port, err := rand.RandLoopPort()
	//if err != nil {
	//	return nil, err
	//}

	extra := []string{
		//"--debuglevel=debug",
		//"--profile=" + port,
	}

	h, err := rpctest.New(net, nil, extra, true)
	if err != nil {
		return nil, err
	}

	p := Soterd{
		process: h,
	}

	return &p, nil
}

// Start starts the soterd process
func (s *Soterd) Start() error {
	return s.process.SetUp(false, 0)
}

// Stop stops the soterd process
func (s *Soterd) Stop() error {
	return s.process.TearDown()
}

// Client returns a pointer to an rpc client connection to the soterd process
func (s *Soterd) Client() *rpcclient.Client {
	return s.process.Node
}

// Addrs returns p2p address info for the node, and peer addresses its aware of
func (s *Soterd) Addrs() ([]string, []string, error) {
	var me []string
	var addrs []string

	listenAddrs, err := s.process.Node.GetListenAddrs()
	if err != nil {
		return me, addrs, err
	}
	me = listenAddrs.P2P

	cache, err := s.process.Node.GetAddrCache()
	if err != nil {
		return me, addrs, err
	}
	addrs = cache.Outbound

	return me, addrs, nil
}

// IsConnectedTo returns true if the node is connected to the address
func (s *Soterd) IsConnectedTo(to string) (bool, error) {
	peers, err := s.process.Node.GetPeerInfo()
	if err != nil {
		return false, err
	}

	// Try looking for an exact match first. This will likely only match against outbound connections.
	for _, p := range peers {
		if p.Addr == to {
			return true, nil
		}
	}

	// Attempt to find matching inbound connections. We only match against host, because we can expect that
	// inbound connections will be using a dynamically-chosen port as the source port of the connection.
	//
	// The problem with this approach is that we don't differentiate between multiple soterd nodes running from behind
	// the same IP. If soterd generated a UUIDv4 on startup and passed it with Version data or another message, we
	// could check if we were connected to a peer based on ID instead of IP.
	for _, p := range peers {
		if !p.Inbound {
			// We already matched against outbound connections
			continue
		}

		var pHost, toHost string
		if strings.Contains(p.Addr, ":") {
			pHost, _, err = net.SplitHostPort(p.Addr)
			if err != nil {
				return false, err
			}
		} else {
			pHost = p.Addr
		}

		if strings.Contains(to, ":") {
			toHost, _, err = net.SplitHostPort(to)
			if err != nil {
				return false, err
			}
		} else {
			toHost = to
		}

		if pHost == toHost {
			return true, nil
		}
	}

	return false, nil
}