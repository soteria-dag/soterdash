// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"

	"github.com/wcharczuk/go-chart"

	"github.com/soteria-dag/soterd/rpcclient"
	"github.com/soteria-dag/soterd/wire"
)

const (
	// How many characters of hash we'll use for 'small' hash
	smallHashLen = 7
)

var (
	// Read templates from templates dir
	templates = template.Must(template.ParseGlob("templates/*.tmpl"))

	// The format used for graphviz color codes
	hexColor = "#%x%x%x"

	// Color bytes are R, G, B values
	green = color(0, 217, 101)
	orange = color(255, 191, 0)
	gray = color(185, 195, 198)
)

type dagRange struct {
	Min int32
	Max int32
}

// color returns a string for the r, g, b values in graphviz format:
// #rrggbb, where rr is 2 hex characters for red, gg is 2 hex characters for green, bb is 2 hex characters for blue.
func color(r, g, b int) string {
	return fmt.Sprintf(hexColor, []byte{uint8(r)}, []byte{uint8(g)}, []byte{uint8(b)})
}

// colorPicker picks a color based on the input value and returns a string in the graphviz format:
// #rrggbb, where rr is 2 hex characters for red, gg is 2 hex characters for green, bb is 2 hex characters for blue.
func colorPicker (v int) string {
	color := chart.GetAlternateColor(v)
	// Slice of bytes is used here instead of int value of color, so that Sprintf
	// uses 2 characters per byte instead of 1, which is what the graphviz format wants.
	return fmt.Sprintf(hexColor, []byte{color.R}, []byte{color.G}, []byte{color.B})
}

// setContentType sets the Content-Type HTTP header of a response
func setContentType(w http.ResponseWriter, cType string) {
	w.Header().Set("Content-Type", cType)
}

// We'll provide renderDagsDot with a function for picking style of the blocks.
//
// The stylePicker should return a string in the graphviz format
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#k:style
func stylePicker (solid, fill bool) string {
	if solid && fill {
		return "filled"
	} else if solid && !fill {
		return "solid"
	} else if !solid && fill {
		return "filled, dashed"
	} else {
		return "solid, dashed"
	}

}

// renderHTML renders the given template text into the response, with optional template data
func renderHTML(w http.ResponseWriter, tmpl string, data interface{}) {
	t := template.New("htmlElem")
	t, err := t.Parse(tmpl)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	err = t.Execute(w, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// renderHTMLErr renders the error in the response
func renderHTMLErr(w http.ResponseWriter, err error) {
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// renderHTMLTmpl renders the template from the file in the response
func renderHTMLTmpl(w http.ResponseWriter, name string, data interface{}) {
	err := templates.ExecuteTemplate(w, name, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// renderHTMLOpen renders the <html> element's opening tag in the response
func renderHTMLOpen(w http.ResponseWriter) {
	setContentType(w, "text/html")
	renderHTML(w, "<!DOCTYPE html>", nil)
	renderHTML(w, "<html>", nil)
}

// renderHTMLClose renders the </html> element's closing tag in the response
func renderHTMLClose(w http.ResponseWriter) {
	renderHTML(w, "</html>", nil)
}

// renderHTMLBodyOpen renders the <body> element's opening tag in the response
func renderHTMLBodyOpen(w http.ResponseWriter) {
	renderHTML(w, "<body>", nil)
}

// renderHTMLBodyClose renders the <body> element's closing tag in the response
func renderHTMLBodyClose(w http.ResponseWriter) {
	renderHTML(w, "</body>", nil)
}

// renderHTMLDagForm renders the dag viewing form in the response
func renderHTMLDagForm(w http.ResponseWriter, min, max int32) {
	f := dagRange{
		Min: min,
		Max: max,
	}

	renderHTMLTmpl(w, "dag_form.tmpl", f)
}

// renderHTMLDagPag renders dag pagination in the response
func renderHTMLDagPag(w http.ResponseWriter, min, max int32, amt int32) {
	start := `
<nav aria-label="dag pagination">
	<ul class="pagination">
`
	end := `
	</ul>
</nav>
`
	renderHTML(w, start, nil)

	if min > 0 {
		// Render previous link
		prevMin := min - amt
		if prevMin < 0 {
			prevMin = 0
		}

		prevMax := max - amt
		if prevMax < 0 {
			prevMax = max
		}

		r := dagRange{
			Min: prevMin,
			Max: prevMax,
		}

		tmpl := `<li class="page-item"><a class="page-link" href="/dag?min={{ .Min }}&max={{ .Max }}">Previous</a></li>`
		renderHTML(w, tmpl, r)
	}

	// Render next link
	nextMin := max
	nextMax := max + amt

	r := dagRange{
		Min: nextMin,
		Max: nextMax,
	}

	tmpl := `<li class="page-item"><a class="page-link" href="/dag?min={{ .Min }}&max={{ .Max }}">Next</a></li>`
	renderHTML(w, tmpl, r)

	renderHTML(w, end, nil)
}

// renderHTMLHeader renders the header.tmpl template in the response
func renderHTMLHeader(w http.ResponseWriter, title string) {
	renderHTMLTmpl(w, "header.tmpl", title)
}

// renderHTMLNavbar renders the navbar.tmpl template in the response
func renderHTMLNavbar(w http.ResponseWriter) {
	type navbar struct {
		// The 'brand' name used in the navbar
		Brand string
	}

	n := navbar{
		Brand: "soterdash",
	}

	renderHTMLTmpl(w, "navbar.tmpl", n)
}

// renderHTMLFooter renders the footer.tmpl template in the response
func renderHTMLFooter(w http.ResponseWriter) {
	renderHTMLTmpl(w, "footer.tmpl", nil)
}

// renderHTMLScript renders the script.tmpl template in the response
func renderHTMLScript(w http.ResponseWriter) {
	renderHTMLTmpl(w, "script.tmpl", nil)
}

// RenderHTML renders the soterdBlock as a bootstrap card in the response
func (b *soterdBlock) RenderHTML(w http.ResponseWriter) {
	renderHTMLTmpl(w, "soterd_block.tmpl", b)
}

// RenderHTML renders the soterdNode as a bootstrap card in the response
func (n *soterdNode) RenderHTML(w http.ResponseWriter) {
	renderHTMLTmpl(w, "soterd_node.tmpl", n)
}

// RenderHTML renders the soterdRPCNode as a bootstrap card in the response
func (rpc *soterdRPCNode) RenderHTML(w http.ResponseWriter) {
	renderHTMLTmpl(w, "soterd_rpc_node.tmpl", rpc)
}

// ShortHash returns the short hash of the blocks
func (b *soterdBlock) ShortHash() string {
    hash := b.Header.BlockHash().String()
    smallHashIndex := len(hash) - smallHashLen
    return hash[smallHashIndex:]
}

// RenderDagsDot returns a representation of the dag in graphviz DOT file format.
//
// RenderDagsDot makes use of the "dot" command, which is a part of the "graphviz" suite of software.
// http://graphviz.org/
func RenderDagsDot(nodes []*rpcclient.Client, minHeight int32, maxHeight int32) ([]byte, error) {
	var dot bytes.Buffer

	// Map blocks to the nodes that created them. This will be used to color blocks in dag
	blockCreator := make(map[string]int)
	for i, n := range nodes {
		resp, err := n.GetBlockMetrics()
		if err != nil {
			continue
		}

		for _, hash := range resp.BlkHashes {
			blockCreator[hash] = i
		}
	}

	// We'll use the first node for the dag, and metrics from all nodes for block coloring
	node := nodes[0]
	tips, err := node.GetDAGTips()
	if err != nil {
		return dot.Bytes(), err
	}

	// Determine the range of dag height we'll render
	if minHeight < 0 {
		minHeight = 0
	}
	if maxHeight > tips.MaxHeight {
		maxHeight = tips.MaxHeight
	}

	dag := make([][]*wire.MsgBlock, 0)
	blockHeight := make(map[string]int32)

	// Index all the blocks
	for height := int32(minHeight); height <= maxHeight; height++ {
		blocks := make([]*wire.MsgBlock, 0)

		hashes, err := node.GetBlockHash(int64(height))
		if err != nil {
			return dot.Bytes(), err
		}

		for _, hash := range hashes {
			blockHeight[hash.String()] = height

			block, err := node.GetBlock(hash)
			if err != nil {
				return dot.Bytes(), err
			}

			blocks = append(blocks, block)
		}

		dag = append(dag, blocks)
	}

	// Build a map of block coloring results
	dagcoloring, err := node.GetDAGColoring()
	if err != nil {
		return dot.Bytes(), err
	}
	blockcoloring := make(map[string]bool)
	for _, dagNode := range dagcoloring {
		hash := dagNode.Hash
		coloring := dagNode.IsBlue
		blockcoloring[hash] = coloring
	}

	// Express dag in DOT file format

	// graphIndex tracks block hash -> graph node number, which is used to connect parent-child blocks together.
	graphIndex := make(map[string]int)
	// n keeps track of the 'node' number in graph file language
	n := 0

	// Specify that this graph is directed, and set the ID to 'dag'
	_, err = fmt.Fprintln(&dot, "digraph dag {")
	if err != nil {
		return dot.Bytes(), err
	}

	// Create a node in the graph for each block
	for _, blocks := range dag {
		for _, block := range blocks {
			hash := block.BlockHash().String()
			smallHashIndex := len(hash) - smallHashLen
			height := blockHeight[hash]
			graphIndex[hash] = n
			url := fmt.Sprintf("/block/%s", hash)

			// determine the coloring of the block and fetch the style string: default, "filled" or "filled,dashed"
			dagcoloring := blockcoloring[hash]
			creator, exists := blockCreator[hash]
			style := stylePicker(dagcoloring, exists)
			var err error
			if exists {
				// Color this block based on which miner created it
				color := colorPicker(creator)
				_, err = fmt.Fprintf(&dot, "n%d [label=\"%s\", tooltip=\"node %d height %d hash %s\", href=\"%s\", fillcolor=\"%s\", style=\"%s\"];\n",
					n, hash[smallHashIndex:], creator, height, hash, url, color, style)
			} else {
				// No color for this block
				_, err = fmt.Fprintf(&dot, "n%d [label=\"%s\", tooltip=\"height %d hash %s\", href=\"%s\", style=\"%s\"];\n",
					n, hash[smallHashIndex:], height, hash, url, style)
			}
			if err != nil {
				return dot.Bytes(), err
			}

			n++
		}
	}

	// Connect the nodes in the graph together
	for _, blocks := range dag {
		for _, block := range blocks {
			blockN := graphIndex[block.BlockHash().String()]

			for _, parent := range block.Parents.Parents {
				parentN, exists := graphIndex[parent.Hash.String()]
				if !exists {
					continue
				}

				_, err := fmt.Fprintf(&dot, "n%d -> n%d;\n", blockN, parentN)
				if err != nil {
					return dot.Bytes(), err
				}
			}
		}
	}

	// Close the graph statement list
	dot.WriteString("}")

	return dot.Bytes(), nil
}

// RenderNodeGraphDot returns a representation of the node connectivity in graphviz DOT file format.
//
// RenderNodeGraphDot makes use of the "dot" command, which is a part of the "graphviz" suite of software.
// http://graphviz.org/
func RenderNodeGraphDot() ([]byte, error) {
	var dot bytes.Buffer

	nodes := e.Nodes()
	// graphIndex tracks node address -> graph node number, which is used to connect nodes together.
	graphIndex := make(map[string]int)
	// n keeps track of the 'node' number in graph file language
	n := 0

	// Specify that this graph is directed, and set the ID to 'soterdNodes'
	_, err := fmt.Fprintln(&dot, "graph soterdNodes {")
	if err != nil {
		return dot.Bytes(), err
	}

	// Create a node in the graph for each soterd node
	for _, sn := range nodes {
		graphIndex[sn.Address] = n

		var color string
		if sn.IsStale(e.Interval * 3) {
			// If we don't have new stats from the node within 3 polling intervals,
			// we can indicate that the node's connectivity info is stale by coloring it gray.
			color = gray
		} else if sn.Online {
			color = green
		} else {
			color = orange
		}

		url := fmt.Sprintf("/node/%s", sn.Address)

		_, err = fmt.Fprintf(&dot, "n%d [label=\"%s\", tooltip=\"version %s online %v\", href=\"%s\", fillcolor=\"%s\", style=filled];\n",
			n, sn, sn.Version, sn.Online, url, color)
		if err != nil {
			return dot.Bytes(), err
		}
		n++
	}

	// Connect nodes in graph together
	for _, soterdNode := range nodes {
		n := graphIndex[soterdNode.Address]

		for _, otherSoterdNode := range soterdNode.Connections() {
			if soterdNode.Address == otherSoterdNode.Address {
				continue
			}

			cn, exists := graphIndex[otherSoterdNode.Address]
			if !exists {
				continue
			}

			_, err := fmt.Fprintf(&dot, "n%d -- n%d;\n", n, cn)
			if err != nil {
				return dot.Bytes(), err
			}
		}
	}

	// Close the graph statement list
	dot.WriteString("}")

	return dot.Bytes(), nil
}
