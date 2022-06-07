package main

import (
	"crypto/tls"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
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
	debug = flag.Bool("debug", false, "wether or not to enable debug mode - static files are not served in this case")
)

const (
	USER_INFO       = uint8(0)
	CHAMPION_CHANGE = uint8(1)
)

type AuthInformation struct {
	Url            string
	Authentication string
}

type UserInformation struct {
	username string
	iconId   string
}

type ChampionChange struct {
	championId string
}

var userInformation UserInformation
var upgrader = websocket.Upgrader{}
var wsQueue chan interface{}
var subscribers = []uint8{}

func main() {
	flag.Parse()

	go LcuCommunication()

	wsQueue = make(chan interface{})

	router := gin.New()

	// if in debug mode, then serve only websocket server.
	if *debug {
		// Accept CORS websocket.
		upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	} else {
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

		subId := Subscribe()

		log.Printf("%s (sid %d) connected (%dmb)", w.RemoteAddr(), subId, GetMemoryUsage())

		// don't send user information if it is not yet defined.
		if userInformation.username != "" {
			msg, err := CreateWSMessage(USER_INFO, userInformation)
			if err != nil {
				log.Println("Failed to create message: ", err)
			} else {
				w.WriteJSON(msg)
			}
		}

		w.SetCloseHandler(func(code int, text string) error {
			log.Printf("%s (sid %d) disconnected (%dmb)", w.RemoteAddr(), subId, GetMemoryUsage())
			Unsubsribe(subId)
			return nil
		})

		go func() {
			for {
				mt, _, err := w.ReadMessage()
				// aparently when the client closes the connection,
				// the ReadMessage() call returns an error. (?)
				if err != nil {
					break
				}
				// most likely this code will never be executed.
				// maybe only if the close message is sent explicitly
				// inside the javascript code on the frontend.
				if mt == websocket.CloseMessage {
					log.Printf("%d listeners", len(subscribers))
					break
				}
			}
		}()

		for {
			message := <-wsQueue

			// if message is a uint8, then it is a subscription id.
			switch message.(type) {
			case uint8:
				// if its this goroutine's id, then break the loop.
				if message == subId {
					break
				}
			default:
				// if it is not a uint8, then it is a message
				// and we should send it to the client.
				err := w.WriteJSON(message)
				log.Println("Sending Message")

				if err != nil {
					log.Println("write:", err)
					return
				}
			}
		}
	})

	if err := router.Run("0.0.0.0:4246"); err != nil {
		log.Fatal("failed run app: ", err)
	}
}

// Start LCU communication.
// Useful information: https://hextechdocs.dev/tag/lcu/.
func LcuCommunication() {
	leaguePath := RetrieveLeaguePath()
	log.Printf("LeagueClientUX: %s", leaguePath)

	authInfo := WatchForLockfile(leaguePath)
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

	// Get current summoner
	r, err := http.NewRequest("GET", fmt.Sprintf("https://%s/lol-summoner/v1/current-summoner", authInfo.Url), nil)
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

	j, err := fastjson.ParseBytes(body)
	if err != nil {
		log.Fatalf("Failed to parse response: %v", err)
	}

	userInformation = UserInformation{
		username: string(j.Get("displayName").GetStringBytes()),
		iconId:   fmt.Sprint(j.Get("profileIconId").GetInt()),
	}

	// Emit the current summoner info to every client
	if err := EmitEvent(USER_INFO, userInformation); err != nil {
		log.Println("Failed to emit event: ", err)
	}

	// LCU Websocket
	u := url.URL{Scheme: "wss", Host: authInfo.Url, Path: "/"}
	d := websocket.Dialer{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	// basic authentication (riot:<password>) -> base64
	b := base64.StdEncoding.EncodeToString([]byte("riot:" + authInfo.Authentication))
	c, res, err := d.Dial(u.String(),
		http.Header{"Authorization": []string{fmt.Sprintf("Basic %s", b)}})

	if err != nil {
		log.Printf("Failed to dial: %v", err)
		if res != nil {
			body, err = ioutil.ReadAll(res.Body)
			log.Println(string(body))
			if err != nil {
				log.Printf("%d: Failed to read response: %v", res.StatusCode, err)
			}
		}
		return
	}

	defer c.Close()

	// Subscribe to changes on the current champion
	SubscribeToLCUEvent("OnJsonApiEvent_lol-champ-select-legacy_v1_current-champion", c)

	// Listen to the websocket
	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			log.Printf("Failed to read message: %v", err)
			continue
		}

		// The client sends an empty string when the websocket is first connected
		if string(message) == "" {
			continue
		}

		j, err := fastjson.ParseBytes(message)
		if err != nil {
			log.Printf("Failed to parse message: %v", err)
			continue
		}

		if string(j.Get("2").GetStringBytes("eventType")) == "Create" {
			EmitEvent(CHAMPION_CHANGE, ChampionChange{
				championId: fmt.Sprint((j.Get("2").GetInt("data"))),
			})
		}
	}
}

func CreateWSMessage(eventType uint8, data interface{}) (msg map[string]interface{}, err error) {
	switch eventType {
	case USER_INFO:
		return map[string]interface{}{
			"type":     eventType,
			"username": data.(UserInformation).username,
			"iconId":   data.(UserInformation).iconId,
		}, nil
	case CHAMPION_CHANGE:
		return map[string]interface{}{
			"type":       eventType,
			"championId": data.(ChampionChange).championId,
		}, nil
	default:
		return nil, errors.New("unknown event type")
	}
}

func EmitEvent(eventType uint8, data interface{}) (err error) {
	msg, err := CreateWSMessage(eventType, data)
	if err != nil {
		return err
	}
	for i := 0; i < len(subscribers); i++ {
		wsQueue <- msg
	}
	return nil
}

// assign this goroutine an unique id that will be used
// to unsubsribe it from the LCU events.
func Subscribe() (id uint8) {
	var subId uint8
	if len(subscribers) == 0 {
		subId = 0
	} else {
		subId = subscribers[len(subscribers)-1] + 1
	}
	subscribers = append(subscribers, subId)
	return subId
}

func Unsubsribe(id uint8) {
	for i := 0; i < len(subscribers); i++ {
		wsQueue <- id
	}
}

// Returns the total virtual memory consumed by the process in MB.
func GetMemoryUsage() uint64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m.Sys / 1024 / 1024
}

// Search for a LeagueClient.exe process and returns its executable path.
func RetrieveLeaguePath() string {
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

				moduleSnapshot, err := windows.CreateToolhelp32Snapshot(
					windows.TH32CS_SNAPMODULE, processEntry.ProcessID,
				)
				for err != nil {
					log.Printf("Failed to create module snapshot: %v", err)

					// Create a snapshot with the information of the process.
					moduleSnapshot, err = windows.CreateToolhelp32Snapshot(
						windows.TH32CS_SNAPMODULE, processEntry.ProcessID,
					)
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
func WatchForLockfile(leaguePath string) AuthInformation {
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
		Url:            fmt.Sprintf("127.0.0.1:%s", splitedLockfile[2]),
		Authentication: splitedLockfile[3],
	}

	return authInfo
}

func SubscribeToLCUEvent(eventName string, c *websocket.Conn) {
	// https://hextechdocs.dev/getting-started-with-the-lcu-websocket#subscribing-to-events
	err := c.WriteJSON([]interface{}{5, eventName})
	if err != nil {
		log.Printf("Failed to subscribe to event: %v", err)
	}
}
