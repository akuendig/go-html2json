package hello

import (
	"code.google.com/p/goweb/goweb"
	"encoding/json"
	"exp/html"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"appengine"
	"appengine/urlfetch"
)

type Error struct {
	Error, Message string
}

func newError(code int, err error) *Error {
	return &Error{
		Error:   http.StatusText(code),
		Message: err.Error(),
	}
}

type Tag struct {
	Data       string
	Attributes []html.Attribute
	Children   []*Tag
	Type       html.NodeType
}

func newTag(n *html.Node) *Tag {
	var t = &Tag{
		Data:       strings.Replace(n.Data, "\\xa6", "", -1),
		Attributes: n.Attr,
		Children:   nil,
		Type:       n.Type,
	}

	for _, child := range n.Child {
		t.Children = append(t.Children, newTag(child))
	}

	return t
}

func init() {
	goweb.MapFunc("/", home, goweb.GetMethod)
	goweb.MapFunc("/", post, goweb.PostMethod)

	http.Handle("/", goweb.DefaultHttpHandler)
}

func home(c *goweb.Context) {
	fmt.Fprint(c.ResponseWriter, `
Post an url to this address to get back its json representation.
Node types are enumerated as follows:

    ErrorNode NodeType  = 0
    TextNode            = 1
    DocumentNode        = 2
    ElementNode         = 3
    CommentNode         = 4
    DoctypeNode         = 5

`)
}

func post(c *goweb.Context) {
	var ctx = appengine.NewContext(c.Request)
	var client = urlfetch.Client(ctx)

	url, err := ioutil.ReadAll(c.Request.Body)

	if err != nil {
		handleError(c, ctx, err)
		return
	}

	resp, err := client.Get(string(url))

	if err != nil {
		handleError(c, ctx, err)
		return
	}

	defer resp.Body.Close()
	node, err := html.Parse(resp.Body)

	if err != nil {
		handleError(c, ctx, err)
		return
	}

	var enc = json.NewEncoder(c.ResponseWriter)

	if err := enc.Encode(newTag(node)); err != nil {
		handleError(c, ctx, err)
		return
	}
}

func handleError(c *goweb.Context, ctx appengine.Context, err error) {
	var enc = json.NewEncoder(c.ResponseWriter)

	ctx.Errorf("%v", err)

	if err := enc.Encode(newError(http.StatusInternalServerError, err)); err != nil {
		ctx.Errorf("%v", err)
		fmt.Fprintln(c.ResponseWriter, http.StatusText(http.StatusInternalServerError))
		fmt.Fprintln(c.ResponseWriter, err)
		return
	}
}
