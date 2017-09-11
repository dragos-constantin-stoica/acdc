//
// @author: Dragos STOICA
// @date: 01.03.2016
//
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/fjl/go-couchdb"
)

const (
	DEBUG = false //TODO - change to true in DEV mode
)

var (
	ServerURL   string //full URL: PROTOCOL://USER:PASSWORD@SERVER:PORT/
	DBName      string //database name to push to and to pull from
	DocsList    string //desing documents list, comma separated
	Verbose     bool   //verbose flag
	OP_Function bool   //the operation to be triggered

	serverConnection *couchdb.Client
	workingDB        *couchdb.DB
	rev              string

	pwd string //working directory for relative paths

)

func main() {

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
	fmt.Printf("Connection to server %s is working.\n", ServerURL)

	workingDB, err = serverConnection.EnsureDB(DBName)
	if err != nil {
		fmt.Println(err)
		//Create the database if not exists, it might be the case for the first time push
		workingDB, _ = serverConnection.CreateDB(DBName)
		fmt.Printf("Database %s created.\n", DBName)
	} else {
		fmt.Printf("Working with %s database.\n", workingDB.Name())
	}

	if OP_Function {
		Push()
	} else {
		Pull()
	}
}

func init() {

	// Verify that a subcommand has been provided
	// os.Arg[0] is the main command
	// os.Arg[1] will be the subcommand

	if len(os.Args) <= 1 {
		fmt.Println("push or pull subcommand is required. help command for more details.")
		os.Exit(1)
	}

	//The subcommand variables
	pushCommand := flag.NewFlagSet("push", flag.ExitOnError)
	pullCommand := flag.NewFlagSet("pull", flag.ExitOnError)

	// push -db -URL [-docs] [-v]
	// pull -db -URL [-docs] [-v]
	// help

	// push subcommand flag pointers

	pushServerURL := pushCommand.String("URL", "http://localhost:5984/", "Full URL to access CouchDB server in the format PROTOCOL://USER:PASSWORD@SERVER:PORT/. (Required)")
	pushDBName := pushCommand.String("db", "", "Database name to be pushed to. (Required)")
	pushDocsList := pushCommand.String("docs", "", "List of comma separated document IDs to be pushed.")
	pushVerbose := pushCommand.Bool("verbose", false, "Verbose mode.")

	pullServerURL := pullCommand.String("URL", "http://localhost:5984/", "Full URL to access CouchDB server in the format PROTOCOL://USER:PASSWORD@SERVER:PORT/. (Required)")
	pullDBName := pullCommand.String("db", "", "Database name to be pulled from. (Required)")
	pullDocsList := pullCommand.String("docs", "", "List of comma separated document IDs to be pulled.")
	pullVerbose := pullCommand.Bool("verbose", false, "Verbose mode.")

	switch os.Args[1] {
	case "push":
		OP_Function = true
		pushCommand.Parse(os.Args[2:])
	case "pull":
		OP_Function = false
		pullCommand.Parse(os.Args[2:])
	case "help":
		fmt.Println("Usage\n\tacdc {pull | push | help} parameters... \n\n acdc pull -db <database> -URL <CouchDB_URL> [-docs <id,...>] [-v]")
		pullCommand.PrintDefaults()
		fmt.Println("\n acdc push -db <database> -URL <CouchDB_URL> [-docs <id,...>] [-v]")
		pushCommand.PrintDefaults()
		fmt.Println("\n acdc help\n\tPrint this message.")

		os.Exit(0)
	default:
		fmt.Println("push or pull subcommand is required. help command for more details.")
		os.Exit(1)
	}

	// Check which subcommand was Parsed using the FlagSet.Parsed() function. Handle each case accordingly.
	// FlagSet.Parse() will evaluate to false if no flags were parsed (i.e. the user did not provide any flags)
	if pushCommand.Parsed() {
		// Required Flags
		if (*pushServerURL == "") || (*pushDBName == "") {
			fmt.Println("acdc push -db <database> -URL <CouchDB_URL> [-docs <id,...>] [-v]")
			pushCommand.PrintDefaults()
			os.Exit(1)
		}

		ServerURL = *pushServerURL
		DBName = *pushDBName
		DocsList = *pushDocsList
		Verbose = *pushVerbose
	}

	if pullCommand.Parsed() {
		// Required Flags
		if (*pullServerURL == "") || (*pullDBName == "") {
			fmt.Println("acdc pull -db <database> -URL <CouchDB_URL> [-docs <id,...>] [-v]")
			pullCommand.PrintDefaults()
			os.Exit(1)
		}

		ServerURL = *pullServerURL
		DBName = *pullDBName
		DocsList = *pullDocsList
		Verbose = *pullVerbose
	}

	pwd, _ = os.Getwd()

	if DEBUG {
		Verbose = true // For DEBUG purposes
	}
}
