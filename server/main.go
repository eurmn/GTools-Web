package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"
	"unsafe"

	"github.com/valyala/fasthttp"
	"golang.org/x/sys/windows"
)

var fsHandler fasthttp.RequestHandler

func main() {
	port := 4246
	addr := fmt.Sprintf("0.0.0.0:%d", port)

	fs := &fasthttp.FS{
		Root:       "./public",
		IndexNames: []string{"index.html"},
		Compress:   true,
	}
	fsHandler = fs.NewRequestHandler()

	log.Printf("Starting HTTP server on http://%s", addr)
	go func() {
		if err := fasthttp.ListenAndServe(addr, requestHandler); err != nil {
			log.Fatalf("error in ListenAndServe: %v", err)
		}
	}()

	// Start LCU communication.
	// Useful information: https://hextechdocs.dev/tag/lcu/.
	leaguePath := retrieveLeaguePath()
	log.Printf("%s", leaguePath)

	watchForLockfile(leaguePath)

	// Wait forever.
	select {}
}

func requestHandler(ctx *fasthttp.RequestCtx) {
	switch string(ctx.Path()) {
	default:
		fsHandler(ctx)
	}
}

// Search for a LeagueClientUx.exe process and returns its executable path.
func retrieveLeaguePath() string {
	var leaguePath string

	for leaguePath == "" {
		// https://docs.microsoft.com/en-us/windows/win32/toolhelp/taking-a-snapshot-and-viewing-processes.
		// Retrieve a snapshot of all processes.
		snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)

		if err != nil {
			log.Fatalf("failed to create snapshot: %v", err)
		}

		var processEntry windows.ProcessEntry32
		processEntry.Size = uint32(unsafe.Sizeof(processEntry))

		windows.Process32First(snapshot, &processEntry)

		for {
			if windows.UTF16ToString(processEntry.ExeFile[:]) == "LeagueClient.exe" {
				log.Printf("Found LeagueClient process with PID %d", processEntry.ProcessID)

				var moduleEntry windows.ModuleEntry32
				moduleEntry.Size = uint32(unsafe.Sizeof(moduleEntry))

				// Create a snapshot with the information of the process.
				moduleSnapshot, err := windows.CreateToolhelp32Snapshot(
					windows.TH32CS_SNAPMODULE, processEntry.ProcessID,
				)

				if err != nil {
					log.Fatalf("Failed to create module snapshot: %v", err)
				}

				windows.Module32First(moduleSnapshot, &moduleEntry)
				leaguePath = filepath.Dir(windows.UTF16ToString(moduleEntry.ExePath[:]))
				break
			}

			// Will throw ERROR_NO_MORE_FILES when reached end of list.
			if err := windows.Process32Next(snapshot, &processEntry); err != nil {
				break
			}
		}
		time.Sleep(time.Second * 2) // Try again in 2 seconds.
	}

	return leaguePath
}

func watchForLockfile(leaguePath string) {
	lockfilePath := fmt.Sprintf("%s\\lockfile", leaguePath)

	fmt.Println(lockfilePath)

	_, err := os.Stat(lockfilePath)

	for errors.Is(err, os.ErrNotExist) {
		_, err = os.Stat(lockfilePath)
		time.Sleep(time.Second * 2) // Try again in 2 seconds.
	}

	content, err := ioutil.ReadFile(lockfilePath)

	if err != nil {
		log.Fatalf("Failed to read lockfile: %v", err)
	}

	fmt.Println(string(content))
}
