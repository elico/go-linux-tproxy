package main

import "log"

// Debug ---
var Debug = false

func debugerr(err error) {
	if Debug {
		log.Println(err)
	}
}
