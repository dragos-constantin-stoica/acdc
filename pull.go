package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/fjl/go-couchdb"
)

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
	is_dir, err := FileExists(filepath.Join(pwd, DBName))
	if is_dir {
		fmt.Println("Found database folder on local disk!")
	} else {
		//create DB folder of local directory
		if os.Mkdir(filepath.Join(pwd, DBName), os.ModePerm) != nil {
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
		is_dir, err := FileExists(filepath.Join(pwd, DBName, one_doc))
		if !is_dir {
			if os.MkdirAll(filepath.Join(pwd, DBName, strings.Replace(one_doc, "_design/", "", -1)), os.ModePerm) != nil {
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
				att_doc_path := filepath.Join(pwd, DBName, strings.Replace(one_doc, "_design/", "", -1), "_attachments",
					strings.Join(strings.Split(att_doc, "/"), string(os.PathSeparator)))
				is_file, err := FileExists(att_doc_path)
				if is_file {
					os.Remove(att_doc_path)
				}
				os.MkdirAll(filepath.Join(pwd, DBName, strings.Replace(one_doc, "_design/", "", -1), "_attachments", att_doc_relpath), os.ModePerm)
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
				view_fct_path := filepath.Join(pwd, DBName, strings.Replace(one_doc, "_design/", "", -1), "views", view_fct)
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
