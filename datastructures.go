//
// @author: Dragos STOICA
// @date: 05.09.2017
//
package main

type CouchDoc map[string]interface{}

/*
attachment_relative_path : {
	file_name,
	mime_type,
	length
}
*/
type DocList []map[string]struct {
	File_Name    string
	Content_Type string
	Byte_Length  int64
}
