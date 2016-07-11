# acdc
Another application Connector for couchapp Developers with CouchDB

I fully agree this is not English but the acronym sounds cool ... ACDC  
It allows the developers to pull couchapplications from CouchDB to local folders, to edit them,
and to push from local folders to CouchDB. There is a great tool couchapp and its successor erica
that are dealing with couchapp format. The main difference is that ACDC pushes
the attachments in a multi-part related document rateher than file by file.

A couchapp has the following properties:
- it is a WEB application writent in JavaScript
- it may be a SPA application with all attachments to a single design document
- it is served directly from CouchDB
- it may use any framework (JQuery, DHTMLX, Webix, Framework7, W2UI, EasyUI etc) for UI
and data manipulation
- it exchanges data between client's browser and CouchDB via AJAX
- it may contain fields related to CouchDB design documents (views, lists etc)

ACDC folder mappings
----
The application will search local directories for specific folder structures
trying to interpret them as CouchDB  
``` Database->Document ``` possible candidates.
The main folder structure is:
```	
[DATABASE] -|
		    |_ [DOCUMENT]
		    |_ [DOCUMENT]
```
The top folder is considered to be the database. Each subfolder is considered
to be a document.
There are 3 types of CouchDB documents:
0. Normal
1. Design
2. Local

0) Local documents
----
They are not replicated/synchronized and they are not subject to map/reduce
in views. In order to use a local document one must address it directly.
Local documents do no have versioning - an inplace update is applied.
Special field:  
```javascript	
"_id" : "_local/[DOCUMENT_NAME]"
```

1) Normal documents
----
They are used to store most of the data and are subject to versioning and to
map/reduce mechanism of views. Those documents are replicated. Special fields:  
```javascript
"_id" : "[STRING]"
"_rev": "#-[DOCUMENT MD5]"
"_attachments" : {"file_attachments"}
```

2) Design documents
----
They are used to store mainly processing scripts: views, lists, shows etc and to
store couchapps: web applications that are directly served from CouchDB. Special
fields:
```javascript
	"_id" : "_design/[DOCUMENT_NAME]"
	"_rev" : "#-[DOCUMENT MD5]"
	"_attachments" : {"file_attachments"}
	"views" : {
		"view_name":{
			"map":"",
			"reduce":""
		},
		"lib":{"name":""}
	}
	"lists" : { "list_name":""}
	"shows" : { "show_name":""}
	"filters" : { "filter_name":"" }
	"updates" : { "update_name":""}
	"rewrites" : {}
	"validate_doc_update" : ""
	"fulltext" : {
		"index_name": {
			"index" : ""
		}
	}
```
The push mechanism will parse recursivelly all subfolders of the project folder,
considering those subfolders as database names. Subsequentrly all subfolders of database
folders will be considered as documents. For each folder representing a document
a multipart-related document will be created and pushed to CouchDB.
The pull mechanism consist in inspecting a single given database for design documents
and to download on local drive in a given folder structure the attachments and special
fields as source code files. Pull function will be able to get only a specified document.
The folder-subfolder-files format is taylored to be backwards compatible with appzip so
that the deployment may be done via appzip also. The goal is to create an seemles IDE integration,
eventualy online web based, that will allow developers to be proficient with couchapp
development and implementation.
