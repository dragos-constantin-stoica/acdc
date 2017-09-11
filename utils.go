package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/Jeffail/gabs"
)

//Check is a values is present in an array
func in_array(val interface{}, array interface{}) (exists bool, index int) {
	exists = false
	index = -1

	switch reflect.TypeOf(array).Kind() {
	case reflect.Slice:
		s := reflect.ValueOf(array)

		for i := 0; i < s.Len(); i++ {
			if reflect.DeepEqual(val, s.Index(i).Interface()) == true {
				index = i
				exists = true
				return
			}
		}
	}

	return
}

//Check if a file exists - full path as argument
func FileExists(file_name string) (bool, error) {
	if _, err := os.Stat(file_name); os.IsNotExist(err) {
		return false, err
	}
	return true, nil
}

//List subdirectories in a directory
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

//Build recursively a JSON part of multipart/realted document
//based on the structure of _attachments folder
func BuildJSONforAttachments(folder string, DDoc *gabs.Container) {
	if DEBUG {
		fmt.Printf("Current path to explore %s\n", folder)
	}
	//For each file in the current folder create an object in the JSON structure
	relativePath := splitPath(folder)
	file_list, err := ListFile(folder)
	if err != nil {
		fmt.Print(err)
	}
	if DEBUG {
		fmt.Printf("Attachment files found %s\n", file_list)
	}

	for _, tmpFile := range file_list {

		fileinfo, err := os.Stat(path.Join(folder, tmpFile))
		if err != nil {
			fmt.Print(err)
			continue
		}
		DDoc.Set(true, "_attachments", relativePath+tmpFile, "follows")
		DDoc.Set(mime.TypeByExtension(path.Ext(tmpFile)), "_attachments", relativePath+tmpFile, "content_type")
		//Get attachement length in bytes

		// get the size
		DDoc.Set(fileinfo.Size(), "_attachments", relativePath+tmpFile, "length")
	}

	//For each subfolder repeat the procedure
	folder_list, err := ListDir(folder)
	if err != nil {
		fmt.Print(err)
	}
	if len(folder_list) == 0 {
		return
	}
	if DEBUG {
		fmt.Printf("Subfolders found %s\n", folder_list)
	}
	for _, tmpFolder := range folder_list {
		BuildJSONforAttachments(path.Join(folder, tmpFolder), DDoc)
	}

}

//List files only in a folder
func ListFile(dir_name string) ([]string, error) {
	var result []string

	files, err := ioutil.ReadDir(dir_name)
	if err != nil {
		return []string{""}, err
	}

	for _, file := range files {
		if !file.IsDir() {
			result = append(result, file.Name())
		}
	}

	return result, err
}

// splitPath returns relative path for _attachments
func splitPath(full_path string) string {
	if i := strings.LastIndex(full_path, "_attachments/"); i != -1 {
		return full_path[(i+len("_attachments/")):] + "/"
	}
	return ""
}

// stripExtension returns the given filename without its extension.
func stripExtension(filename string) string {
	if i := strings.LastIndex(filename, "."); i != -1 {
		return filename[:i]
	}
	return filename
}

// filename contains view_name.view_type.extension
// the function will return view_name and view_type
func splitFileNameforView(filename string) (string, string) {
	tmpFileName := filename
	if i := strings.LastIndex(filename, "."); i != -1 {
		tmpFileName = filename[:i]
	}

	if i := strings.LastIndex(tmpFileName, "."); i != -1 {
		return tmpFileName[:i], tmpFileName[(i + 1):]
	}
	return "", ""
}

// loadJSON decodes the content of the given file as JSON.
func loadJSON(file string) (CouchDoc, error) {
	content, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}
	// TODO: use json.Number
	var val CouchDoc
	if err := json.Unmarshal(content, &val); err != nil {
		if syntaxerr, ok := err.(*json.SyntaxError); ok {
			line := findLine(content, syntaxerr.Offset)
			err = fmt.Errorf("JSON syntax error at %v:%v: %v", file, line, err)
			return nil, err
		}
		return nil, fmt.Errorf("JSON unmarshal error in %v: %v", file, err)
	}
	return val, nil
}

// findLine returns the line number for the given offset into data.
func findLine(data []byte, offset int64) (line int) {
	line = 1
	for i, r := range string(data) {
		if int64(i) >= offset {
			return
		}
		if r == '\n' {
			line++
		}
	}
	return
}

//load full file as a string
func loadFile(file string) (string, error) {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return "", err
	}
	return string(bytes.Trim(data, " \n\r")), nil
}

//Save any structure to a file in JSON format
func SaveJsonFile(v interface{}, path string) {
	fo, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	defer fo.Close()
	e := json.NewEncoder(fo)
	if err := e.Encode(v); err != nil {
		panic(err)
	}
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

//Process pull attribute
func attribute2file(tmpdoc CouchDoc, attr_name string, dir_path string) CouchDoc {

	//Create the folder
	is_dir, _ := FileExists(filepath.Join(pwd, DBName, dir_path, attr_name))
	if is_dir {
		os.Remove(filepath.Join(pwd, DBName, dir_path, attr_name))
	}
	os.MkdirAll(filepath.Join(pwd, DBName, dir_path, attr_name), os.ModePerm)

	for fct, _ := range tmpdoc[attr_name].(map[string]interface{}) {
		//Create [update_name].js file
		attr_file_path := filepath.Join(pwd, DBName, dir_path, attr_name, fct+".js")
		is_dir, err := FileExists(attr_file_path)
		if is_dir {
			os.Remove(attr_file_path)
		}

		output, err := os.Create(attr_file_path)
		if err != nil {
			fmt.Println("Error while creating", attr_file_path, "-", err)
		}
		defer output.Close()
		n, err := output.WriteString(tmpdoc[attr_name].(map[string]interface{})[fct].(string))
		if err != nil {
			fmt.Print("Error while writing ", fct, " function for ", attr_name, " -", err)
		}
		fmt.Printf("%s %v bytes written\n", fct, n)

	}
	delete(tmpdoc, attr_name)
	return tmpdoc

}

//Transform CouchDBDoc structure to JSON
func (doc *CouchDoc) toJSON() string {
	o := &bytes.Buffer{}
	enc := json.NewEncoder(o)
	if err := enc.Encode(&doc); err != nil {
		fmt.Println(err)
	}
	return o.String()
}
