# AC :zap: DC

**A**nother application **C**onnector for couchapp **D**evelopers with **C**ouchDB

I fully agree this is not English but the acronym sounds cool ... **AC**:zap:**DC**  
It allows the developers to pull couchapplications from CouchDB to local folders, to edit them,
and to push from local folders back to CouchDB. There is a great tool called [couchapp](https://github.com/couchapp/couchapp) and its successor [erica](https://github.com/benoitc/erica)
that are dealing with couchapp format. The main difference is that **AC**:zap:**DC** simplifies a lot the structure on disk of a couchapp and pushes
the attachments in a multipart related document rateher than file by file this will create a single version of the document.

A couchapp has the following properties:

- it is a WEB application writent in JavaScript
- it may be a SPA application with all attachments to a single design document. By convention `_design/app`
- it is served directly from CouchDB
- it may use any framework (JQuery, DHTMLX, Webix, Framework7, Polymer, Bootstrap etc) for UI and data manipulation
- it exchanges data between client's browser and CouchDB via AJAX or it can use PouchDB for offline activity
- it may contain fields related to CouchDB design documents (views, lists etc)

## How to clone and compile

You need to have pre-installed on your machine:

- [CouchDB](http://couchdb.apache.org/) on your local machine or use a virtualized container like: Docker, Vaagrant, etc
- [Go lang](https://golang.org/) verions 1.9 installed on your machine
- Git tool in order to clone git repository

Open a terminal and start typing:

```
git clone https://github.com/iqcouch/acdc
cd acdc
./build.sh
acdc -help
```

If you see the following message:

```
push or pull subcommand is required. help command for more details.
```

it means that you mangaged to clone and compile **AC**:zap:**DC** successfully. Congratulations!!!

## Examples of usage

```
./acdc pull -db test_db -URL http://localhost:5984/

./acdc push -db test_db -URL http://localhost:5984/
```

#### Command line parameters

```
Usage
	acdc {pull | push | help} parameters... 

 acdc pull -db <database> -URL <CouchDB_URL> [-docs <id,...>] [-v]
  -URL string
    	Full URL to access CouchDB server in the format PROTOCOL://USER:PASSWORD@SERVER:PORT/. (Required) (default "http://localhost:5984/")
  -db string
    	Database name to be pulled from. (Required)
  -docs string
    	List of comma separated document IDs to be pulled.
  -verbose
    	Verbose mode.

 acdc push -db <database> -URL <CouchDB_URL> [-docs <id,...>] [-v]
  -URL string
    	Full URL to access CouchDB server in the format PROTOCOL://USER:PASSWORD@SERVER:PORT/. (Required) (default "http://localhost:5984/")
  -db string
    	Database name to be pushed to. (Required)
  -docs string
    	List of comma separated document IDs to be pushed.
  -verbose
    	Verbose mode.

 acdc help
	Print this message.
```

## AC :zap: DC folder mappings

The application receive as input parameter the name of a local folder, having a specific structure. The appicaton will try to parse them as CouchDB structure based on the assumption that folder structure maps to CouchDB structure as follows: `Database->Document`, where the main folder is the database name the subfolders are documents.

A good example of such a folder to database coduments mapping is in **test_db** subfolder. So, please have a looke there and let us know if you have any questions.

The main folder structure is:

```	
[DATABASE] -|
	    |_ [DOCUMENT]
	    |_ [DOCUMENT]
```

There are 3 types of CouchDB documents:  

1. Local
1. Normal
1. Design

### Local documents

They are not replicated/synchronized and they are not subject to map/reduce
in views. The `_id` of the document must start with **_local/**. In order to use a local document one must address it directly.
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
	|_ [DOCUMENT_FOLDER]
		|_ doc.json
```

The `doc.json` file must contain at least the `_id` attribute.

### Normal documents

They are used to store most of the data and are subject to versioning and to
map/reduce mechanism of views. Those documents are replicated. Special fields:

```json
{
	"_id" : "[STRING]",
	"_attachments" : { ... "file_attachments" ...}
}
```

The folder mapping will be:

```
[DATABASE]
	|_ [DOCUMENT_FOLDER]
		|_ doc.json
		|_ attachments
			|_ ... files and foldres
```
The `doc.json` file must contain at least the `_id` attribute.

### Design documents

They are used to store mainly processing scripts: views, lists, shows etc and to
store couchapps: web applications stored as attachments and are directly served from CouchDB. Special
fields:

```json
{
	"_id" : "_design/[DOCUMENT_NAME]",
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
	|_ [DOCUMENT_FOLDER]
		|_ doc.json
		|_ attachments
			|_ ... files and folders
		|_ views
			|_ [VIEW_NAME].map.js
			|_ [VIEW_NAME].reduce.js
			|_ [LIB_NAME].[FUNCTION_NAME].js
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

The `doc.json` file must contain at least the `_id` attribute.

## How does it work?

The push mechanism will parse recursivelly all subfolders of the database folder, seraching for `doc.json` file and then for specific files and subfolders. For each folder representing a document
a multipart-related document will be created and pushed to CouchDB, if `_attachments` folder is present and contains files and subfolders. The file `doc.json` will be read and pushed as such.

The pull mechanism consist in inspecting a single given database for design documents
and downloads them on local drive. It will create a given folder structure, copmatible with push command.

The goal is to create a seemles IDE integration that will allow developers to be proficient with couchapp development and implementation.

Some interesting ideas here: http://blog.couchbase.com/2015/october/bulk-operations-using-couchbase-and-golang

### Some use cases

#### 1. Clone an exising couchapp from a remote database to local folder

You already have a couchapp running on `demo.server.com` CouchDB server in the database `mygreatapp`. You want to clone the design documents from that serve to a local folder in order to develop the application.

Make a project subfolder, let say `couchapp_project` and copy the `acdc` application there, then:

```bash
cd couchapp_project
./acdc pull -db mygreatapp -URL http://demo.server.com:5984/
```

#### 2. Push changes to remote database from local folder

Suppose that you have already done use case 1., above. After done coding on local files, then:

```bash
./acdc push -db mygreatapp -URL http://demo.server.com:5984/
```
