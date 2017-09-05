//
// @author: Dragos STOICA
// @date: 01.03.2016
//
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"

	"github.com/Jeffail/gabs"
	"github.com/fjl/go-couchdb"
	"github.com/philippfranke/multipart-related/related"
)

const (
	MPBoundary     = "5u93-0"
	MPContent_Type = "application/json"
	DEBUG          = true
)

var (
	ServerURL string //full URL: PROTOCOL://USER:PASSWORD@SERVER/
	ProjDir   string //full path to database directory on the local machine. Directory name = Database name
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

//Pull design documents form a CouchDB database
//to local directory for editing
func Pull() {
	/* -----------------------------------------------------------------------
	   Pull - get design documents from CouchDB and copy them to local folders
	   the existing local files are overwritten
	   Mandatory parameters
	   db [database_name]
	   ddoc [array_of_ddocs], comma separated no spaces
	----------------------------------------------------------------------- */
	//Check mandatory parameters
	if len(DBName) == 0 {
		fmt.Println("Please specify db parameter!")
		os.Exit(0)
	}

	if len(DDocList) == 0 {
		fmt.Println("Please specify ddoc parameter!")
		os.Exit(0)
	}

	ddoc := strings.Split(DDocList, ",")
	//Check if database exists on server
	serverConnection, err := couchdb.NewClient(ServerURL, nil)
	if err != nil {
		fmt.Println(err)
	}
	workingDB, err := serverConnection.EnsureDB(DBName) //(DBDir)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(workingDB.Name() + " found on server " + ServerURL)
	//Check local directory for pull destination
	is_dir, err := FileExists(filepath.Join(ProjDir, DBName))
	if is_dir {
		fmt.Println("Found database folder on local disk!")
	} else {
		//create DB folder of local directory
		if os.Mkdir(filepath.Join(ProjDir, DBName), os.ModePerm) != nil {
			fmt.Println("Can not create directory " + DBName)
		} else {
			fmt.Println("Local directory for database created!")
		}
	}
	//For each ddoc
	for _, one_doc := range ddoc {
		//Check if ddoc exists in the database
		etag, err := ETag(DBName, one_doc)
		if err != nil {
			fmt.Println("Document " + one_doc + " not found in database")
			continue
		}
		fmt.Println("Document " + one_doc + " found with rev:" + etag)
		//Get the ddoc and 'serialize' it to local directory and files
		is_dir, err := FileExists(filepath.Join(ProjDir, DBName, one_doc))
		if !is_dir {
			if os.MkdirAll(filepath.Join(ProjDir, DBName, strings.Replace(one_doc, "_design/", "", -1)), os.ModePerm) != nil {
				fmt.Println("Can not create subdirectory " + one_doc)
			} else {
				fmt.Println("Local directory for ddoc created!")
			}
		}
		//Get ddoc from DB
		var tmpdoc CouchDoc //interface{}
		if workingDB.Get(one_doc, &tmpdoc, nil) != nil {
			fmt.Println("Error fethcing document form DB!")
		}
		//fmt.Println(tmpdoc)
		//Handle special fileds
		if tmpdoc["_attachments"] != nil {
			//fmt.Print("attachments >> ")
			//fmt.Println(tmpdoc["_attachments"])
			for att_doc, _ := range tmpdoc["_attachments"].(map[string]interface{}) {
				//fmt.Println(att_doc)
				att_doc_relpath, _ := filepath.Split(att_doc)
				att_doc_path := filepath.Join(ProjDir, DBName, strings.Replace(one_doc, "_design/", "", -1), "_attachments",
					strings.Join(strings.Split(att_doc, "/"), string(os.PathSeparator)))
				is_file, err := FileExists(att_doc_path)
				if is_file {
					os.Remove(att_doc_path)
				}
				os.MkdirAll(filepath.Join(ProjDir, DBName, strings.Replace(one_doc, "_design/", "", -1), "_attachments", att_doc_relpath), os.ModePerm)
				output, err := os.Create(att_doc_path)
				if err != nil {
					fmt.Println("Error while creating", att_doc, "-", err)
				}
				defer output.Close()
				response, err := http.Get(ServerURL + "/" + DBName + "/" + one_doc + "/" + att_doc)
				if err != nil {
					fmt.Println("Error while downloading", att_doc, "-", err)
				}
				defer response.Body.Close()

				n, err := io.Copy(output, response.Body)
				if err != nil {
					fmt.Println("Error while downloading", att_doc, "-", err)
				}
				fmt.Printf("%s size %v bytes downloaded\n", att_doc, n)
			}
		}

		if tmpdoc["views"] != nil {
			//fmt.Println(tmpdoc["views"])
			for view_fct, _ := range tmpdoc["views"].(map[string]interface{}) {
				view_fct_path := filepath.Join(ProjDir, DBName, strings.Replace(one_doc, "_design/", "", -1), "views", view_fct)
				is_dir, err := FileExists(view_fct_path)
				if is_dir {
					os.Remove(view_fct_path)
				}
				os.MkdirAll(view_fct_path, os.ModePerm)
				//TODO - The extension should be determined by "language" attribute
				output, err := os.Create(filepath.Join(view_fct_path, "map.js"))
				if err != nil {
					fmt.Println("Error while creating", view_fct_path, "-", err)
				}
				defer output.Close()
				n, err := output.WriteString(tmpdoc["views"].(map[string]interface{})[view_fct].(map[string]interface{})["map"].(string))
				if err != nil {
					fmt.Println("Error while writing map function for ", view_fct, "-", err)
				}
				fmt.Printf("%s map - %v bytes written\n", view_fct, n)
				if tmpdoc["views"].(map[string]interface{})[view_fct].(map[string]interface{})["reduce"] != nil {
					//Write reduce function in reduce.js file
					routput, err := os.Create(filepath.Join(view_fct_path, "reduce.js"))
					if err != nil {
						fmt.Println("Error while creating", view_fct_path, "-", err)
					}
					defer routput.Close()
					n, err := routput.WriteString(tmpdoc["views"].(map[string]interface{})[view_fct].(map[string]interface{})["reduce"].(string))
					if err != nil {
						fmt.Println("Error while writing reduce function for ", view_fct, "-", err)
					}
					fmt.Printf("%s reduce - %v bytes written\n", view_fct, n)
				} else {
					fmt.Println("No reduce function for view " + view_fct)
				}
			}
		}

		//The other fields will be dumped in doc_attributes.json
		//except rev
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
	is_file, err := FileExists(filepath.Join(ProjDir, "manifest.json"))
	if is_file {
		if Verbose {
			fmt.Println("An AppZip repository - maybe?!?")
		}
		//This may be an appzip repository
	}
	//Check database directory
	db_list, err := ListDir(ProjDir)
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

func main() {

	flag.StringVar(&ServerURL, "URL", "http://root:root@localhost:5984/", "specify full `URL` to access CouchDB server")

	flag.BoolVar(&OP_Push, "push", false, "sync local directory with CouchDB documents")
	pwd, _ = os.Getwd()
	flag.StringVar(&ProjDir, "project", pwd, "full path to project `directory` where databases are stored")
	flag.BoolVar(&Verbose, "verbose", false, "verbose mode")

	flag.BoolVar(&OP_Pull, "pull", false, "sync documents from CouchDB to local directory")
	flag.StringVar(&DBName, "db", "", "`database` name to be pulled")
	flag.StringVar(&DDocList, "ddoc", "", " list of `design documents` IDs to be pulled, list is comma separated")

	flag.Parse()

	if DEBUG {
		Verbose = true // For DEBUG purposes
	}

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
}
