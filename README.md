# AC:zap:DC
**A**nother application **C**onnector for couchapp **D**evelopers with **C**ouchDB

I fully agree this is not English but the acronym sounds cool ... **AC**:zap:**DC**  
It allows the developers to pull couchapp-lications from CouchDB to local folders, to edit them,
and to push from local folders back to CouchDB. There is a great tool couchapp and its successor erica
that are dealing with couchapp format. The main difference is that **AC**:zap:**DC** pushes
the attachments in a multi-part related document rateher than file by file this will create a single version of the design document.

A couchapp has the following properties:
- it is a WEB application writent in JavaScript
- it may be a SPA application with all attachments to a single design document. By convention `_design/app`
- it is served directly from CouchDB
- it may use any framework (JQuery, DHTMLX, Webix, Framework7, W2UI, EasyUI, Bootstrap etc) for UI and data manipulation
- it exchanges data between client's browser and CouchDB via AJAX or it can use PouchDB for offline activity
- it may contain fields related to CouchDB design documents (views, lists etc)

## Examples of usage

```
./acdc -pull -db test_db -URL http://localhost:5984/

./acdc -push -db test_db -URL http://192.168.0.69:5984/
```

#### Command line parameters

```
Usage of ./acdc:
  -URL URL
    	specify full URL to access CouchDB server (default "http://root:root@localhost:5984/")
  -db database
    	database name to be pulled
  -ddoc design documents
    	 list of design documents IDs to be pulled, list is comma separated
  -project directory
    	full path to project directory where databases are stored (default "/home/dragos/acdc")
  -pull
    	sync documents from CouchDB to local directory
  -push
    	sync local directory with CouchDB documents
  -verbose
    	verbose mode
```

## AC:zap:DC folder mappings


The application receive as input parameter the name of a local folder, having a specific structure. The appicaton will try to parse them as CouchDB `Database->Document` possible candidates.
The main folder structure is:

```	
[DATABASE] -|
	    |_ [DOCUMENT]
	    |_ [DOCUMENT]
```

The top folder is considered to be the database. Each subfolder is considered to be a document.
There are 3 types of CouchDB documents:  

1. Normal
1. Design
1. Local

### 0) Local documents

They are not replicated/synchronized and they are not subject to map/reduce
in views. In order to use a local document one must address it directly.
Local documents do no have versioning - an inplace update is applied.
Special field:  

```json
{	
	"_id" : "_local/[DOCUMENT_NAME]"
}
```

The folder mapping will be:

```
[DATABASE]
	|_ [DOCUMENT_NAME]
		|_ local_doc.json
```


### Normal documents

They are used to store most of the data and are subject to versioning and to
map/reduce mechanism of views. Those documents are replicated. Special fields:

```json
{
	"_id" : "[STRING]",
	"_rev": "#-[DOCUMENT MD5]",
	"_attachments" : { ... "file_attachments" ...}
}
```

The folder mapping will be:

```
[DATABASE]
	|_ [DOCUMENT_NAME]
		|_ doc.json
		|_ attachments
			|_ ... files and foldres
```


### Design documents

They are used to store mainly processing scripts: views, lists, shows etc and to
store couchapps: web applications that are directly served from CouchDB. Special
fields:

```json
{
	"_id" : "_design/[DOCUMENT_NAME]",
	"_rev" : "#-[DOCUMENT MD5]",
	"_attachments" : {"file_attachments"},
	"views" : {
		"view_name":{
			"map":"",
			"reduce":""
		},
		"lib":{"name":""}
	},
	"lists" : { "list_name":""},
	"shows" : { "show_name":""},
	"filters" : { "filter_name":"" },
	"updates" : { "update_name":""},
	"rewrites" : [ {} ],
	"validate_doc_update" : "",
	"fulltext" : {
		"index_name": {
			"index" : ""
		}
	}
}
```

The folder structure of a design document will be:

```
[DATABASE]
	|_ [DOCUMENT_NAME]
		|_ design_document.json
		|_ attachments
			|_ ... files and folders
		|_ views
			|_ [VIEW_NAME]_map.js
			|_ [VIEW_NAME]_reduce.js
			|_ [LIB_NAME].js
		|_ lists
			|_ [LIST_NAME].js
		|_ shows
			|_ [SHOW_NAME].js
		|_ updates
			|_ [UPDATE_NAME].js
		|_ filters
			|_ [FILTER_NAME].js
		|_ full_text
			|_ [INDEX_NAME].js
		|_ rewrites.js
		|_ validate_doc_update.js
```

## How does it work?

The push mechanism will parse recursivelly all subfolders of the database folder, and will be considered as documents. For each folder representing a document
a multipart-related document will be created and pushed to CouchDB. The files `.json` will be read and pushed as such.

The pull mechanism consist in inspecting a single given database for design documents
and to download on local drive in a given folder structure the attachments and special
fields as source code files. Pull function will be able to get only a specified document.

The goal is to create a seemles IDE integration that will allow developers to be proficient with couchapp
development and implementation.

Some interesting ideas here: http://blog.couchbase.com/2015/october/bulk-operations-using-couchbase-and-golang
