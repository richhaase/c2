package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/richhaase/c2cli/internal/cmd"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	var err error
	switch os.Args[1] {
	case "auth":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "Usage: c2cli auth <token>")
			os.Exit(1)
		}
		err = cmd.RunAuth(os.Args[2])
	case "sync":
		err = cmd.RunSync()
	case "log":
		n := 10
		if len(os.Args) >= 3 {
			if v, e := strconv.Atoi(os.Args[2]); e == nil {
				n = v
			}
		}
		err = cmd.RunLog(n)
	case "status":
		err = cmd.RunStatus()
	case "help", "--help", "-h":
		usage()
	case "version", "--version", "-v":
		fmt.Println("c2cli v0.1.0")
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", os.Args[1])
		usage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Println(`c2cli — Concept2 Logbook CLI

Usage:
  c2cli auth <token>    Save access token and verify
  c2cli sync            Pull new workouts from the API
  c2cli log [n]         Show last N workouts (default: 10)
  c2cli status          Show progress toward million-meter goal
  c2cli version         Show version`)
}
