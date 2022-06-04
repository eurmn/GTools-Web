package main

import (
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
	"unsafe"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/valyala/fastjson"
	"golang.org/x/sys/windows"
)

var (
	lcuOnly = flag.Bool("lcu-only", false, "wether or not to disable file serving and only serve LCU communication")
)

const (
	USER_INFO  = uint8(0)
	LCU_UPDATE = uint8(1)
)

type AuthInformation struct {
	Url            string
	Authentication string
}

type UserInformation struct {
	username string
	iconId   string
}

var userInformation UserInformation
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}
var wsQueue chan interface{}

func main() {
	flag.Parse()

	go LcuCommunication()

	wsQueue = make(chan interface{}, 1)

	router := gin.New()

	// if lcu-only, serve only websocket server.
	if !(*lcuOnly) {
		router.GET("/", func(c *gin.Context) {
			c.Redirect(http.StatusMovedPermanently, "public/")
		})
		router.StaticFS("/public/", http.Dir("./public/"))
	}

	// lcu websocket endpoint
	router.GET("/lcu", func(c *gin.Context) {
		w, err := upgrader.Upgrade(c.Writer, c.Request, nil)

		if err != nil {
			log.Println("upgrade:", err)
			return
		}

		defer w.Close()
		log.Printf("%s connected", w.RemoteAddr())

		// don't send user information if it is not yet defined.
		if userInformation.username != "" {
			msg, err := CreateWSMessage(USER_INFO, userInformation)
			if err != nil {
				log.Println("Failed to create message: ", err)
			} else {
				w.WriteJSON(msg)
			}
		}

		for {
			message := <-wsQueue
			err := w.WriteJSON(message)
			if err != nil {
				log.Println("write:", err)
				return
			}
		}
	})

	if err := router.Run(":4246"); err != nil {
		log.Fatal("failed run app: ", err)
	}
}

func LcuCommunication() {
	// Start LCU communication.
	// Useful information: https://hextechdocs.dev/tag/lcu/.
	leaguePath := retrieveLeaguePath()
	log.Printf("LeagueClientUX: %s", leaguePath)

	authInfo := watchForLockfile(leaguePath)
	log.Printf("%s", authInfo.Authentication)

	/// LCU uses a self-signed certificate, so we need to disable TLS verification.
	// https://developer.riotgames.com/docs/lol#game-client-api_root-certificate-ssl-errors
	lcuClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	r, err := http.NewRequest("GET", fmt.Sprintf("%s/lol-summoner/v1/current-summoner", authInfo.Url), nil)
	if err != nil {
		log.Fatalf("Failed to create request: %v", err)
	}
	r.SetBasicAuth("riot", authInfo.Authentication)

	res, err := lcuClient.Do(r)
	if err != nil {
		log.Fatalf("Failed to send request: %v", err)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatalf("Failed to read response: %v", err)
	}

	j, err := fastjson.Parse(string(body))
	if err != nil {
		log.Fatalf("Failed to parse response: %v", err)
	}

	userInformation = UserInformation{
		username: string(j.Get("displayName").GetStringBytes()),
		iconId:   fmt.Sprint(j.Get("profileIconId").GetInt()),
	}

	msg, err := CreateWSMessage(USER_INFO, userInformation)
	if err != nil {
		log.Println("Failed to create message: ", err)
	} else {
		for range wsQueue {
			wsQueue <- msg
		}
	}
}

func CreateWSMessage(eventType uint8, data interface{}) (map[string]interface{}, error) {
	switch eventType {
	case USER_INFO:
		return map[string]interface{}{
			"type":     eventType,
			"username": data.(UserInformation).username,
			"iconId":   data.(UserInformation).iconId,
		}, nil
	default:
		return nil, errors.New("unknown event type")
	}
}

func GetMemoryUsage() uint64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m.Sys / 1024 / 1024
}

// Search for a LeagueClient.exe process and returns its executable path.
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

// Search on the specified path for the lockfile and return its relevant part
// - API URL and auth code.
func watchForLockfile(leaguePath string) AuthInformation {
	lockfilePath := fmt.Sprintf("%s\\lockfile", leaguePath)

	_, err := os.Stat(lockfilePath)

	for errors.Is(err, os.ErrNotExist) {
		_, err = os.Stat(lockfilePath)
		time.Sleep(time.Second * 2) // Try again in 2 seconds.
	}

	content, err := ioutil.ReadFile(lockfilePath)

	if err != nil {
		log.Fatalf("Failed to read lockfile: %v", err)
	}

	// Extract content from lockfile
	splitedLockfile := strings.Split(string(content), ":")

	authInfo := AuthInformation{
		Url:            fmt.Sprintf("https://127.0.0.1:%s", splitedLockfile[2]),
		Authentication: splitedLockfile[3],
	}

	return authInfo
}
