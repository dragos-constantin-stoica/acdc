#!/bin/bash

#---------------------------------------------------------#
# This script is used to build from command line the acdc
# Clean the build and leave only the sources
# Push to git
# And help message
#
# @author Dragos STOICA - 01.09.2017
#---------------------------------------------------------#


#
# Auxiliary function of how to use this tool
#
usage()
{
	echo "Script or build, clean and git sync of ACDC"
	echo "Usage example:"
	echo -e "\n$0 --build\tbuild the entire project"
	echo -e "\n$0 --clear\tclean all files and prepare for push"
	echo -e "\n$0 --git-push\tgit push"
	echo -e "\n$0 --git-pull\tgit pull"
	echo -e "\n$0 --help\tthis message."
	echo "Chill out and enjoy life!"
}


#
# Check if at leas 2 arguments were passed
#
if [ $# -lt 1 ]; then
	usage
	exit 0
fi

#
# Check command line arguments and exit in the absence of 
# mandatory arguments
#
while [ "$1" != "" ]; do
    PARAM=`echo $1 | awk -F= '{print $1}'`
    VALUE=`echo $1 | awk -F= '{print $2}'`
    case $PARAM in
        --help)
            usage
            exit
            ;;
        --build)
			# build all
			export GOPATH=`pwd`
			export GOBIN=$GOPATH/bin
			
			#Create the specific GO project folders
			mkdir -p src
			mkdir -p bin
			mkdir -p pkg
			
			
			go get github.com/Jeffail/gabs
			go get github.com/fjl/go-couchdb
			go get github.com/philippfranke/multipart-related/related
			# go install
			go build -a
			exit
			;;
        --clear)
			# clean option is present
            rm -fr bin pkg src acdc
			exit
            ;;
        --git-push)
            git push
			exit
            ;;
        --git-pull)
            git pull
			exit
            ;;
        *)
		    usage
            echo "ERROR: unknown parameter \"$PARAM\""
            exit 1
            ;;
    esac
    shift
done

