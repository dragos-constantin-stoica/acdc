package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"mime"
	"net/http"
	"net/textproto"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

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
content of the document to be posted:

--5u93-0
content-type: application/json

{
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
func SaveMPRDoc(doc CouchDoc, attachments_path string) {

	doc_name := doc["_id"].(string)
	//Build documents list for attachments
	//search _attachments folder and use it
	//as starting base for the relative path

	//Build the JSON part
	oo := &bytes.Buffer{}
	enc := json.NewEncoder(oo)
	if err := enc.Encode(&doc); err != nil {
		fmt.Println(err)
	}
	CDBDoc, err := gabs.ParseJSON(oo.Bytes())

	//for all files in the current folder construct an object
	//repeat for subfolders the procedure
	BuildJSONforAttachments(attachments_path, CDBDoc)

	if DEBUG {
		fmt.Println(doc_name, CDBDoc.String())
	}

	//Create the multipart/related document
	b := &bytes.Buffer{}
	w := related.NewWriter(b)
	w.SetBoundary(MPBoundary)

	rootPart, err := w.CreateRoot("", MPContent_Type, nil)
	if err != nil {
		fmt.Print(err)
	}
	//Write JSON to document
	tmpDDocString := CDBDoc.String()
	rootPart.Write([]byte(tmpDDocString))
	header := make(textproto.MIMEHeader)

	q := regexp.MustCompile(":\\{.*?\\}")
	tmpDDocString = strings.Replace(strings.Replace(tmpDDocString, "}}", "", -1), "{\"_attachments\":{", "", -1)
	attachments_list := strings.Split(strings.Replace(q.ReplaceAllString(tmpDDocString, ""), "\"", "", -1), ",")

	for _, child := range attachments_list {
		//Construct attached parts
		header.Set("Content-Type", mime.TypeByExtension(path.Ext(child))) //\r\nContent-transfer-encoding: binary")
		nextPart, err := w.CreatePart("", header)
		if err != nil {
			fmt.Print(err)
		}

		// Add file content
		f, err := os.Open(path.Join(attachments_path, child))
		if err != nil {
			fmt.Print(err)
		}

		data, err := ioutil.ReadAll(f)
		if err != nil {
			fmt.Print(err)
		}
		nextPart.Write(data)
		f.Close()

	}

	if err := w.Close(); err != nil {
		fmt.Print(err)
	}

	//Save the document to CouchDB database
	targetUrl := ServerURL + "/" + DBName + "/" + doc_name

	if rev, err = workingDB.Rev(doc_name); err == nil {
		targetUrl += "?rev=" + rev
	}

	request, err := http.NewRequest("PUT", targetUrl, b)

	//request.SetBasicAuth("root", "root")
	request.Header.Set("Content-Type", "multipart/related;boundary="+MPBoundary)
	request.Close = true

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		fmt.Print(err)
	}
	defer response.Body.Close()
	contents, err := ioutil.ReadAll(response.Body)
	if err != nil {
		fmt.Print(err)
	}

	if Verbose {
		fmt.Println("   ", response.StatusCode)
		hdr := response.Header
		for key, value := range hdr {
			fmt.Println("   ", key, ":", value)
		}
	}
	fmt.Printf("new multipart/related document saved %s", string(contents))

}

//Upsert a document to CouchDB
func upsert(doc CouchDoc) (result string, err error) {
	if rev, err = workingDB.Rev(doc["_id"].(string)); err == nil {
		return workingDB.Put(doc["_id"].(string), doc, rev)
	} else if couchdb.NotFound(err) {
		return workingDB.Put(doc["_id"].(string), doc, "")
	} else {
		return result, err
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

	//In database folder
	//Check for documents folders
	doc_list, err := ListDir(filepath.Join(pwd, DBName))
	if Verbose {
		fmt.Printf("Document folders: %s\n", doc_list)
	}
	docs := strings.Split(DocsList, ",")
	if len(docs[0]) == 0 {
		docs = []string{}
	}
	for _, doc := range doc_list {
		//Check for doc.json file and _id field inside
		var tmpDoc CouchDoc
		tmpDoc = make(map[string]interface{})
		if is_file, err := FileExists(filepath.Join(pwd, DBName, doc, "doc.json")); is_file {
			if Verbose {
				fmt.Printf("Found doc.json file in %s\n", doc)
			}
			tmpDoc, err = loadJSON(filepath.Join(pwd, DBName, doc, "doc.json"))
			if err != nil {
				fmt.Print(err)
			}
			if tmpDoc["_id"] == nil {
				fmt.Printf("The doc.json does not contain field _id. Found %s. Skipping folder.\n", tmpDoc)
				continue
			} else {
				fmt.Printf("Read %s", tmpDoc.toJSON())
				//Check if the document is in input document list
				if len(docs) > 0 {
					if ok, _ := in_array(tmpDoc["_id"], docs); !ok {
						fmt.Printf("Document %s not in docs list. Skipping doc.\n", tmpDoc["_id"])
						continue
					}
				}
			}
		} else {
			fmt.Println("doc.json not found in folder ", doc)
			continue
		}

		//Check for design documents structure:
		//rewrites, validate_doc_update, filters, shows, updates, lists, views, fulltext

		//rewrites may be written directly inside doc.json
		if is_file, _ := FileExists(filepath.Join(pwd, DBName, doc, "rewrites.js")); is_file {
			if Verbose {
				fmt.Printf("Found rewrites.js file in %s\n", doc)
			}

			tmpDoc["rewrites"], err = loadFile(filepath.Join(pwd, DBName, doc, "rewrites.js"))
			if err != nil {
				fmt.Print(err)
			}
		}

		//validate_doc_update may be written directly inside doc.json
		if is_file, _ := FileExists(filepath.Join(pwd, DBName, doc, "validate_doc_update.js")); is_file {
			if Verbose {
				fmt.Printf("Found validate_doc_update.js file in %s\n", doc)
			}

			tmpDoc["validate_doc_update"], err = loadFile(filepath.Join(pwd, DBName, doc, "validate_doc_update.js"))
			if err != nil {
				fmt.Print(err)
			}
		}

		//filters may be written directly inside doc.json
		if is_file, _ := FileExists(filepath.Join(pwd, DBName, doc, "filters")); is_file {
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
		if is_file, _ := FileExists(filepath.Join(pwd, DBName, doc, "shows")); is_file {
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
		if is_file, _ := FileExists(filepath.Join(pwd, DBName, doc, "updates")); is_file {
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
		if is_file, _ := FileExists(filepath.Join(pwd, DBName, doc, "lists")); is_file {
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
		if is_file, _ := FileExists(filepath.Join(pwd, DBName, doc, "updates")); is_file {
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
		if is_file, _ := FileExists(filepath.Join(pwd, DBName, doc, "views")); is_file {
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
				if err != nil {
					fmt.Print(err)
				}
			}
			tmpDoc["views"] = tmpViews
		}

		//Check for attachemnts folder
		if is_file, _ := FileExists(filepath.Join(pwd, DBName, doc, "_attachments")); is_file {
			if Verbose {
				fmt.Printf("Found _attachments folder in %s\n", doc)
			}
			//inside _attachment folder there are files and folders
			//each file will be saved via a multipart/related document
			//the root path starts in _attachment subfolder
			SaveMPRDoc(tmpDoc, filepath.Join(pwd, DBName, doc, "_attachments"))
		} else {
			//Upsert to CouchDB
			rev, err = upsert(tmpDoc)
			if err != nil {
				fmt.Printf("%s", err.Error())
			}
			fmt.Printf("new document saved with rev: %s\n", rev)
		}

	}

}
