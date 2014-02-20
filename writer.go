package main

import (
	"encoding/xml"
	xmlx "github.com/anaminus/go-pkg-xmlx"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Creates a new node with optional name and value
func newnode(id byte, name string, value string) *xmlx.Node {
	node := xmlx.NewNode(id)
	if name != "" {
		node.Name = xml.Name{
			Local: name,
		}
	}
	if value != "" {
		node.Value = value
	}
	return node
}

// Generate leading whitespace for indentation.
func leading(length int) *xmlx.Node {
	return newnode(xmlx.NT_TEXT, "", "\n"+strings.Repeat("\t", length))
}

func fixIndentation(nodes []*xmlx.Node, indent int) {
	for _, node := range nodes {
		// Remove any child text nodes
		i := 0
		for i < len(node.Children) {
			child := node.Children[i]
			if child.Type == xmlx.NT_TEXT { // verify just whitespace?
				node.RemoveChildAt(i)
			} else {
				i++
			}
		}

		// Determine indentation level
		ind := -2
		nd := node
		for nd != nil {
			nd = nd.Parent
			ind++
		}

		// Insert child indentation before each child
		n := len(node.Children) * 2
		for i := 0; i < n; i += 2 {
			node.AddChildAt(leading(indent+ind), i)
		}
		// Add node indentation to indent closing tag
		node.AddChild(leading(indent + ind - 1))
	}
}

// An xml representation of a ROBLOX class.
type Item struct {
	Class      string
	Properties []Property
}

type Property struct {
	Type  string
	Name  string
	Value string
}

func (item *Item) Node(indent int) *xmlx.Node {
	node := newnode(xmlx.NT_ELEMENT, "Item", "")
	node.AddChild(leading(indent))

	node.SetAttr("class", item.Class)

	props := newnode(xmlx.NT_ELEMENT, "Properties", "")
	props.AddChild(leading(indent + 1))

	node.AddChildAt(leading(indent+1), -2)
	node.AddChildAt(props, -2)

	for _, prop := range item.Properties {
		pele := newnode(xmlx.NT_ELEMENT, prop.Type, "")
		pele.SetAttr("name", prop.Name)

		var pval *xmlx.Node
		if prop.Type == "Content" && prop.Value == "null" {
			pval = newnode(xmlx.NT_ELEMENT, "null", "")
		} else {
			pval = newnode(xmlx.NT_TEXT, "", prop.Value)
		}
		pele.AddChild(pval)

		props.AddChildAt(leading(indent+2), -2)
		props.AddChildAt(pele, -2)
	}

	return node
}

// A map of base file names to xml nodes
type nodeMap map[string]*xmlx.Node

func walk(path string, info os.FileInfo, parent *xmlx.Node, uncles *nodeMap, ref *int, indent int) error {
	var child *xmlx.Node
	if info.IsDir() {
		// If possible, reuse a node created by a file that shares the same
		// base name. The sorting function ensures that the uncles map will be
		// populated with nodes, since files are traversed before directories.
		base := info.Name()
		node, ok := (*uncles)[base]
		if !ok {
			// Convert directories into Backpack objects
			item := Item{
				Class: "Backpack",
				Properties: []Property{
					{"string", "Name", info.Name()},
				},
			}

			node = item.Node(indent)

			node.SetAttr("referent", "RBX"+strconv.Itoa(*ref))
			*(ref)++

			parent.AddChildAt(leading(indent), -2)
			parent.AddChildAt(node, -2)
		}
		child = node
	} else {
		base, ext := splitName(info.Name())
		content, err := ioutil.ReadFile(path)
		if err != nil {
			return nil
		}

		var node *xmlx.Node
		if ext == ".lua" {
			// convert lua files into script objects
			subbase, subext := splitName(base)
			var item Item
			if subext == ".module" {
				// If the base name of the file ends with ".module"
				// (script.module.lua), then convert the file to a
				// ModuleScript object.
				item = Item{
					Class: "ModuleScript",
					Properties: []Property{
						{"string", "Name", subbase},
						{"ProtectedString", "Source", string(content)},
					},
				}
			} else {
				// Otherwise, convert it to a Script object.
				item = Item{
					Class: "Script",
					Properties: []Property{
						{"bool", "Disabled", "false"},
						{"Content", "LinkedSource", "null"},
						{"string", "Name", base},
						{"ProtectedString", "Source", string(content)},
					},
				}
			}

			node = item.Node(indent)

			node.SetAttr("referent", "RBX"+strconv.Itoa(*ref))
			*(ref)++

			parent.AddChildAt(leading(indent), -2)
			parent.AddChildAt(node, -2)
		} else if ext == ".rbxm" {
			// Insert contents of a roblox model file directly into the tree.
			doc := xmlx.New()
			if err := doc.LoadFile(path, nil); err != nil {
				return nil
			}

			nodes := doc.SelectNodes("", "Item")
			if len(nodes) == 0 {
				// Somehow, the model doesn't contain any objects.
				return nil
			}

			// Remap referents of items in this file
			refmap := make(map[string]string)
			items := doc.SelectNodesRecursive("", "Item")
			for _, item := range items {
				if item.HasAttr("", "referent") {
					newref := "RBX" + strconv.Itoa(*ref)
					refmap[item.As("", "referent")] = newref
					item.SetAttr("referent", newref)
					*(ref)++
				}
			}
			refs := doc.SelectNodesRecursive("", "Ref")
			for _, refnode := range refs {
				value := refnode.GetValue()
				if newref, ok := refmap[value]; ok {
					if len(refnode.Children) > 0 {
						refnode.Children[0].Value = newref
					}
				}
			}

			// Fix identation.
			fixIndentation(items, indent)
			fixIndentation(doc.SelectNodesRecursive("", "Properties"), indent)

			// Move each child node to new document
			for _, node := range nodes {
				parent.AddChildAt(leading(indent), -2)
				parent.AddChildAt(node, -2)
			}

			// Since a model may have multiple nodes, only the first node will
			// be paired with the model file.
			node = nodes[0]
		} else {
			// convert anything else into StringValue objects
			item := Item{
				Class: "StringValue",
				Properties: []Property{
					{"string", "Name", base},
					{"string", "Value", string(content)},
				},
			}

			node = item.Node(indent)
			node.SetAttr("referent", "RBX"+strconv.Itoa(*ref))
			*(ref)++

			parent.AddChildAt(leading(indent), -2)
			parent.AddChildAt(node, -2)
		}

		// Pair the node with the file's base name, which may be used later in
		// place of a directory that shares the same name.
		if _, ok := (*uncles)[base]; !ok {
			(*uncles)[base] = node
		}
		child = node
	}

	if !info.IsDir() {
		return nil
	}

	list, err := readDir(path)
	if err != nil {
		return nil
	}

	siblings := make(nodeMap)
	for _, fileInfo := range list {
		err = walk(filepath.Join(path, fileInfo.Name()), fileInfo, child, &siblings, ref, indent+1)
		if err != nil {
			return err
		}
	}
	return nil
}

func Walk(path string, node *xmlx.Node) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	ref := 0
	return walk(path, info, node, new(nodeMap), &ref, 1)
}

func readDir(dirname string) ([]os.FileInfo, error) {
	f, err := os.Open(dirname)
	if err != nil {
		return nil, err
	}
	list, err := f.Readdir(-1)
	f.Close()
	if err != nil {
		return nil, err
	}
	sort.Sort(byName(list))
	return list, nil
}

type byName []os.FileInfo

func (f byName) Len() int      { return len(f) }
func (f byName) Swap(i, j int) { f[i], f[j] = f[j], f[i] }
func (f byName) Less(i, j int) bool {
	dj := f[j].IsDir()
	if f[i].IsDir() == dj {
		// compare names of either two files, or two dirs
		return f[i].Name() < f[j].Name()
	} else {
		// put dirs at end of list
		return dj
	}
}

func splitName(s string) (string, string) {
	ext := filepath.Ext(s)
	return s[:len(s)-len(ext)], ext
}

func WriteRBXM(input string) (out []byte, err error) {
	doc := xmlx.New()
	doc.SaveDocType = false

	root := newnode(xmlx.NT_ELEMENT, "roblox", "")
	root.AddChild(leading(0))
	doc.Root = root

	root.SetAttr("xmlns:xmime", "http://www.w3.org/2005/05/xmlmime")
	root.SetAttr("xmlns:xsi", "http://www.w3.org/2001/XMLSchema-instance")
	root.SetAttr("xsi:noNamespaceSchemaLocation", "http://www.roblox.com/roblox.xsd")
	root.SetAttr("version", "4")

	ext := newnode(xmlx.NT_ELEMENT, "External", "")
	ext.AddChild(newnode(xmlx.NT_TEXT, "", "null"))
	root.AddChildAt(leading(1), -2)
	root.AddChildAt(ext, -2)

	ext = newnode(xmlx.NT_ELEMENT, "External", "")
	ext.AddChild(newnode(xmlx.NT_TEXT, "", "nil"))
	root.AddChildAt(leading(1), -2)
	root.AddChildAt(ext, -2)

	err = Walk(input, root)
	if err != nil {
		return out, err
	}

	// Reorder referents of items in the tree
	ref := 0
	refmap := make(map[string]string)
	items := doc.SelectNodesRecursive("", "Item")
	for _, item := range items {
		if item.HasAttr("", "referent") {
			newref := "RBX" + strconv.Itoa(ref)
			refmap[item.As("", "referent")] = newref
			item.SetAttr("referent", newref)
			ref++
		}
	}
	refs := doc.SelectNodesRecursive("", "Ref")
	for _, refnode := range refs {
		value := refnode.GetValue()
		if newref, ok := refmap[value]; ok {
			if len(refnode.Children) > 0 {
				refnode.Children[0].Value = newref
			}
		}
	}

	return doc.SaveBytes(), nil
}

func init() {
	xmlx.IndentPrefix = "\t"
	xmlx.CollapseEmpty = false
	xmlx.EscapeText = func(w io.Writer, s []byte) error {
		var esc []byte
		for i := 0; i < len(s); i++ {
			c := s[i]
			switch c {
			case byte('"'):
				esc = []byte("&quot;")
			case byte('\''):
				esc = []byte("&apos;")
			case byte('&'):
				esc = []byte("&amp;")
			case byte('<'):
				esc = []byte("&lt;")
			case byte('>'):
				esc = []byte("&gt;")
			case byte('\n'):
				esc = []byte("\n")
			case byte('\r'):
				esc = []byte("\r")
			default:
				if c >= 32 && c <= 126 {
					esc = []byte{c}
				} else {
					esc = []byte("&#" + strconv.Itoa(int(c)) + ";")
				}
			}
			if _, err := w.Write(esc); err != nil {
				return err
			}
		}
		return nil
	}
}
