// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package census

import (
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"github.com/soteria-dag/soterdash/driver"
	"github.com/soteria-dag/soterd/rpcclient"
)

// Represent a an Enumeration worker
type Worker struct {
	// A pointer to the Enumerator that controls this worker.
	// We use it to pick nodes to check, and to determine when it's ok to re-check a node.
	e *Enumerator

	// An identifier for the worker
	num int

	// The soterd instance that this worker controls. We use it to communicate with the nodes we're polling.
	soterd *driver.Soterd

	// How long worker will wait between actions like attempting to check nodes
	wait time.Duration

	// Listens on quit for a message to shutdown
	quit chan struct{}

	// This is used to determine if the worker is running
	status int32
}

// checkNode attempts to check a node and update our information for it
func (w *Worker) checkNode(n *Node) error {
	defer n.free()

	c := w.soterd.Client()

	info, err := c.GetInfo()
	n.updateLock.Lock()
	if err != nil {
		n.Online = false
		n.updateLock.Unlock()
		return err
	}
	n.Version = fmt.Sprintf("%d", info.Version)
	n.updateLock.Unlock()

	connected, err := w.soterd.IsConnectedTo(n.Address)
	if err != nil {
		return err
	}

	if !connected {
		err = c.AddNode(n.Address, rpcclient.ANAdd)
		if err != nil {
			return err
		}
	}

	_, peers, err := w.soterd.Addrs()
	if err != nil {
		n.updateLock.Lock()
		n.Online = false
		n.updateLock.Unlock()
		return err
	}

	conns := make([]*Node, 0)
	for _, p := range peers {
		pn := Node{Address: p}
		conns = append(conns, &pn)

		// Add the node's peers to the survey for future polls, if they haven't been added already
		w.e.AddToCensus(&pn)
	}

	n.updateLock.Lock()
	n.connections = conns
	n.Online = true
	n.LastChecked = time.Now()
	n.updateLock.Unlock()

	// Add the node to the survey for future polls, if it hasn't already
	w.e.AddToCensus(n)

	return nil
}

// checkSeeds attempts to check all the seeds
func (w *Worker) checkSeeds() {
	for !allFresh(w.e.seeds, w.e.Interval) {
		for _, n := range w.e.seeds {
			if n.reserve() {
				log.Printf("worker %s\tchecking seed %s", w, n)
				err := w.checkNode(n)
				if err != nil {
					log.Printf("worker %s\terror checking %s: %s", w, n, err)
				}
			}
		}
		time.Sleep(w.wait)
	}
}

// run runs the worker
func (w *Worker) run() {
	atomic.AddInt32(&w.status, busy)
	ticker := time.NewTicker(w.wait)

	// Start soterd process
	err := w.soterd.Start()

	defer func() {_ = w.soterd.Stop()}()
	defer atomic.StoreInt32(&w.status, free)
	defer ticker.Stop()
	defer w.e.wg.Done()

	if err != nil {
		errMsg := fmt.Errorf("worker %s failed to start soterd process: %s", w, err)
		w.e.workerNotifications <- errMsg.Error()
		return
	}

	// Initialize the seed nodes
	w.checkSeeds()

	// Loop polling of nodes until worker asked to quit
	for {
		select {
			case <-ticker.C:
				n, ours := w.e.pickNode()
				if ours {
					err := w.checkNode(n)
					log.Printf("worker %s\tchecked %s\tver %s", w, n, n.Version)
					if err != nil {
						log.Printf("worker %s\terror checking %s: %s", w, n, err)
					}
				}
			case <-w.quit:
				return
		}
	}
}

// isRunning returns true if the worker is currently running
func (w *Worker) isRunning() bool {
	v := atomic.LoadInt32(&w.status)
	if v == busy {
		return true
	}
	return false
}

func (w *Worker) String() string {
	return fmt.Sprintf("%d", w.num)
}

// NewWorker returns a new instance of Worker type
func NewWorker(e *Enumerator, num int, wait time.Duration) (*Worker, error) {
	s, err := driver.NewSoterd(e.soterdNet)
	if err != nil {
		return nil, err
	}

	w := Worker{
		e: e,
		num: num,
		soterd: s,
		wait: wait,
		quit: make(chan struct{}),
	}

	return &w, nil
}