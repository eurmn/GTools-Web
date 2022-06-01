package main

import (
	"fmt"
	"log"
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

	var leaguePath string

	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)

	if err != nil {
		log.Fatalf("failed to create snapshot: %v", err)
	}

	var processEntry windows.ProcessEntry32
	processEntry.Size = uint32(unsafe.Sizeof(processEntry))

	windows.Process32First(snapshot, &processEntry)

	for leaguePath == "" {
		for {
			if windows.UTF16ToString(processEntry.ExeFile[:]) == "LeagueClient.exe" {
				log.Printf("Found LeagueClient process with PID %d", processEntry.ProcessID)

				var moduleEntry windows.ModuleEntry32
				moduleEntry.Size = uint32(unsafe.Sizeof(moduleEntry))
				moduleSnapshot, err := windows.CreateToolhelp32Snapshot(
					windows.TH32CS_SNAPMODULE, processEntry.ProcessID,
				)

				if err != nil {
					log.Fatalf("Failed to create module snapshot: %v", err)
				}

				windows.Module32First(moduleSnapshot, &moduleEntry)
				leaguePath = windows.UTF16ToString(moduleEntry.ExePath[:])
				break
			}

			// Will throw ERROR_NO_MORE_FILES when reached end of list.
			if err := windows.Process32Next(snapshot, &processEntry); err != nil {
				break
			}
		}
		time.Sleep(time.Second * 2) // Try again in 2 seconds.
	}

	log.Printf("%s", leaguePath)

	// Wait forever.
	select {}
}

func requestHandler(ctx *fasthttp.RequestCtx) {
	switch string(ctx.Path()) {
	default:
		fsHandler(ctx)
	}
}
