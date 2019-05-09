// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/soteria-dag/soterd/soterutil"
)

// beforeBody renders common HTML document sections including the opening <body> element
func beforeBody(w http.ResponseWriter, title string) {
	renderHTMLOpen(w)
	renderHTMLHeader(w, title)
	renderHTMLBodyOpen(w)
	renderHTMLNavbar(w)
}

// afterBody renders common HTML document sections starting from the closing </body> element
// to the end of the HTML document.
func afterBody(w http.ResponseWriter) {
	renderHTMLFooter(w)
	renderHTMLScript(w)
	renderHTMLBodyClose(w)
	renderHTMLClose(w)
}

// handleRoot responds to requests for root url /
func handleRoot(w http.ResponseWriter, r *http.Request) {
	// By default we'll print out node information
	handleRPCNodes(w, r)
}

// handleFavicon responds to requests for /favicon.ico
func handleFavicon(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "static/favicon.ico")
}

// handleStatic responds to requests for files in static/
func handleStatic(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, r.URL.Path[1:])
}

// handleBlock responds to requests for /block/<block hash>
// It renders block details.
func handleBlock(w http.ResponseWriter, r *http.Request) {
	title := "soterdash - block"
	// For r.URL.Path of /block/09d41fa, parts will be: ["", "block", "09d41fa"]
	parts := strings.Split(r.URL.Path, "/")

	// TODO(cedric): If there's no block specified, let the user search for one using a hash (render a search bar)
	if len(parts) != 3 {
		renderHTMLErr(w, fmt.Errorf("couldn't find block hash in request url: %s", r.URL.Path))
		return
	}
	block := parts[2]

	client, err := pickClient(clients)
	if err != nil {
		renderHTMLErr(w, fmt.Errorf("couldn't pick a soterd node to use: %s", err))
		return
	}

	// Render the different HTML sections for the response
	beforeBody(w, title)

	// Render block info
	info, err := blockInfo(client, block)
	if err != nil {
		renderHTMLErr(w, err)
	}
	info.RenderHTML(w)

	// Render HTML sections after the body
	afterBody(w)
}

// handleDag responds to requests for /dag, which renders the dag with the given parameters
func handleDag(w http.ResponseWriter, r *http.Request) {
	title := "soterdash - dag"
	// How many blocks we'll paginate per 'page'
	pagAmt := int32(10)

	client, err := pickClient(clients)
	if err != nil {
		renderHTMLErr(w, fmt.Errorf("couldn't pick a soterd node to use: %s", err))
		return
	}

	// Parse query parameters from request URL
	values := r.URL.Query()
	min := values.Get("min")
	max := values.Get("max")
	var minHeight, maxHeight int32

	tips, err := client.GetDAGTips()
	if err != nil {
		renderHTMLErr(w, err)
	}

	if len(min) == 0 {
		minHeight = tips.MaxHeight - recentDagRange
	} else {
		i, err := strconv.ParseInt(min, 10, 32)
		if err != nil {
			renderHTMLErr(w, err)
		}
		minHeight = int32(i)
	}

	if minHeight < 0 {
		minHeight = 0
	}

	if len(max) == 0 {
		maxHeight = tips.MaxHeight
	} else {
		i, err := strconv.ParseInt(max, 10, 32)
		if err != nil {
			renderHTMLErr(w, err)
		}
		maxHeight = int32(i)
	}

	// Render the different HTML sections for the response
	beforeBody(w, title)
	renderHTML(w, "<br>", nil)

	// Render form for updating dag view
	formMaxHeight := maxHeight
	if formMaxHeight > tips.MaxHeight {
		formMaxHeight = tips.MaxHeight
	}
	renderHTMLDagForm(w, minHeight, formMaxHeight)
	renderHTML(w, "<br>", nil)

	// Dag svg rendering
	dot, err := RenderDagsDot(clients, minHeight, maxHeight)
	if err != nil {
		renderHTMLErr(w, err)
	}
	svg, err := soterutil.DotToSvg(dot)
	if err != nil {
		renderHTMLErr(w, err)
	}
	svgEmbed, err := soterutil.StripSvgXmlDecl(svg)
	if err != nil {
		renderHTMLErr(w, err)
	}
	renderHTML(w, "<figure>{{ . }}</figure>", template.HTML(svgEmbed))

	// Render dag pagination links
	renderHTMLDagPag(w, minHeight, formMaxHeight, pagAmt)

	// Render HTML sections after the body
	afterBody(w)
}

// handleNode responds to requests for /node
// It renders census-enumerated node details
func handleNode(w http.ResponseWriter, r *http.Request) {
	title := "soterdash - node"

	// For r.URL.Path of /node/127.0.0.1:18555, parts will be: ["", "node", "127.0.0.1:18555"]
	parts := strings.Split(r.URL.Path, "/")

	// TODO(cedric): If there's no node specified, let the user search for one using an address (render a search bar)?
	if len(parts) != 3 {
		renderHTMLErr(w, fmt.Errorf("couldn't find node address in request url: %s", r.URL.Path))
		return
	}

	address := parts[2]

	// Render the different HTML sections for the response
	beforeBody(w, title)
	renderHTML(w, "<br>", nil)

	nodeInfo, err := nodeInfo(address)
	if err != nil {
		renderHTMLErr(w, err)
		return
	}
	nodeInfo.RenderHTML(w)

	// Render HTML sections after the body
	afterBody(w)
}

// handleNodeGraph responds to requests for /nodegraph
// It renders a census-enumerated node graph
func handleNodeGraph(w http.ResponseWriter, r *http.Request) {
	title := "soterdash - node graph"

	// Render the different HTML sections for the response
	beforeBody(w, title)
	renderHTML(w, "<br>", nil)

	// Render node graph
	dot, err := RenderNodeGraphDot()
	if err != nil {
		renderHTMLErr(w, err)
	}
	svg, err := soterutil.DotToSvg(dot)
	if err != nil {
		renderHTMLErr(w, err)
	}
	svgEmbed, err := soterutil.StripSvgXmlDecl(svg)
	if err != nil {
		renderHTMLErr(w, err)
	}
	renderHTML(w, "<figure>{{ . }}</figure>", template.HTML(svgEmbed))

	// Render HTML sections after the body
	afterBody(w)
}

// handleRPCNodes responds to requests for /rpcnodes
// It renders directly-connected RPC node details.
func handleRPCNodes(w http.ResponseWriter, r *http.Request) {
	title := "soterdash - rpcnodes"

	// Render the different HTML sections for the response
	beforeBody(w, title)
	renderHTML(w, "<br>", nil)

	for id, client := range clients {
		// Render node info
		info, err := rpcNodeInfo(client)
		if err != nil {
			renderHTMLErr(w, err)
		}

		info.Id = id
		info.RenderHTML(w)
	}

	// Render HTML sections after the body
	afterBody(w)
}