package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/textproto"
	"os"
	"path/filepath"

	"github.com/Jeffail/gabs"
	"github.com/fjl/go-couchdb"
	"github.com/philippfranke/multipart-related/related"
)

//Save multipart/related document in CouchDB
//This type of function should be part of CouchDB driver framework
//also it must be part of replication mechanism

/*
 document structure:

--5u93-0
content-type: application/json

{
	"attribute_0": "value",
	...

	"_attachments": {
		"attachment_relative_path/filename: {
			"follows":true,
			"content_type":"detect_mime_type_from_file_extension",
			"length": bytes
		},

	...

	}
}

--5u93-0
Content-type: detect_mime_Type_from_file_extension

Kill Roy was here!
--5u93-0
Content-type: detect_mime_Type_from_file_extension

...
--5u93-0--
*/
func SaveMPRDoc(doc_name string) {

	//Build documents list for attachments
	//search _attachments folder and use it
	//as starting base for the relative path

	//Build CouchDB specific design documents
	//views, libs, lists, shows, filters, updates, ???

	//Build the JSON part
	CDBDoc := gabs.New()
	CDBDoc.Set(true, "_attachments", "pufi.png", "follows")
	CDBDoc.Set("image/png", "_attachments", "pufi.png", "content_type")
	//Get attachement length in bytes
	CDBDoc.Set(3010, "_attachments", "pufi.png", "length")

	//fmt.Println(doc_name, CDBDoc.String())

	//add a text part
	CDBDoc.Set(true, "_attachments", "salam/de/sibiu/text.txt", "follows")
	CDBDoc.Set("text/plain", "_attachments", "salam/de/sibiu/text.txt", "content_type")
	CDBDoc.Set(10, "_attachments", "salam/de/sibiu/text.txt", "length")

	b := &bytes.Buffer{}
	w := related.NewWriter(b)
	w.SetBoundary(MPBoundary)

	rootPart, err := w.CreateRoot("", MPContent_Type, nil)
	if err != nil {
		panic(err)
	}
	//Write JSON to document
	rootPart.Write([]byte(CDBDoc.String()))

	//Construct attached parts
	header := make(textproto.MIMEHeader)
	header.Set("Content-Type", "image/png") //\r\nContent-transfer-encoding: binary")
	nextPart, err := w.CreatePart("", header)
	if err != nil {
		panic(err)
	}

	// Add your image file
	file := "logo.png"
	f, err := os.Open(file)
	if err != nil {
		return
	}
	defer f.Close()

	data, err := ioutil.ReadAll(f)
	if err != nil {
		panic(err)
	}
	nextPart.Write(data)

	//continue with text part
	header.Set("Content-Type", "plain/text")
	nextPart, err = w.CreatePart("", header)
	if err != nil {
		panic(err)
	}
	nextPart.Write([]byte("0123456789"))

	if err := w.Close(); err != nil {
		panic(err)
	}

	//fmt.Printf("The compound Object Content-Type:\n %s \n", w.FormDataContentType())
	if Verbose {
		fmt.Println("+-------+")
		fmt.Printf("Body: \n %s", b.String())
		fmt.Println("+-------+")
	}

	serverConnection, err := couchdb.NewClient(ServerURL, nil)
	if err != nil {
		fmt.Println(err)
	}
	workingDB, err := serverConnection.EnsureDB("apptest") //(DBDir)
	if err != nil {
		fmt.Println(err)
	}

	rev, err := workingDB.Rev("pufi")
	if err != nil {
		fmt.Println(err)
	}

	targetUrl := "http://127.0.0.1:5984/apptest/" + doc_name

	if len(rev) > 0 {
		targetUrl += "?rev=" + rev
	}
	request, err := http.NewRequest("PUT", targetUrl, b)

	request.SetBasicAuth("root", "root")
	request.Header.Set("Content-Type", "multipart/related;boundary="+MPBoundary)
	request.Close = true

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		panic(err)
	}
	defer response.Body.Close()
	contents, err := ioutil.ReadAll(response.Body)
	if err != nil {
		panic(err)
	}

	if Verbose {
		fmt.Println("   ", response.StatusCode)
		hdr := response.Header
		for key, value := range hdr {
			fmt.Println("   ", key, ":", value)
		}
		fmt.Println(string(contents))
	}

}

//Push design documents from local directory to
//CouchDB, multiple database are accepted
func Push() {
	/*
		0) get input parameters: directory, server URL
		   directory name -> database name
		   subdirectory_level_1 -> document name
		   subdirectory_level_2 -> attributes name
		1) recognize document structure: data, attachments,
		   design doc fields (views, lists, libs, shows, rewrites, validate_doc_update, update, index etc).
		2) build document
		3) save document
	*/
	/* -----------------------------------------------------------------------
	   Push local directory structure to database
	----------------------------------------------------------------------- */
	//Get connection URL

	//Get database directory

	//Get document directory

	//Check for manifest.json file
	is_file, err := FileExists(filepath.Join(pwd, "manifest.json"))
	if is_file {
		if Verbose {
			fmt.Println("An AppZip repository - maybe?!?")
		}
		//This may be an appzip repository
	}
	//Check database directory
	db_list, err := ListDir(pwd)
	if Verbose {
		fmt.Println(db_list)
	}
	//Check for database on CouchDB server
	serverConnection, err := couchdb.NewClient(ServerURL, nil)
	if err != nil {
		fmt.Println(err)
	}
	workingDB, err := serverConnection.EnsureDB("apptest") //(DBDir)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(workingDB.Name())

	//In database folder
	//Check for documents folders
	//Check for design documents structure
	//Check for attachemnts structure
	//Check for document on CouchDB server

	//Save multipart/related document to CouchDB
	SaveMPRDoc("pufi")

}
