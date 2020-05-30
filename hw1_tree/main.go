package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
)

// Node of our directories tree
type Node struct {
	File     os.FileInfo
	Children []Node
}

// Name of the directory or name of the file with it's size
func (n Node) Name() string {
	if n.File.IsDir() {
		return n.File.Name()
	}

	return fmt.Sprintf("%s (%s)", n.File.Name(), n.Size())
}

// Size of the file
func (n Node) Size() string {
	if n.File.Size() > 0 {
		return fmt.Sprintf("%db", n.File.Size())
	}

	return "empty"
}

func getNodes(path string, shouldIncludeFiles bool) ([]Node, error) {
	var nodes []Node

	files, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}

	for _, f := range files {
		if !shouldIncludeFiles && !f.IsDir() {
			continue
		}

		node := Node{
			File: f,
		}

		if f.IsDir() {
			children, err := getNodes(path+string(os.PathSeparator)+f.Name(), shouldIncludeFiles)
			if err != nil {
				return nil, err
			}

			node.Children = children
		}

		nodes = append(nodes, node)
	}

	return nodes, nil
}

func printNodes(out io.Writer, nodes []Node, parentPrefix string) {
	lastID, prefix, childPrefix := len(nodes)-1, "├───", "│\t"

	for i, node := range nodes {
		if i == lastID {
			prefix = "└───"
			childPrefix = "\t"
		}

		fmt.Fprint(out, parentPrefix, prefix, node.Name(), "\n")

		if node.File.IsDir() {
			printNodes(out, node.Children, parentPrefix+childPrefix)
		}
	}
}

func dirTree(out io.Writer, path string, shouldIncludeFiles bool) error {
	nodes, err := getNodes(path, shouldIncludeFiles)
	if err != nil {
		return err
	}

	printNodes(out, nodes, "")
	return nil
}

func main() {
	out := os.Stdout
	if !(len(os.Args) == 2 || len(os.Args) == 3) {
		panic("usage go run main.go . [-f]")
	}
	path := os.Args[1]
	printFiles := len(os.Args) == 3 && os.Args[2] == "-f"
	err := dirTree(out, path, printFiles)
	if err != nil {
		panic(err.Error())
	}
}
