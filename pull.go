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

	//Get connection URL
	if DEBUG {
		fmt.Printf("URL: %s\n", ServerURL)
	}
	//Get database directory
	if DEBUG {
		fmt.Printf("Database name: %s\n", DBName)
	}

	//Check for database on CouchDB server
	serverConnection, err := couchdb.NewClient(ServerURL, nil)
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

	var ddoc []string
	//ddoc := strings.Split(DDocList, ",")

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
	//Get design documents by default
	//TODO - add DDocList processing to ddoc for specific documents

	var result alldocsResult
	err = workingDB.AllDocs(&result, couchdb.Options{
		"startkey": "_design",
		"endkey":   "_designZ",
	})

	for _, val := range result.Rows {
		ddoc = append(ddoc, val["id"].(string))
	}

	for _, one_doc := range ddoc {
		//Check if ddoc exists in the database
		etag, err := ETag(DBName, one_doc)
		if err != nil {
			fmt.Println("Document " + one_doc + " not found in database")
			continue
		}
		fmt.Println("Document " + one_doc + " found with rev:" + etag)
		//Get the ddoc and 'serialize' it to local directory and files
		dir_path := strings.Replace(one_doc, "_design/", "", -1)
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
					//TODO - The extension should be determined by "language" attribute
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
				//TODO - The extension should be determined by "language" attribute
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
			//TODO - The extension should be determined by "language" attribute
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
			//TODO - The extension should be determined by "language" attribute
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
			//Create updates folder
			is_dir, _ := FileExists(filepath.Join(pwd, DBName, dir_path, "updates"))
			if is_dir {
				os.Remove(filepath.Join(pwd, DBName, dir_path, "updates"))
			}
			os.MkdirAll(filepath.Join(pwd, DBName, dir_path, "updates"), os.ModePerm)

			for updates_fct, _ := range tmpdoc["updates"].(map[string]interface{}) {
				//Create [update_name].js file
				updates_file_path := filepath.Join(pwd, DBName, dir_path, "updates", updates_fct+".js")
				is_dir, err := FileExists(updates_file_path)
				if is_dir {
					os.Remove(updates_file_path)
				}
				//os.MkdirAll(view_file_path, os.ModePerm)
				//TODO - The extension should be determined by "language" attribute
				output, err := os.Create(updates_file_path)
				if err != nil {
					fmt.Println("Error while creating", updates_file_path, "-", err)
				}
				defer output.Close()
				n, err := output.WriteString(tmpdoc["updates"].(map[string]interface{})[updates_fct].(string))
				if err != nil {
					fmt.Print("Error while writing ", updates_fct, " function for updates -", err)
				}
				fmt.Printf("%s %v bytes written\n", updates_fct, n)

			}
			delete(tmpdoc, "updates")
		}

		//Handle lists
		if tmpdoc["lists"] != nil {
			//Create fulltext folder
			is_dir, _ := FileExists(filepath.Join(pwd, DBName, dir_path, "lists"))
			if is_dir {
				os.Remove(filepath.Join(pwd, DBName, dir_path, "lists"))
			}
			os.MkdirAll(filepath.Join(pwd, DBName, dir_path, "lists"), os.ModePerm)

			for lists_fct, _ := range tmpdoc["lists"].(map[string]interface{}) {
				//Create [update_name].js file
				lists_file_path := filepath.Join(pwd, DBName, dir_path, "lists", lists_fct+".js")
				is_dir, err := FileExists(lists_file_path)
				if is_dir {
					os.Remove(lists_file_path)
				}
				//os.MkdirAll(view_file_path, os.ModePerm)
				//TODO - The extension should be determined by "language" attribute
				output, err := os.Create(lists_file_path)
				if err != nil {
					fmt.Println("Error while creating", lists_file_path, "-", err)
				}
				defer output.Close()
				n, err := output.WriteString(tmpdoc["lists"].(map[string]interface{})[lists_fct].(string))
				if err != nil {
					fmt.Print("Error while writing ", lists_fct, " function for lists -", err)
				}
				fmt.Printf("%s %v bytes written\n", lists_fct, n)

			}
			delete(tmpdoc, "lists")
		}

		//Handle shows
		if tmpdoc["shows"] != nil {
			//Create shows folder
			is_dir, _ := FileExists(filepath.Join(pwd, DBName, dir_path, "shows"))
			if is_dir {
				os.Remove(filepath.Join(pwd, DBName, dir_path, "shows"))
			}
			os.MkdirAll(filepath.Join(pwd, DBName, dir_path, "shows"), os.ModePerm)

			for shows_fct, _ := range tmpdoc["shows"].(map[string]interface{}) {
				//Create [update_name].js file
				shows_file_path := filepath.Join(pwd, DBName, dir_path, "shows", shows_fct+".js")
				is_dir, err := FileExists(shows_file_path)
				if is_dir {
					os.Remove(shows_file_path)
				}
				//os.MkdirAll(view_file_path, os.ModePerm)
				//TODO - The extension should be determined by "language" attribute
				output, err := os.Create(shows_file_path)
				if err != nil {
					fmt.Println("Error while creating", shows_file_path, "-", err)
				}
				defer output.Close()
				n, err := output.WriteString(tmpdoc["shows"].(map[string]interface{})[shows_fct].(string))
				if err != nil {
					fmt.Print("Error while writing ", shows_fct, " function for shows -", err)
				}
				fmt.Printf("%s %v bytes written\n", shows_fct, n)

			}
			delete(tmpdoc, "shows")
		}

		//Handle filters
		if tmpdoc["filters"] != nil {
			//Create filters folder
			is_dir, _ := FileExists(filepath.Join(pwd, DBName, dir_path, "filters"))
			if is_dir {
				os.Remove(filepath.Join(pwd, DBName, dir_path, "filters"))
			}
			os.MkdirAll(filepath.Join(pwd, DBName, dir_path, "filters"), os.ModePerm)

			for filters_fct, _ := range tmpdoc["filters"].(map[string]interface{}) {
				//Create [update_name].js file
				filters_file_path := filepath.Join(pwd, DBName, dir_path, "filters", filters_fct+".js")
				is_dir, err := FileExists(filters_file_path)
				if is_dir {
					os.Remove(filters_file_path)
				}
				//os.MkdirAll(view_file_path, os.ModePerm)
				//TODO - The extension should be determined by "language" attribute
				output, err := os.Create(filters_file_path)
				if err != nil {
					fmt.Println("Error while creating", filters_file_path, "-", err)
				}
				defer output.Close()
				n, err := output.WriteString(tmpdoc["filters"].(map[string]interface{})[filters_fct].(string))
				if err != nil {
					fmt.Print("Error while writing ", filters_fct, " function for filters -", err)
				}
				fmt.Printf("%s %v bytes written\n", filters_fct, n)

			}
			delete(tmpdoc, "filters")
		}

		//The other fields will be dumped in doc.json
		//except _rev
		delete(tmpdoc, "_rev")
		SaveJsonFile(tmpdoc, filepath.Join(pwd, DBName, dir_path, "doc.json"))

	}

}
