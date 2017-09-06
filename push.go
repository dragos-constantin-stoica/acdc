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

const (
	MPBoundary     = "5u93-0"
	MPContent_Type = "application/json"
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

	//Build CouchDB specific design documents structure
	//views, libs, lists, shows, filters, updates,

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

//Upsert a document to CouchDB
func upsert(doc CouchDoc) (string, error) {
	if rev, err := workingDB.Rev(doc["_id"].(string)); err == nil {
		return workingDB.Put(doc["_id"].(string), doc, rev)
	} else if couchdb.NotFound(err) {
		return workingDB.Put(doc["_id"].(string), doc, "")
	} else {
		return "", err
	}
}

//Push design documents from local directory to
//CouchDB, multiple database are accepted
func Push() {
	/*
		-------------------------------------------------
		   Push local directory structure to database
		-------------------------------------------------
			0) get input parameters: directory, server URL
			   directory name -> database name
			   subdirectory_level_1 -> document name
			   subdirectory_level_2 -> attributes name
			                        -> file doc.json must be present and must contain _id
			1) recognize document structure: _id, fileds, attachments,
			   design doc fields (views, libs, lists, shows, rewrites, validate_doc_update, update, full_text-index etc).
			2) build document
			3) upsert document to CouchDB
	*/

	//Get connection URL
	if DEBUG {
		fmt.Printf("URL: %s\n", ServerURL)
	}
	//Get database directory
	if DEBUG {
		fmt.Printf("Database name: %s\n", DBName)
	}

	//Check for manifest.json file
	is_file, err := FileExists(filepath.Join(pwd, "manifest.json"))
	if is_file {
		if Verbose {
			fmt.Println("An AppZip repository - maybe?!?")
		}
		//This may be an appzip repository
		//TODO - implement appzip repository structure processing and push
	}

	//Check for database on CouchDB server
	serverConnection, err = couchdb.NewClient(ServerURL, nil)
	if err != nil {
		fmt.Println(err)
	}
	workingDB, err = serverConnection.EnsureDB(DBName)
	if err != nil {
		fmt.Println(err)
	}
	if DEBUG {
		fmt.Printf("Working with %s database.\n", workingDB.Name())
	}

	//In database folder
	//Check for documents folders
	doc_list, err := ListDir(filepath.Join(pwd, DBName))
	if Verbose {
		fmt.Printf("Document folders: %s\n", doc_list)
	}
	for _, doc := range doc_list {
		//Check for doc.json file and _id field inside
		is_file, err = FileExists(filepath.Join(pwd, DBName, doc, "doc.json"))
		var tmpDoc CouchDoc
		if is_file {
			if Verbose {
				fmt.Printf("Found doc.json file in %s\n", doc)
			}
			tmpDoc, err = loadJSON(filepath.Join(pwd, DBName, doc, "doc.json"))
			if err != nil {
				fmt.Print(err)
			}
			if tmpDoc["_id"] == nil {
				fmt.Printf("The doc.json does not contain field _id. Found %s\n", tmpDoc)
			} else {
				fmt.Printf("Read %s\n", tmpDoc)
			}

		}

		//Check for design documents structure:
		//rewrites, validate_doc_update, filters, shows, updates, lists, views, fulltext

		//rewrites may be written directly inside doc.json
		if is_file, _ = FileExists(filepath.Join(pwd, DBName, doc, "rewrites.js")); is_file {
			if Verbose {
				fmt.Printf("Found rewrites.js file in %s\n", doc)
			}

			tmpDoc["rewrites"], err = loadFile(filepath.Join(pwd, DBName, doc, "rewrites.js"))
			if err != nil {
				fmt.Print(err)
			}
		}

		//validate_doc_update may be written directly inside doc.json
		if is_file, _ = FileExists(filepath.Join(pwd, DBName, doc, "validate_doc_update.js")); is_file {
			if Verbose {
				fmt.Printf("Found validate_doc_update.js file in %s\n", doc)
			}

			tmpDoc["validate_doc_update"], err = loadFile(filepath.Join(pwd, DBName, doc, "validate_doc_update.js"))
			if err != nil {
				fmt.Print(err)
			}
		}

		//filters may be written directly inside doc.json
		if is_file, _ = FileExists(filepath.Join(pwd, DBName, doc, "filters")); is_file {
			if Verbose {
				fmt.Printf("Found filters folder in %s\n", doc)
			}
			//inside filters folder there are files
			//each file contains the code for a filter function
			//the name of the file is the name of the function
			file_list, err := ListFile(filepath.Join(pwd, DBName, doc, "filters"))
			if err != nil {
				fmt.Print(err)
			}
			if DEBUG {
				fmt.Printf("Files found %s\n", file_list)
			}
			tmpFilters := make(map[string]interface{})
			for _, source_file := range file_list {
				tmpFilters[stripExtension(source_file)], err = loadFile(filepath.Join(pwd, DBName, doc, "filters", source_file))
				if err != nil {
					fmt.Print(err)
				}
			}
			tmpDoc["filters"] = tmpFilters
		}

		//shows may be written directly inside doc.json
		if is_file, _ = FileExists(filepath.Join(pwd, DBName, doc, "shows")); is_file {
			if Verbose {
				fmt.Printf("Found shows folder in %s\n", doc)
			}
			//inside shows folder there are files
			//each file contains the code for a show function
			//the name of the file is the name of the function
			file_list, err := ListFile(filepath.Join(pwd, DBName, doc, "shows"))
			if err != nil {
				fmt.Print(err)
			}
			if DEBUG {
				fmt.Printf("Files found %s\n", file_list)
			}
			tmpShows := make(map[string]interface{})
			for _, source_file := range file_list {
				tmpShows[stripExtension(source_file)], err = loadFile(filepath.Join(pwd, DBName, doc, "shows", source_file))
				if err != nil {
					fmt.Print(err)
				}
			}
			tmpDoc["shows"] = tmpShows
		}

		//updates may be written directly inside doc.json
		if is_file, _ = FileExists(filepath.Join(pwd, DBName, doc, "updates")); is_file {
			if Verbose {
				fmt.Printf("Found updates folder in %s\n", doc)
			}
			//inside updates folder there are files
			//each file contains the code for an update function
			//the name of the file is the name of the function
			file_list, err := ListFile(filepath.Join(pwd, DBName, doc, "updates"))
			if err != nil {
				fmt.Print(err)
			}
			if DEBUG {
				fmt.Printf("Files found %s\n", file_list)
			}
			tmpUpdates := make(map[string]interface{})
			for _, source_file := range file_list {
				tmpUpdates[stripExtension(source_file)], err = loadFile(filepath.Join(pwd, DBName, doc, "updates", source_file))
				if err != nil {
					fmt.Print(err)
				}
			}
			tmpDoc["updates"] = tmpUpdates
		}

		//lists may be written directly inside doc.json
		if is_file, _ = FileExists(filepath.Join(pwd, DBName, doc, "lists")); is_file {
			if Verbose {
				fmt.Printf("Found lists folder in %s\n", doc)
			}
			//inside lists folder there are files
			//each file contains the code for a list function
			//the name of the file is the name of the function
			file_list, err := ListFile(filepath.Join(pwd, DBName, doc, "lists"))
			if err != nil {
				fmt.Print(err)
			}
			if DEBUG {
				fmt.Printf("Files found %s\n", file_list)
			}
			tmpLists := make(map[string]interface{})
			for _, source_file := range file_list {
				tmpLists[stripExtension(source_file)], err = loadFile(filepath.Join(pwd, DBName, doc, "lists", source_file))
				if err != nil {
					fmt.Print(err)
				}
			}
			tmpDoc["lists"] = tmpLists
		}

		//fulltext may be written directly inside doc.json
		if is_file, _ = FileExists(filepath.Join(pwd, DBName, doc, "updates")); is_file {
			if Verbose {
				fmt.Printf("Found fulltext folder in %s\n", doc)
			}
			//inside fulltext folder there are files
			//each file contains the code for an index function
			//the name of the file is the name of the index
			file_list, err := ListFile(filepath.Join(pwd, DBName, doc, "fulltext"))
			if err != nil {
				fmt.Print(err)
			}
			if DEBUG {
				fmt.Printf("Files found %s\n", file_list)
			}
			tmpFulltext := make(map[string]interface{})
			for _, source_file := range file_list {
				tmpFulltext[stripExtension(source_file)] = map[string]interface{}{"index": ""}
				tmpFulltext[stripExtension(source_file)].(map[string]interface{})["index"], err = loadFile(filepath.Join(pwd, DBName, doc, "fulltext", source_file))
				if err != nil {
					fmt.Print(err)
				}
			}
			tmpDoc["fulltext"] = tmpFulltext
		}

		//views may be written directly inside doc.json
		if is_file, _ = FileExists(filepath.Join(pwd, DBName, doc, "views")); is_file {
			if Verbose {
				fmt.Printf("Found views folder in %s\n", doc)
			}
			//inside views folder there are files
			//each file contains the code for a map, reduce or CommonJS function
			//the name of the file is the name of the view
			file_list, err := ListFile(filepath.Join(pwd, DBName, doc, "views"))
			if err != nil {
				fmt.Print(err)
			}
			if DEBUG {
				fmt.Printf("Files found %s\n", file_list)
			}
			tmpViews := make(map[string]map[string]string)
			for _, source_file := range file_list {
				view_name, view_type := splitFileNameforView(source_file)
				//tmpViews[view_name][view_type] = {view_name: {view_type: ""}}
				vn, ok := tmpViews[view_name]
				if !ok {
					vn = make(map[string]string)
					tmpViews[view_name] = vn
				}
				tmpViews[view_name][view_type], err = loadFile(filepath.Join(pwd, DBName, doc, "views", source_file))
				if DEBUG {
					fmt.Print("View data %s\n", tmpViews)
				}
				if err != nil {
					fmt.Print(err)
				}
			}
			tmpDoc["views"] = tmpViews
		}

		//Upsert to CouchDB
		rev, err = upsert(tmpDoc)
		if err != nil {
			fmt.Printf("%s", err.Error())
		}
		fmt.Printf("rev: %s\n", rev)

		//Check for attachemnts folder
		//Save multipart/related document to CouchDB
		//SaveMPRDoc("pufi")

	}

}
