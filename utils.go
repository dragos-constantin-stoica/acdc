package main

import (
	"errors"
	"io/ioutil"
	"net/http"
	"os"
)

//Check if a file exists - full path as argument
func FileExists(file_name string) (bool, error) {
	_, err := os.Stat(file_name)
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

//List files in a directory
func ListDir(dir_name string) ([]string, error) {
	var result []string

	files, err := ioutil.ReadDir(dir_name)
	if err != nil {
		return []string{""}, err
	}

	for _, file := range files {
		if file.IsDir() {
			result = append(result, file.Name())
		}
	}

	return result, err
}

//Read ETag of a HEAD response from CouchDB
func ETag(db, doc string) (string, error) {
	var result string

	//Get current version of the document via HEAD in ETag
	resp, err := http.Head(ServerURL + db + "/" + doc)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.Header["Etag"] != nil {
		result = resp.Header["Etag"][0]
	} else {
		err = errors.New("No ETag in response!")
		return "", err
	}

	return result, nil
}
