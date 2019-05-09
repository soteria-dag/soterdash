// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package census

import (
	"sync"
	"sync/atomic"
	"time"
)

// Represent what we know about a node
type Node struct {
	// ip:port of the node
	Address string

	// Soterd version running on node
	Version string

	// How many connections away from our enumerator we found the node.
	// (directly-connected nodes like our seeds are zero hops away)
	Hops int

	// If the node was responding to requests from us the last time we checked it
	Online bool

	// Who the node was last known to be connected to.
	// This is used for extending our census's area, and in graphing the connectivity of the p2p network.
	connections []*Node

	// When the node was last polled. This is used to help determine when we should next poll the same node.
	LastChecked time.Time

	// A lock to prevent concurrent updates to various node fields (not the busy field)
	updateLock sync.RWMutex

	// This value is atomically updated, based on if this node is currently being checked by an enumeration worker
	busy int32
}

// allFresh returns true if none of the nodes' LastChecked times were older than the given duration
func allFresh(nodes []*Node, d time.Duration) bool {
	for _, n := range nodes {
		if n.IsStale(d) {
			return false
		}
	}

	return true
}

// isBusy returns true if the node is currently being checked by an enumeration worker
func (n *Node) isBusy() bool {
	v := atomic.LoadInt32(&n.busy)
	if v == free {
		return false
	}
	return true
}

// IsStale returns true if the node's LastChecked time is older from now than the given duration
func (n *Node) IsStale(d time.Duration) bool {
	n.updateLock.RLock()
	defer n.updateLock.RUnlock()
	return n.LastChecked.Add(d).Before(time.Now())
}

// reserve attempts to mark the node as being checked by an enumeration worker. It returns true if the node was able to
// be checked-out by this worker, false if not (in which case the worker could attempt to check another node).
func (n *Node) reserve() bool {
	return atomic.CompareAndSwapInt32(&n.busy, free, busy)
}

// free marks the node as not being checked by an enumeration worker
func (n *Node) free() {
	atomic.StoreInt32(&n.busy, free)
}

func (n *Node) Connections() []*Node {
	n.updateLock.RLock()
	defer n.updateLock.RUnlock()

	var conns []*Node
	for _, n := range n.connections {
		conns = append(conns, n)
	}

	return conns
}

// String returns a string representing the node
func (n *Node) String() string {
	return n.Address
}
