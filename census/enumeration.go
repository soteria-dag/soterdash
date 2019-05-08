// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package census

import (
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/soteria-dag/soterd/chaincfg"
)

const (
	// The value we use for indicating that a node *is not* being checked by an enumeration worker
	free = int32(0)
	// The value we use for indicating that a node *is* being checked by an enumeration worker
	busy = int32(1)
)

var (
	// How long workers should sleep between taking actions (like checking nodes)
	workerWait = time.Second * 5
)

// Enumerator is used to collect information from soterd nodes participating in the p2p network.
// It makes the census data available to other goroutines (soterdash web ui).
type Enumerator struct {
	// nodes we'll start the first census from
	seeds []*Node

	// All nodes we've collected census for.
	// It is also used to prevent polling the same node multiple times when there are circular connections between nodes.
	nodes map[string]*Node

	// A lock to prevent multiple updates to the nodes map at the same time
	nodesLock sync.RWMutex

	// The interval that we'll poll each node at
	interval time.Duration

	// How many workers we should use for enumeration
	maxWorkers int

	// Which p2p network to use, when connecting to soterd nodes
	soterdNet *chaincfg.Params

	// Help Start and Stop methods to determine if enumeration has already been started/stopped
	started        int32
	shutdown       int32

	// Helps wait for all goroutines to finish before shutdown completes
	wg sync.WaitGroup

	// Listens on notifications from workers, and logs them on behalf of workers
	workerNotifications chan string

	// Listens on quit for a message to shutdown
	quit	chan struct{}
}

// enumeration starts the work of polling nodes in the p2p network.
func (e *Enumerator) enumeration() {
	defer e.wg.Done()

	// Launch worker processes
	log.Println("Starting workers")
	var err error
	var workers []*Worker
	for i := 0; i < e.maxWorkers; i++ {
		var w *Worker
		w, err = NewWorker(e, i, workerWait)
		if err != nil {
			break
		}

		workers = append(workers, w)
		e.wg.Add(1)
		go w.run()
	}

	// Abort if there was an error starting workers
	if err != nil {
		log.Printf("Failed to start worker: %s", err)
		for _, w := range workers {
			close(w.quit)
		}
		return
	}

	// Wait for messages
	for {
		select {
			case m := <-e.workerNotifications:
				log.Println(m)
			case <-e.quit:
				for _, w := range workers {
					close(w.quit)
				}
				return
		}
	}
}

// pickNode returns a node that is available for polling.
// This method is meant to be called from the context of an enumeration worker goroutine
func (e *Enumerator) pickNode() (*Node, bool) {
	e.nodesLock.RLock()
	defer e.nodesLock.RUnlock()

	for _, n := range e.nodes {
		if !n.isStale(e.interval) {
			// Don't choose a node whose last poll results are still fresh
			continue
		}

		if n.reserve() {
			// We were successful in reserving this node for polling
			return n, true
		}
	}

	return nil, false
}

// New returns an Enumerator.
// Use Start() to start taking census from soterd nodes.
func New(seeds []*Node, interval time.Duration, workers int, net *chaincfg.Params) *Enumerator {
	e := Enumerator{
		seeds:      seeds,
		nodes:		make(map[string]*Node),
		interval:   interval,
		maxWorkers: workers,
		soterdNet: net,
		workerNotifications: make(chan string),
		quit:       make(chan struct{}),
	}

	return &e
}

// AddToCensus adds the node to the list of nodes to be polled in enumeration
func (e *Enumerator) AddToCensus(n *Node) {
	e.nodesLock.Lock()
	defer e.nodesLock.Unlock()

	_, exists := e.nodes[n.Address]
	if !exists {
		e.nodes[n.Address] = n
	}
}

// Get returns a *Node whose address matches the string, and a bool of if a match was found
func (e *Enumerator) Get(a string) (*Node, bool) {
	e.nodesLock.RLock()
	defer e.nodesLock.RUnlock()

	n, exists := e.nodes[a]
	return n, exists
}

// IsInCensus returns true if the node is included in the surveys
func (e *Enumerator) IsInCensus(n *Node) bool {
	e.nodesLock.RLock()
	defer e.nodesLock.RUnlock()

	_, exists := e.nodes[n.Address]
	return exists
}

// Nodes returns all of the known nodes in the census
func (e *Enumerator) Nodes() []*Node {
	e.nodesLock.RLock()
	defer e.nodesLock.RUnlock()

	var nodes []*Node
	for _, n := range e.nodes {
		nodes = append(nodes, n)
	}

	return nodes
}

// RemoveFromCensus removes the node to the list of nodes to be polled in enumeration
func (e *Enumerator) RemoveFromCensus(n *Node) {
	e.nodesLock.Lock()
	defer e.nodesLock.Unlock()

	delete(e.nodes, n.Address)
}

// Start enumeration in a new goroutine
func (e *Enumerator) Start() {
	// Already started?
	if atomic.AddInt32(&e.started, 1) != 1 {
		return
	}

	e.wg.Add(1)
	go e.enumeration()
}

// Stop enumeration
func (e *Enumerator) Stop() {
	if atomic.AddInt32(&e.shutdown, 1) != 1 {
		// Already in the process of stopping
		return
	}

	close(e.quit)
	e.wg.Wait()
}
