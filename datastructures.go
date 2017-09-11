//
// @author: Dragos STOICA
// @date: 05.09.2017
//
package main

type CouchDoc map[string]interface{}

type alldocsResult struct {
	TotalRows int `json:"total_rows"`
	Offset    int
	Rows      []map[string]interface{}
}
