package renderer

import (
	"errors"
	"strings"

	"github.com/88250/lute/html"
)

func getNodeValue(content string, selector string, attrKey string) string {
	node, err := getNode(content, selector)
	if err != nil {
		return ""
	}
	if attrKey == "" {
		if node.FirstChild == nil {
			return ""
		}
		return strings.TrimSpace(node.FirstChild.Data)
	}
	for _, attr := range node.Attr {
		if attr.Key == attrKey {
			return attr.Val
		}
	}
	return ""
}

func getNode(content string, selector string) (*html.Node, error) {
	htmlTree, err := html.Parse(strings.NewReader(content))
	if err != nil {
		return nil, err
	}
	var nodes *html.Node
	var findNodes func(*html.Node) error
	findNodes = func(n *html.Node) error {
		if n.Type == html.ElementNode && n.Data == selector {
			nodes = n
			return nil
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if err = findNodes(c); err == nil {
				return nil
			}
		}
		return errors.New("search nodes error")
	}
	err = findNodes(htmlTree)
	if err != nil {
		return nil, err
	}
	if nodes == nil {
		return nil, errors.New("not found nodes")
	}
	return nodes, nil
}
