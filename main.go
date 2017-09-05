//
// @author: Dragos STOICA
// @date: 01.03.2016
//
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/fjl/go-couchdb"
)

const (
	MPBoundary     = "5u93-0"
	MPContent_Type = "application/json"
	DEBUG          = true
)

var (
	ServerURL string //full URL: PROTOCOL://USER:PASSWORD@SERVER:PORT/
	DBName    string //database name to copy on local directory
	DDocList  string //desing documents list, comma separated to be copied to local directory
	Verbose   bool   //verbose flag
	OP_Pull   bool   //pull flag
	OP_Push   bool   //push flag

	serverConnection couchdb.Client
	workingDB        couchdb.DB
	rev              string

	pwd string

	DDocFileds = []string{
		"_attachments",
		"_views",
		"_lists",
		"_shows",
		"_updates",
		"_filters",
		"_rewrites",
		"validate_doc_update",
		"_fulltext"}
)

func main() {

	if (!OP_Pull && !OP_Push) || (OP_Pull && OP_Push) {
		fmt.Println("Use -push or -pull option.")
		os.Exit(0)
	}

	if OP_Pull {
		Pull()
	}

	if OP_Push {
		Push()
	}
}

func init() {
	flag.StringVar(&ServerURL, "URL", "http://root:root@localhost:5984/", "specify full `URL` to access CouchDB server")

	flag.BoolVar(&OP_Push, "push", false, "sync local directory with CouchDB documents")
	pwd, _ = os.Getwd()
	flag.BoolVar(&Verbose, "verbose", false, "verbose mode")

	flag.BoolVar(&OP_Pull, "pull", false, "sync documents from CouchDB to local directory")
	flag.StringVar(&DBName, "db", "", "`database` name to be pulled")
	flag.StringVar(&DDocList, "ddoc", "", " list of `design documents` IDs to be pulled, list is comma separated")

	//TODO - have a look here on how to build the CLI
	//https://blog.komand.com/build-a-simple-cli-tool-with-golang

	flag.Parse()

	if DEBUG {
		Verbose = true // For DEBUG purposes
	}
}
