#!/usr/bin/env bash

set -x
set -e

PACKAGEDIR=`pwd`
export PACKAGEDIR

cd $GOPATH/src
ln -s $PACKAGEDIR

cd $GOPATH/src/`basename $PACKAGEDIR`
case $1 in

ensure)
dep ensure -v
;;

update)
dep update -v
;;

status)
dep status -v
;;

init)
dep init -v
;;

build)
mkdir -p bin
go build -o "./bin/`basename $PACKAGEDIR`-`uname -s`-`uname -m`"
;;

*)
echo "use one of the options: (ensure|update|status)"

esac

cd $GOPATH/src
rm -v $GOPATH/src/`basename $PACKAGEDIR`

set +e
set +x
