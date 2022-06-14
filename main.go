package main

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/microcosm-cc/bluemonday"
	"github.com/valyala/fastjson"
	"golang.org/x/sys/windows"
)

var (
	debug = flag.Bool("debug", false, "wether or not to enable debug mode - static files are not served in this case")
)

const (
	USER_INFO       = uint8(0)
	CHAMPION_CHANGE = uint8(1)
	CDRAGON         = "https://raw.communitydragon.org/latest"
)

type AuthInformation struct {
	Url            string
	Authentication string
}

type UserInformation struct {
	Username   string
	IconId     string
	SummonerId string
}

type ChampionChange struct {
	ChampionId        string
	ChampionName      string
	RunesByPopularity []Rune
	RunesByWinRate    []Rune
	Role              string
}

type RuneInfo struct {
	Name        string
	Description string
}

type Rune struct {
	Id    uint16
	Asset string
	Info  RuneInfo
}

var userInformation UserInformation
var upgrader = websocket.Upgrader{}
var wsQueue chan interface{}
var subscribers = []uint8{}
var championNames = map[uint16]string{}
var runeInfo = map[uint16]Rune{}
var authInfo AuthInformation
var p *bluemonday.Policy = bluemonday.StripTagsPolicy()

func main() {
	flag.Parse()

	go func() {
		for {
			LcuCommunication()
			// wait 10s to try a new connection with the LCU.
			time.Sleep(time.Second * 10)
		}
	}()

	var err error
	championNames, err = GetUpdatedChampionNames()
	if err != nil {
		log.Fatalf("Failed to get updated champion names: %v", err)
	}

	runeInfo, err = GetUpdatedRuneAssets()
	if err != nil {
		log.Fatalf("Failed to get updated rune asset paths: %v", err)
	}

	wsQueue = make(chan interface{})

	if !*debug {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(cors.Default())

	// if in debug mode, then serve only websocket server.
	if *debug {
		// Accept CORS websocket.
		upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	} else {
		router.GET("/", func(c *gin.Context) {
			c.Redirect(http.StatusMovedPermanently, "public/")
		})
		router.StaticFS("/public/", http.Dir("./web/dist/"))

		// TODO: set log to log-file at %appdata% or %localappdata%
	}

	router.GET("/sample-build", func(c *gin.Context) {
		c.JSON(http.StatusOK, map[string]interface{}{
			"id":   136,
			"name": "Aurelion Sol",
			"role": "Mid",
			"runes": []map[string]interface{}{
				{
					"Id":    8100,
					"Asset": "https://raw.communitydragon.org/latest/plugins/rcp-be-lol-game-data/global/default/v1/perk-images/styles/7200_domination.png",
					"Info": map[string]string{
						"Name":        "Domination",
						"Description": "",
					},
				},
				{
					"Id":    8300,
					"Asset": "https://raw.communitydragon.org/latest/plugins/rcp-be-lol-game-data/global/default/v1/perk-images/styles/7203_whimsy.png",
					"Info": map[string]string{
						"Name":        "Inspiration",
						"Description": "",
					},
				},
				{
					"Id":    8112,
					"Asset": "https://raw.communitydragon.org/latest/plugins/rcp-be-lol-game-data/global/default/v1/perk-images/styles/domination/electrocute/electrocute.png",
					"Info": map[string]string{
						"Name":        "Eletrocute",
						"Description": "Hitting a champion with 3 separate attacks or abilities in 3s deals bonus adaptive damage.",
					},
				},
				{
					"Id":    8139,
					"Asset": "https://raw.communitydragon.org/latest/plugins/rcp-be-lol-game-data/global/default/v1/perk-images/styles/domination/tasteofblood/greenterror_tasteofblood.png",
					"Info": map[string]string{
						"Name":        "Taste of Blood",
						"Description": "Heal when you damage an enemy champion.",
					},
				},
				{
					"Id":    8138,
					"Asset": "https://raw.communitydragon.org/latest/plugins/rcp-be-lol-game-data/global/default/v1/perk-images/styles/domination/eyeballcollection/eyeballcollection.png",
					"Info": map[string]string{
						"Name":        "Eyeball Collection",
						"Description": "Collect eyeballs for champion takedowns. Gain permanent AD or AP, adaptive for each eyeball plus bonus upon collection completion.",
					},
				},
				{
					"Id":    8105,
					"Asset": "https://raw.communitydragon.org/latest/plugins/rcp-be-lol-game-data/global/default/v1/perk-images/styles/domination/relentlesshunter/relentlesshunter.png",
					"Info": map[string]string{
						"Name":        "Relentless Hunter",
						"Description": "Unique takedowns grant permanent out of combat MS. ",
					},
				},
				{
					"Id":    8345,
					"Asset": "https://raw.communitydragon.org/latest/plugins/rcp-be-lol-game-data/global/default/v1/perk-images/styles/inspiration/biscuitdelivery/biscuitdelivery.png",
					"Info": map[string]string{
						"Name":        "Biscuit Delivery",
						"Description": "Gain a free Biscuit every 2 min, until 6 min. Consuming or selling a Biscuit permanently increases your max mana and restores health and mana.",
					},
				},
				{
					"Id":    8352,
					"Asset": "https://raw.communitydragon.org/latest/plugins/rcp-be-lol-game-data/global/default/v1/perk-images/styles/inspiration/timewarptonic/timewarptonic.png",
					"Info": map[string]string{
						"Name":        "Time Warp Tonic",
						"Description": "Potions and biscuits grant some restoration immediately. Gain MS  while under their effects.",
					},
				},
				{
					"Id":    5005,
					"Asset": "https://raw.communitydragon.org/latest/plugins/rcp-be-lol-game-data/global/default/v1/perk-images/statmods/statmodsattackspeedicon.png",
					"Info": map[string]string{
						"Name":        "",
						"Description": "",
					},
				},
				{
					"Id":    5008,
					"Asset": "https://raw.communitydragon.org/latest/plugins/rcp-be-lol-game-data/global/default/v1/perk-images/statmods/statmodsadaptiveforceicon.png",
					"Info": map[string]string{
						"Name":        "",
						"Description": "",
					},
				},
				{
					"Id":    5003,
					"Asset": "https://raw.communitydragon.org/latest/plugins/rcp-be-lol-game-data/global/default/v1/perk-images/statmods/statmodsmagicresicon.magicresist_fix.png",
					"Info": map[string]string{
						"Name":        "",
						"Description": "",
					},
				},
			},
		})
	})

	router.POST("/import-runes", func(c *gin.Context) {
		defer c.Request.Body.Close()

		// Read request body.
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			log.Printf("Failed to read request body: %v", err)
			c.Status(http.StatusBadRequest)
			return
		}

		bodyJson, err := fastjson.ParseBytes(body)
		if err != nil {
			log.Printf("Failed to parse request body: %v", err)
			c.Status(http.StatusBadRequest)
			return
		}

		runesJson := bodyJson.GetArray("runes")
		championIdString := string(bodyJson.GetStringBytes("champion_id"))
		role := string(bodyJson.GetStringBytes("role"))

		championId, err := strconv.ParseUint(championIdString, 10, 16)
		if err != nil {
			log.Printf("Failed to parse champion id: %v", err)
			c.Status(http.StatusBadRequest)
			return
		}

		runes := []uint16{}
		for _, run := range runesJson {
			runes = append(runes, uint16(run.GetInt()))
		}

		runeLCU := RuneArrayToObject(runes, uint16(championId), role)

		// LCU uses a self-signed certificate, so we need to disable TLS verification.
		// https://developer.riotgames.com/docs/lol#game-client-api_root-certificate-ssl-errors
		lcuClient := &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		}

		r, err := http.NewRequest(http.MethodGet, "https://"+authInfo.Url+"/lol-perks/v1/pages", nil)
		if err != nil {
			log.Printf("Failed to create request: %v", err)
			c.Status(http.StatusInternalServerError)
			return
		}
		r.SetBasicAuth("riot", authInfo.Authentication)

		res, err := lcuClient.Do(r)
		if err != nil {
			log.Printf("Failed to send request: %v", err)
			c.Status(http.StatusInternalServerError)
			return
		}

		defer res.Body.Close()
		body, err = io.ReadAll(res.Body)
		if err != nil {
			log.Printf("Failed to read response body: %v", err)
			c.Status(http.StatusInternalServerError)
			return
		}

		bodyJson, err = fastjson.ParseBytes(body)
		if err != nil {
			log.Printf("Failed to parse response body: %v", err)
			c.Status(http.StatusInternalServerError)
			return
		}

		currentRuneId := -1
		for _, page := range bodyJson.GetArray() {
			if page.GetBool("isDeletable") {
				currentRuneId = page.GetInt("id")
				break
			}
		}

		if currentRuneId != -1 {
			r, err = http.NewRequest(http.MethodDelete, fmt.Sprintf("https://%s/lol-perks/v1/pages/%d", authInfo.Url, currentRuneId), nil)
			r.SetBasicAuth("riot", authInfo.Authentication)
			if err != nil {
				log.Printf("Failed to create request: %v", err)
				c.Status(http.StatusInternalServerError)
				return
			}

			_, err = lcuClient.Do(r)
			if err != nil {
				log.Printf("Failed to send request: %v", err)
				c.Status(http.StatusInternalServerError)
				return
			}
		}

		runeBytes := runeLCU.MarshalTo(nil)

		r, err = http.NewRequest(http.MethodPost, "https://"+authInfo.Url+"/lol-perks/v1/pages", bytes.NewBuffer(runeBytes))
		r.SetBasicAuth("riot", authInfo.Authentication)

		if err != nil {
			log.Printf("Failed to create request: %v", err)
			c.Status(http.StatusInternalServerError)
			return
		}

		_, err = lcuClient.Do(r)
		if err != nil {
			log.Printf("Failed to send request: %v", err)
			c.Status(http.StatusInternalServerError)
			return
		}

		c.Status(http.StatusOK)
	})

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
		if userInformation.Username != "" {
			msg, err := CreateWSMessage(USER_INFO, userInformation)
			if err != nil {
				log.Println("Failed to create message: ", err)
			} else {
				w.WriteJSON(msg)
			}
		}

		w.SetCloseHandler(func(code int, text string) error {
			log.Printf("%s (sid %d) disconnected (%dmb)", w.RemoteAddr(), subId, GetMemoryUsage())
			Unsubscribe(subId)
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
				// if it's this goroutine's id, then break the loop.
				if message == subId {
					return
				}
			default:
				// if it is not a uint8, then it is a message
				// and we should send it to the client.
				err := w.WriteJSON(message)
				log.Printf("(sid %d) Sending Message", subId)

				if err != nil {
					log.Println("write:", err)
					Unsubscribe(subId)
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

	authInfo = WatchForLockfile(leaguePath)
	log.Printf("%s", authInfo.Authentication)

	// LCU uses a self-signed certificate, so we need to disable TLS verification.
	// https://developer.riotgames.com/docs/lol#game-client-api_root-certificate-ssl-errors
	lcuClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	// Get current summoner
	r, err := http.NewRequest(http.MethodGet, "https://"+authInfo.Url+"/lol-summoner/v1/current-summoner", nil)
	for err != nil {
		r, err = http.NewRequest(http.MethodGet, "https://"+authInfo.Url+"/lol-summoner/v1/current-summoner", nil)
		log.Printf("Failed to create request: %v", err)
	}
	r.SetBasicAuth("riot", authInfo.Authentication)

	res, err := lcuClient.Do(r)
	for err != nil {
		time.Sleep(time.Second * 2)
		res, err = lcuClient.Do(r)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Fatalf("Failed to read response: %v", err)
	}

	// you are not logged in.
	if res.StatusCode != 404 {
		j, err := fastjson.ParseBytes(body)
		if err != nil {
			log.Fatalf("Failed to parse response: %v", err)
		}

		userInformation = UserInformation{
			Username:   string(j.GetStringBytes("displayName")),
			IconId:     fmt.Sprint(j.GetInt("profileIconId")),
			SummonerId: fmt.Sprint(j.GetInt("summonerId")),
		}

		// Emit the current summoner info to every client
		if err := EmitEvent(USER_INFO, userInformation); err != nil {
			log.Println("Failed to emit event: ", err)
		}
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
		http.Header{"Authorization": []string{"Basic " + b}})

	defer res.Body.Close()

	if err != nil {
		log.Printf("Failed to dial: %v", err)
		if res != nil {
			body, err := io.ReadAll(res.Body)
			log.Println(string(body))
			if err != nil {
				log.Printf("%d: Failed to read response: %v", res.StatusCode, err)
			}
		}
		return
	}

	defer c.Close()

	// Subscribe to changes on the current champion/summoner
	SubscribeToLCUEvent("OnJsonApiEvent_lol-champ-select_v1_session", c)
	SubscribeToLCUEvent("OnJsonApiEvent_lol-summoner_v1_current-summoner", c)

	lastChampionId := uint16(0)
	// Listen to the websocket
	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			log.Printf("Failed to read message: %v", err)
			return
		}

		if mt == websocket.CloseMessage {
			log.Println("Websocket closed")
			return
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

		switch string(j.GetStringBytes("1")) {
		case "OnJsonApiEvent_lol-champ-select_v1_session":
			if string(j.Get("2").GetStringBytes("eventType")) != "Delete" {
				data := j.Get("2").Get("data")

				cid := uint16(0)
				role := ""
				for _, summoner := range data.GetArray("myTeam") {
					if fmt.Sprint(summoner.GetInt("summonerId")) == userInformation.SummonerId {
						switch string(summoner.GetStringBytes("assignedPosition")) {
						case "bottom":
							role = "ADC"
						case "support":
							role = "SUPPORT"
						case "jungle":
							role = "JUNGLE"
						case "top":
							role = "TOP"
						case "middle":
							role = "MID"
						}
						cid = uint16(summoner.GetInt("championId"))
						if cid == 0 {
							cid = uint16(summoner.GetInt("championPickIntent"))
						}
						break
					}
				}

				if cid == 0 || lastChampionId == cid {
					break
				}

				lastChampionId = cid
				queue := "RANKED_SOLO_5X5"

				if role == "" {
					// Get current queue.
					r, err := http.NewRequest(http.MethodGet, "https://"+authInfo.Url+"/lol-lobby/v1/parties/gamemode", nil)
					if err != nil {
						log.Printf("Failed to create request: %v", err)
						break
					}
					r.SetBasicAuth("riot", authInfo.Authentication)

					resp, err := lcuClient.Do(r)
					if err != nil {
						log.Printf("Failed to send request: %v", err)
						break
					}

					defer resp.Body.Close()
					body, err := io.ReadAll(resp.Body)
					if err != nil {
						log.Printf("Failed to read response: %v", err)
						break
					}

					content, err := fastjson.ParseBytes(body)
					if err != nil {
						log.Printf("Failed to parse response: %v", err)
						break
					}

					queueId := content.GetInt("queueId")
					// 430 = blink pick; 450 = aram.
					if queueId == 450 || queueId == 900 {
						queue = "HOWLING_ABYSS_ARAM"
					} else {
						role, err = GetPrimaryRoleForChampion(cid)
						if err != nil {
							log.Printf("Failed to get primary role: %v", err)
							continue
						}
					}
				}

				// Get runes by popularity
				runesPop, err := GetRunesForChampion(cid, role, queue, false)
				if err != nil {
					log.Printf("Failed to get runes: %v", err)
					continue
				}

				// Get runes by winRate
				runesWr, err := GetRunesForChampion(cid, role, queue, true)
				if err != nil {
					log.Printf("Failed to get runes: %v", err)
					continue
				}

				assetsPop := []Rune{}
				for _, run := range runesPop {
					assetsPop = append(assetsPop, runeInfo[run])
				}

				assetsWr := []Rune{}
				for _, run := range runesWr {
					assetsWr = append(assetsWr, runeInfo[run])
				}

				EmitEvent(CHAMPION_CHANGE, ChampionChange{
					ChampionId:        fmt.Sprint(cid),
					ChampionName:      championNames[cid],
					RunesByPopularity: assetsPop,
					RunesByWinRate:    assetsWr,
					Role:              role,
				})
			}
		case "OnJsonApiEvent_lol-summoner_v1_current-summoner":
			if string(j.Get("2").GetStringBytes("uri")) != "/lol-summoner/v1/current-summoner" {
				continue
			}

			data := j.Get("2").Get("data")

			if string(data.GetStringBytes("eventType")) == "Delete" {
				userInformation = UserInformation{
					Username:   "",
					IconId:     "",
					SummonerId: "",
				}
			} else {
				userInformation = UserInformation{
					Username:   string(j.Get("2").Get("data").GetStringBytes("displayName")),
					IconId:     fmt.Sprint(j.Get("2").Get("data").GetInt("profileIconId")),
					SummonerId: fmt.Sprint(j.Get("2").Get("data").GetInt("summonerId")),
				}
			}

			// Emit the current summoner info to every client
			if err := EmitEvent(USER_INFO, userInformation); err != nil {
				log.Println("Failed to emit event: ", err)
			}
		}
	}
}

// Build the WS Message used by EmitEvent.
func CreateWSMessage(eventType uint8, data interface{}) (msg map[string]interface{}, err error) {
	switch eventType {
	case USER_INFO:
		return map[string]interface{}{
			"type":     eventType,
			"username": data.(UserInformation).Username,
			"iconId":   data.(UserInformation).IconId,
		}, nil
	case CHAMPION_CHANGE:
		return map[string]interface{}{
			"type":              eventType,
			"id":                data.(ChampionChange).ChampionId,
			"name":              data.(ChampionChange).ChampionName,
			"role":              data.(ChampionChange).Role,
			"runesByPopularity": data.(ChampionChange).RunesByPopularity,
			"runesByWinRate":    data.(ChampionChange).RunesByWinRate,
		}, nil
	default:
		return nil, errors.New("unknown event type")
	}
}

// Emit event to all connected clients.
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
// to unsubscribe it from the wsQueue.
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

// Unsubscribe the goroutine id from the wsQueue.
func Unsubscribe(id uint8) {
	for i := 0; i < len(subscribers); i++ {
		if subscribers[i] == id {
			subscribers = append(subscribers[:i], subscribers[i+1:]...)
		}
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
				if err != nil {
					log.Printf("Failed to create module snapshot: %v", err)
					continue
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
	lockfilePath := leaguePath + "\\lockfile"

	_, err := os.Stat(lockfilePath)

	for errors.Is(err, os.ErrNotExist) {
		_, err = os.Stat(lockfilePath)
		time.Sleep(time.Second * 2) // Try again in 2 seconds.
	}

	content, err := os.ReadFile(lockfilePath)

	if err != nil {
		log.Fatalf("Failed to read lockfile: %v", err)
	}

	// Extract content from lockfile
	splitedLockfile := strings.Split(string(content), ":")

	authInfo := AuthInformation{
		Url:            "127.0.0.1:" + splitedLockfile[2],
		Authentication: splitedLockfile[3],
	}

	return authInfo
}

// Subscribe to the LCU event.
func SubscribeToLCUEvent(eventName string, c *websocket.Conn) {
	// https://hextechdocs.dev/getting-started-with-the-lcu-websocket#subscribing-to-events
	err := c.WriteJSON([]interface{}{5, eventName})
	if err != nil {
		log.Printf("Failed to subscribe to event: %v", err)
	}
}

// Update the map of champion id -> champion name.
func GetUpdatedChampionNames() (championNames map[uint16]string, err error) {
	res, err := http.Get(CDRAGON + "/plugins/rcp-be-lol-game-data/global/default/v1/champion-summary.json")
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	content, err := fastjson.ParseBytes(body)
	if err != nil {
		return nil, err
	}

	champions, err := content.Array()
	if err != nil {
		return nil, err
	}

	var localChampionNames = map[uint16]string{}
	for _, champion := range champions[1:] { // skip first champion (id of -1)
		championName := champion.Get("name").GetStringBytes()
		championId := uint16(champion.Get("id").GetUint())
		localChampionNames[championId] = string(championName)
	}

	return localChampionNames, nil
}

// Get the icon path for every rune.
func GetUpdatedRuneAssets() (runeAssetPaths map[uint16]Rune, err error) {
	res, err := http.Get(CDRAGON + "/plugins/rcp-be-lol-game-data/global/default/v1/perkstyles.json")
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	content, err := fastjson.ParseBytes(body)
	if err != nil {
		return nil, err
	}

	runes := map[uint16]Rune{}
	for _, run := range content.GetArray("styles") {
		runeId := run.Get("id").GetUint()
		runePath := string(run.GetStringBytes("iconPath"))
		runePath = CDRAGON +
			strings.ToLower(
				strings.Replace(string(runePath), "/lol-game-data/assets/v1/", "/plugins/rcp-be-lol-game-data/global/default/v1/", 1),
			)
		runeName := run.GetStringBytes("name")

		runes[uint16(runeId)] = Rune{
			Id:    uint16(runeId),
			Asset: string(runePath),
			Info: RuneInfo{
				Name:        string(runeName),
				Description: "",
			},
		}
	}

	res, err = http.Get(CDRAGON + "/plugins/rcp-be-lol-game-data/global/default/v1/perks.json")
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()
	body, err = io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	content, err = fastjson.ParseBytes(body)
	if err != nil {
		return nil, err
	}

	for _, run := range content.GetArray() {
		runeId := run.Get("id").GetUint()
		runePath := string(run.GetStringBytes("iconPath"))
		runePath = CDRAGON +
			strings.ToLower(
				strings.Replace(string(runePath), "/lol-game-data/assets/v1/", "/plugins/rcp-be-lol-game-data/global/default/v1/", 1),
			)
		runeName := run.GetStringBytes("name")
		runeDescription := p.Sanitize(string(run.GetStringBytes("shortDesc")))
		runeDescription = html.UnescapeString(runeDescription)

		runes[uint16(runeId)] = Rune{
			Id:    uint16(runeId),
			Asset: string(runePath),
			Info: RuneInfo{
				Name:        string(runeName),
				Description: runeDescription,
			},
		}
	}

	return runes, nil
}

// Get the URL (from champion.gg/blitz.gg) used to get info about the champion.
func BlitzGGUrlForChampion(championId uint16, role string, queue string) string {
	// these urls were extracted from the network tab at champion.gg, it was probably
	// not made for public use, so it can die at any moment (making the code break D:).
	if queue == "HOWLING_ABYSS_ARAM" {
		return "https://league-champion-aggregate.iesdev.com/graphql?query=query%20" +
			url.QueryEscape(
				"ChampionBuilds($championId:Int!, $queue:Queue!, $role:Role, $opponentChampionId:Int, $key:ChampionBuildKey) {"+
					"championBuildStats("+
					"championId:$championId, queue:$queue, role:$role, opponentChampionId:$opponentChampionId, key:$key) {"+
					"championId opponentChampionId queue role builds { "+
					"completedItems {"+
					"games index averageIndex itemId wins"+
					"} games mythicId mythicAverageIndex primaryRune runes {"+
					"games index runeId wins treeId"+
					"} skillOrders {"+
					"games skillOrder wins"+
					"} startingItems {"+
					"games startingItemIds wins"+
					"} summonerSpells {"+
					"games summonerSpellIds wins"+
					"} wins}}}",
			) + "&variables=" +
			url.QueryEscape(fmt.Sprintf(
				`{"championId": %d, "queue": "%s", "opponentChampionId": null, "key": "PUBLIC"}`,
				championId, queue,
			))
	}
	return "https://league-champion-aggregate.iesdev.com/graphql?query=query%20" +
		url.QueryEscape(
			"ChampionBuilds($championId:Int!, $queue:Queue!, $role:Role, $opponentChampionId:Int, $key:ChampionBuildKey) {"+
				"championBuildStats("+
				"championId:$championId, queue:$queue, role:$role, opponentChampionId:$opponentChampionId, key:$key) {"+
				"championId opponentChampionId queue role builds { "+
				"completedItems {"+
				"games index averageIndex itemId wins"+
				"} games mythicId mythicAverageIndex primaryRune runes {"+
				"games index runeId wins treeId"+
				"} skillOrders {"+
				"games skillOrder wins"+
				"} startingItems {"+
				"games startingItemIds wins"+
				"} summonerSpells {"+
				"games summonerSpellIds wins"+
				"} wins}}}",
		) + "&variables=" +
		url.QueryEscape(fmt.Sprintf(
			`{"championId": %d, "role": "%s", "queue": "%s", "opponentChampionId": null, "key": "PUBLIC"}`,
			championId, role, queue,
		))
}

// Get primary role for champion (MID | TOP | SUPPORT | JUNGLE | ADC)
func GetPrimaryRoleForChampion(championId uint16) (role string, err error) {
	url := "https://league-champion-aggregate.iesdev.com/graphql?query=query%20" +
		url.QueryEscape("ChampionMainRole($championId:ID){primaryRole(championId:$championId)}") +
		"&variables=" + url.QueryEscape(fmt.Sprintf(`{"championId": %d}`, championId))

	res, err := http.Get(url)
	if err != nil {
		return "", err
	}

	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	content, err := fastjson.ParseBytes(body)
	if err != nil {
		return "", err
	}

	return string(content.Get("data").GetStringBytes("primaryRole")), nil
}

// Return the runes for the selected champion.
func GetRunesForChampion(championId uint16, role string, queue string, winRate bool) (runes []uint16, err error) {
	url := BlitzGGUrlForChampion(championId, role, queue)
	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	content, err := fastjson.ParseBytes(body)
	if err != nil {
		return nil, err
	}

	builds := content.Get("data").Get("championBuildStats").GetArray("builds")
	sort.SliceStable(builds, func(i, j int) bool {
		if winRate {
			// sort by winRate
			return builds[i].GetInt("wins")/builds[i].GetInt("games") > builds[j].GetInt("wins")/builds[j].GetInt("games")
		} else {
			// sort by popularity
			return builds[i].GetInt("games") > builds[j].GetInt("games")
		}
	})

	selectedBuild := builds[0]
	selectedRunes := selectedBuild.GetArray("runes")

	var primaryStyleId uint16
	var secondaryStyleId uint16

	finalRunes := []uint16{}
	for i := 0; i < 8; i++ {
		currentIndex := []*fastjson.Value{}
		for _, run := range selectedRunes {
			if run.GetInt("index") == i {
				currentIndex = append(currentIndex, run)
			}
		}

		sort.SliceStable(currentIndex, func(i, j int) bool {
			if winRate {
				// sort by win-rate:
				return currentIndex[i].GetInt("wins")/currentIndex[i].GetInt("games") > currentIndex[j].GetInt("wins")/currentIndex[j].GetInt("games")
			} else {
				// sort by popularity
				return currentIndex[i].GetInt("games") > currentIndex[j].GetInt("games")
			}
		})

		if i == 0 {
			primaryStyleId = uint16(currentIndex[0].GetInt("treeId"))
		} else if i == 3 {
			secondaryStyleId = uint16(currentIndex[0].GetInt("treeId"))
		}

		finalRunes = append(finalRunes, uint16(currentIndex[0].GetInt("runeId")))
	}

	primaryRune := uint16(selectedBuild.GetInt("primaryRune"))

	// append primary and secondary style runes at THE BEGGINING of finalRunes
	finalRunes = append([]uint16{primaryStyleId, secondaryStyleId, primaryRune}, finalRunes...)

	return finalRunes, nil
}

// Returns an object accepted by the /lol-perks/v1/perks LCU endpoint.
func RuneArrayToObject(runes []uint16, championId uint16, role string) (runesObject *fastjson.Value) {
	a := fastjson.Arena{}

	r := a.NewObject()

	var desc string
	if role == "" {
		desc = "ARAM"
	} else {
		desc = role
	}

	r.Set("name", a.NewString("[GTools] "+championNames[championId]+" "+desc))
	r.Set("primaryStyleId", a.NewNumberInt(int(runes[0])))
	r.Set("subStyleId", a.NewNumberInt(int(runes[1])))

	// remove the first two runes from runes
	runes = runes[2:]
	runeArray := a.NewArray()
	for i := 0; i < len(runes); i++ {
		runeArray.Set(fmt.Sprint(i), a.NewNumberInt(int(runes[i])))
	}

	r.Set("selectedPerkIds", runeArray)
	r.Set("current", a.NewTrue())

	return r
}
