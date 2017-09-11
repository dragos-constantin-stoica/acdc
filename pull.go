package main

import (
	"encoding/json"
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
	   docs [array_of_ddocs], comma separated no spaces
	----------------------------------------------------------------------- */

	docs := strings.Split(DocsList, ",")

	//Check local directory for pull destination

	if is_dir, _ := FileExists(filepath.Join(pwd, DBName)); is_dir {
		fmt.Println("Found database folder on local disk!")
	} else {
		//create DB folder of local directory
		if os.Mkdir(filepath.Join(pwd, DBName), os.ModePerm) != nil {
			fmt.Println("Can not create directory " + DBName)
		} else {
			fmt.Println("Local directory for database created!")
		}
	}
	//Get design documents by default

	var result alldocsResult

	if len(DocsList) == 0 {
		docs = []string{}
		//By default pull all _desgis docs
		err := workingDB.AllDocs(&result, couchdb.Options{
			"startkey": "_design",
			"endkey":   "_designZ",
		})
		if err != nil {
			fmt.Print(err)
		}
	} else {
		//Pull only the docs specified, it may be any docs
		docsArr, _ := json.Marshal(docs)
		err := workingDB.AllDocs(&result, couchdb.Options{
			"keys": string(docsArr),
		})
		if err != nil {
			fmt.Print(err)
		}
	}
	if DEBUG {
		fmt.Println(result)
	}
	for _, val := range result.Rows {
		if val["id"] != nil {
			docs = append(docs, val["id"].(string))
		}
		if val["error"] != nil {
			fmt.Printf("%s - %s\n", val["key"], val["error"])
		}
	}

	for _, one_doc := range docs {
		//Check if doc exists in the database
		etag, err := ETag(DBName, one_doc)
		if err != nil {
			fmt.Println("Document " + one_doc + " not found in database")
			continue
		}
		fmt.Println("Document " + one_doc + " found with rev:" + etag)
		//Get the ddoc and 'serialize' it to local directory and files
		dir_path := strings.Replace(strings.Replace(one_doc, "_local/", "", -1), "_design/", "", -1)
		if is_dir, _ := FileExists(filepath.Join(pwd, DBName, dir_path)); !is_dir {
			if os.MkdirAll(filepath.Join(pwd, DBName, dir_path), os.ModePerm) != nil {
				fmt.Println("Can not create subdirectory " + dir_path)
			} else {
				fmt.Println("Local directory for ddoc created!")
			}
		}
		//Get ddoc from DB
		var tmpdoc CouchDoc //interface{}
		if workingDB.Get(one_doc, &tmpdoc, nil) != nil {
			fmt.Println("Error fethcing document form DB!")
		}

		//Handle _attachments
		if tmpdoc["_attachments"] != nil {
			//fmt.Print("attachments >> ")
			//fmt.Println(tmpdoc["_attachments"])
			for att_doc, _ := range tmpdoc["_attachments"].(map[string]interface{}) {
				//fmt.Println(att_doc)
				att_doc_relpath, _ := filepath.Split(att_doc)
				att_doc_path := filepath.Join(pwd, DBName, dir_path, "_attachments",
					strings.Join(strings.Split(att_doc, "/"), string(os.PathSeparator)))
				is_file, err := FileExists(att_doc_path)
				if is_file {
					os.Remove(att_doc_path)
				}
				os.MkdirAll(filepath.Join(pwd, DBName, dir_path, "_attachments", att_doc_relpath), os.ModePerm)
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
			delete(tmpdoc, "_attachments")
		}

		//Handle views
		if tmpdoc["views"] != nil {
			//fmt.Println(tmpdoc["views"])
			//Create views folder
			is_dir, _ := FileExists(filepath.Join(pwd, DBName, dir_path, "views"))
			if is_dir {
				os.Remove(filepath.Join(pwd, DBName, dir_path, "views"))
			}
			os.MkdirAll(filepath.Join(pwd, DBName, dir_path, "views"), os.ModePerm)

			for view_fct, _ := range tmpdoc["views"].(map[string]interface{}) {
				for view_type, _ := range tmpdoc["views"].(map[string]interface{})[view_fct].(map[string]interface{}) {
					//Create [view_name].[view_type].js file
					view_file_path := filepath.Join(pwd, DBName, dir_path, "views", view_fct+"."+view_type+".js")
					is_dir, err := FileExists(view_file_path)
					if is_dir {
						os.Remove(view_file_path)
					}
					//os.MkdirAll(view_file_path, os.ModePerm)
					output, err := os.Create(view_file_path)
					if err != nil {
						fmt.Println("Error while creating", view_file_path, "-", err)
					}
					defer output.Close()
					n, err := output.WriteString(tmpdoc["views"].(map[string]interface{})[view_fct].(map[string]interface{})[view_type].(string))
					if err != nil {
						fmt.Print("Error while writing ", view_fct, " function for ", view_type, "-", err)
					}
					fmt.Printf("%s %s - %v bytes written\n", view_fct, view_type, n)
				}

			}
			delete(tmpdoc, "views")
		}

		//Handle fulltext
		if tmpdoc["fulltext"] != nil {
			//Create fulltext folder
			is_dir, _ := FileExists(filepath.Join(pwd, DBName, dir_path, "fulltext"))
			if is_dir {
				os.Remove(filepath.Join(pwd, DBName, dir_path, "fulltext"))
			}
			os.MkdirAll(filepath.Join(pwd, DBName, dir_path, "fulltext"), os.ModePerm)

			for fulltext_fct, _ := range tmpdoc["fulltext"].(map[string]interface{}) {
				//Create [index_name].js file
				fulltext_file_path := filepath.Join(pwd, DBName, dir_path, "fulltext", fulltext_fct+".js")
				is_dir, err := FileExists(fulltext_file_path)
				if is_dir {
					os.Remove(fulltext_file_path)
				}
				//os.MkdirAll(view_file_path, os.ModePerm)
				output, err := os.Create(fulltext_file_path)
				if err != nil {
					fmt.Println("Error while creating", fulltext_file_path, "-", err)
				}
				defer output.Close()
				n, err := output.WriteString(tmpdoc["fulltext"].(map[string]interface{})[fulltext_fct].(map[string]interface{})["index"].(string))
				if err != nil {
					fmt.Print("Error while writing ", fulltext_fct, " function for index -", err)
				}
				fmt.Printf("%s - index %v bytes written\n", fulltext_fct, n)

			}
			delete(tmpdoc, "fulltext")
		}

		//Handle rewrites
		if tmpdoc["rewrites"] != nil {
			//Create rewrites file in the document root

			rewrites_file_path := filepath.Join(pwd, DBName, dir_path, "rewrites.js")
			is_dir, err := FileExists(rewrites_file_path)
			if is_dir {
				os.Remove(rewrites_file_path)
			}
			//os.MkdirAll(view_file_path, os.ModePerm)
			output, err := os.Create(rewrites_file_path)
			if err != nil {
				fmt.Println("Error while creating", rewrites_file_path, "-", err)
			}
			defer output.Close()
			n, err := output.WriteString(tmpdoc["rewrites"].(string))
			if err != nil {
				fmt.Print("Error while writing rewrites.js -", err)
			}
			fmt.Printf("rewrites %v bytes written\n", n)

			delete(tmpdoc, "rewrites")
		}

		//Handle validate_doc_update
		if tmpdoc["validate_doc_update"] != nil {
			//Create validate_doc_update file in the document root

			validate_doc_update_file_path := filepath.Join(pwd, DBName, dir_path, "validate_doc_update.js")
			is_dir, err := FileExists(validate_doc_update_file_path)
			if is_dir {
				os.Remove(validate_doc_update_file_path)
			}
			//os.MkdirAll(view_file_path, os.ModePerm)
			output, err := os.Create(validate_doc_update_file_path)
			if err != nil {
				fmt.Println("Error while creating", validate_doc_update_file_path, "-", err)
			}
			defer output.Close()
			n, err := output.WriteString(tmpdoc["validate_doc_update"].(string))
			if err != nil {
				fmt.Print("Error while writing validate_doc_update.js -", err)
			}
			fmt.Printf("validate_doc_update %v bytes written\n", n)

			delete(tmpdoc, "validate_doc_update")
		}

		//Handle updates
		if tmpdoc["updates"] != nil {
			tmpdoc = attribute2file(tmpdoc, "updates", dir_path)
		}

		//Handle lists
		if tmpdoc["lists"] != nil {
			tmpdoc = attribute2file(tmpdoc, "lists", dir_path)
		}

		//Handle shows
		if tmpdoc["shows"] != nil {
			tmpdoc = attribute2file(tmpdoc, "shows", dir_path)
		}

		//Handle filters
		if tmpdoc["filters"] != nil {
			tmpdoc = attribute2file(tmpdoc, "filters", dir_path)
		}

		//The other fields will be dumped in doc.json
		//except _rev
		delete(tmpdoc, "_rev")
		SaveJsonFile(tmpdoc, filepath.Join(pwd, DBName, dir_path, "doc.json"))

	}

}
