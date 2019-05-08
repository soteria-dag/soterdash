// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"html/template"
	"time"

	"github.com/soteria-dag/soterdash/census"
	"github.com/soteria-dag/soterd/chaincfg/chainhash"
	"github.com/soteria-dag/soterd/rpcclient"
	"github.com/soteria-dag/soterd/soterjson"
	"github.com/soteria-dag/soterd/soterutil"
	"github.com/soteria-dag/soterd/wire"
)

const (
	// How many generations from tips we'll render for RecentDagSvg
	recentDagRange = int32(3)
)

// Represent block data that we're interested in rendering
type soterdBlock struct {
	Header       wire.BlockHeader
	Parents      wire.ParentSubHeader
	Transactions []*wire.MsgTx
	Confirmations int64
	Height       int32
	MerkleRoot   string
	NextHashes   []string
	Difficulty   float64
}

// Represents census-enumerated node data that we're interested in rendering
type soterdNode struct {
	Address string
	Version string
	Online bool
	Connections []*census.Node
	LastChecked time.Time
}

// Represent node data that we're interested in rendering
type soterdRPCNode struct {
	Id int
	// The dag net of the node (testnet, etc)
	Net string
	Version string
	InboundPeers map[int32]*soterjson.GetPeerInfoResult
	OutboundPeers map[int32]*soterjson.GetPeerInfoResult
	InboundPeerCount int
	OutboundPeerCount int
	// Hashes of dag tips of node
	Tips []string
	// Hash of tips of node
	VirtualHash string
	MinHeight int32
	MaxHeight int32
	BlkCount uint32
	// SVG rendering of recent dag (MaxHeight - recentDagRange generations) to MaxHeight
	RecentDagSvg template.HTML
}

// sortPeers returns the number of **unique** peer connections
func sortPeers(peers []soterjson.GetPeerInfoResult) (map[int32]*soterjson.GetPeerInfoResult, map[int32]*soterjson.GetPeerInfoResult) {
	inbound := make(map[int32]*soterjson.GetPeerInfoResult)
	outbound := make(map[int32]*soterjson.GetPeerInfoResult)

	for _, p := range peers {
		var targetMap *map[int32]*soterjson.GetPeerInfoResult
		if p.Inbound {
			targetMap = &inbound
		} else {
			targetMap = &outbound
		}

		_, exists := (*targetMap)[p.ID]
		if exists {
			continue
		}
		(*targetMap)[p.ID] = &p
	}

	return inbound, outbound
}

// blockInfo returns a soterdBlock, which can be rendered
func blockInfo(c *rpcclient.Client, hash string) (soterdBlock, error) {
	h, err := chainhash.NewHashFromStr(hash)
	if err != nil {
		return soterdBlock{}, err
	}

	block, err := c.GetBlock(h)
	if err != nil {
		return soterdBlock{}, err
	}

	header, err := c.GetBlockHeaderVerbose(h)
	if err != nil {
		return soterdBlock{}, err
	}

	// NOTE(cedric): soterd node(s) we're connecting to need to be running a wallet service, in order for us to use
	// gettransaction to pull more transaction info.
	sb := soterdBlock{
		Header: block.Header,
		Parents: block.Parents,
		Transactions: block.Transactions,
		Confirmations: header.Confirmations,
		Height: header.Height,
		MerkleRoot: header.MerkleRoot,
		NextHashes: header.NextHashes,
		Difficulty: header.Difficulty,
	}

	return sb, nil
}

// rpcNodeInfo returns a soterdRPCNode struct, which can be rendered
func rpcNodeInfo(c *rpcclient.Client) (soterdRPCNode, error) {
	// Node network
	net, err := c.GetCurrentNet()
	if err != nil {
		return soterdRPCNode{}, err
	}

	// Version
	result, err := c.Version()
	if err != nil {
		return soterdRPCNode{}, err
	}
	verInfo, verExists := result["soterdjsonrpcapi"]

	// Peers
	peers, err := c.GetPeerInfo()
	if err != nil {
		return soterdRPCNode{}, err
	}
	inbound, outbound := sortPeers(peers)

	// Dag tip
	tips, err := c.GetDAGTips()
	if err != nil {
		return soterdRPCNode{}, err
	}

	// Determine dag rendering range
	minHeight := (tips.MaxHeight - recentDagRange)
	if minHeight < 0 {
		minHeight = 0
	}

	// Dag svg rendering
	nodes := []*rpcclient.Client{c}
	dot, err := RenderDagsDot(nodes, minHeight, tips.MaxHeight)
	if err != nil {
		return soterdRPCNode{}, err
	}
	svg, err := soterutil.DotToSvg(dot)
	if err != nil {
		return soterdRPCNode{}, err
	}
	svgEmbed, err := soterutil.StripSvgXmlDecl(svg)
	if err != nil {
		return soterdRPCNode{}, err
	}

	// NOTE(cedric): Could call getnettotals to add to node Network info
	// Assemble the data into a soterdRPCNode model
	n := soterdRPCNode{
		Net: net.String(),
		InboundPeers: inbound,
		OutboundPeers: outbound,
		InboundPeerCount: len(inbound),
		OutboundPeerCount: len(outbound),
		Tips: tips.Tips,
		VirtualHash: tips.Hash,
		MinHeight: tips.MinHeight,
		MaxHeight: tips.MaxHeight,
		BlkCount: tips.BlkCount,
		RecentDagSvg: template.HTML(svgEmbed),
	}

	if verExists {
		n.Version = verInfo.VersionString
	}

	return n, nil
}

// nodeInfo returns a soterdNode, which can be rendered
func nodeInfo(address string) (soterdNode, error) {
	cNode, exists := e.Get(address)
	if !exists {
		return soterdNode{}, fmt.Errorf("node with address %s not found in census", address)
	}

	n := soterdNode{
		Address: cNode.Address,
		Version: cNode.Version,
		Online: cNode.Online,
		Connections: cNode.Connections(),
		LastChecked: cNode.LastChecked,
	}

	return n, nil
}