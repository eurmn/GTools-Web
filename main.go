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
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
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
	"github.com/thoas/go-funk"
	"github.com/valyala/fastjson"
	"golang.org/x/sys/windows"
)

var (
	debug = flag.Bool("debug", false, "wether or not to enable debug mode - static files are not served in this case")
)

const (
	USER_INFO         = uint8(0)
	CHAMPION_CHANGE   = uint8(1)
	QUIT_CHAMP_SELECT = uint8(2)
	CDRAGON           = "https://raw.communitydragon.org/latest"
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
	ChampionId                string
	ChampionName              string
	RunesByPopularity         []Rune
	RunesByWinRate            []Rune
	ItemsByPopularity         []Item
	ItemsByWinRate            []Item
	StartingItemsByPopularity []Item
	StartingItemsByWinRate    []Item
	Role                      string
}

type RuneInfo struct {
	Name        string
	Description string
}

type ChampionBuild struct {
	StartingItems SortedItems
	Items         SortedItems
	Runes         SortedRunes
}

type SortedRunes struct {
	ByPopularity []uint16
	ByWinRate    []uint16
}

type SortedItems struct {
	ByPopularity []uint16
	ByWinRate    []uint16
}

type Rune struct {
	Id    uint16
	Asset string
	Info  RuneInfo
}

type Item struct {
	Id    uint16
	Asset string
	Name  string
}

var userInformation UserInformation
var upgrader = websocket.Upgrader{}
var wsQueue chan interface{}
var subscribers = []uint8{}
var championNames = map[uint16]string{}
var runeInfo = map[uint16]Rune{}
var itemInfo = map[uint16]Item{}
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

	itemInfo, err = GetUpdatedItemAssets()
	if err != nil {
		log.Fatalf("Failed to get updated item asset paths: %v", err)
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

	router.GET("/tier-list", func(c *gin.Context) {
		url := "https://league-champion-aggregate.iesdev.com/graphql?query=query" +
			url.QueryEscape(
				" TierList($region:Region,$queue:Queue,$tier:Tier){allChampionStats(region:$region,queue:$queue,tier:$tier,mostPopular:true)"+
					"{championId role patch wins games tierListTier{tierRank previousTierRank status}}}") +
			"&variables=" + url.QueryEscape("{\"queue\":\"SUMMONERS_RIFT_DRAFT_PICK\",\"region\":\"WORLD\"}")

		resp, err := http.Get(url)

		if err != nil {
			log.Printf("Failed to request: %v", err)
			c.Status(http.StatusInternalServerError)
			return
		}

		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Failed to read response: %v", err)
			c.Status(http.StatusInternalServerError)
			return
		}

		content, err := fastjson.ParseBytes(body)
		if err != nil {
			log.Printf("Failed to parse response: %v", err)
			c.Status(http.StatusInternalServerError)
			return
		}

		tierList := content.Get("data").GetArray("allChampionStats")
		a := fastjson.Arena{}

		topTierList := funk.Filter(tierList, func(champ *fastjson.Value) bool {
			return string(champ.GetStringBytes("role")) == "TOP"
		}).([]*fastjson.Value)
		jungleTierList := funk.Filter(tierList, func(champ *fastjson.Value) bool {
			return string(champ.GetStringBytes("role")) == "JUNGLE"
		}).([]*fastjson.Value)
		midTierList := funk.Filter(tierList, func(champ *fastjson.Value) bool {
			return string(champ.GetStringBytes("role")) == "MID"
		}).([]*fastjson.Value)
		adcTierList := funk.Filter(tierList, func(champ *fastjson.Value) bool {
			return string(champ.GetStringBytes("role")) == "ADC"
		}).([]*fastjson.Value)
		supTierList := funk.Filter(tierList, func(champ *fastjson.Value) bool {
			return string(champ.GetStringBytes("role")) == "SUPPORT"
		}).([]*fastjson.Value)

		sort.SliceStable(tierList, func(i, j int) bool {
			return tierList[i].GetFloat64("wins")/tierList[i].GetFloat64("games") > tierList[j].GetFloat64("wins")/tierList[j].GetFloat64("games")
		})
		sort.SliceStable(topTierList, func(i, j int) bool {
			return topTierList[i].GetFloat64("wins")/topTierList[i].GetFloat64("games") > topTierList[j].GetFloat64("wins")/topTierList[j].GetFloat64("games")
		})
		sort.SliceStable(jungleTierList, func(i, j int) bool {
			return jungleTierList[i].GetFloat64("wins")/jungleTierList[i].GetFloat64("games") > jungleTierList[j].GetFloat64("wins")/jungleTierList[j].GetFloat64("games")
		})
		sort.SliceStable(midTierList, func(i, j int) bool {
			return midTierList[i].GetFloat64("wins")/midTierList[i].GetFloat64("games") > midTierList[j].GetFloat64("wins")/midTierList[j].GetFloat64("games")
		})
		sort.SliceStable(adcTierList, func(i, j int) bool {
			return adcTierList[i].GetFloat64("wins")/adcTierList[i].GetFloat64("games") > adcTierList[j].GetFloat64("wins")/adcTierList[j].GetFloat64("games")
		})
		sort.SliceStable(supTierList, func(i, j int) bool {
			return supTierList[i].GetFloat64("wins")/supTierList[i].GetFloat64("games") > supTierList[j].GetFloat64("wins")/supTierList[j].GetFloat64("games")
		})

		tiers := []string{"S", "A", "B", "C", "D"}
		generateTierObject := func(currentTierList []*fastjson.Value) *fastjson.Value {
			tierArray := a.NewArray()

			i := 0
			for _, champ := range currentTierList {
				if champ.Get("tierListTier").Type() == fastjson.TypeNull {
					continue
				}

				id := uint16(champ.GetUint("championId"))
				champObj := a.NewObject()

				champObj.Set("role", champ.Get("role"))
				champObj.Set("name", a.NewString(championNames[id]))
				champObj.Set("id", champ.Get("championId"))
				champObj.Set("winrate", a.NewNumberFloat64(
					math.Round(champ.GetFloat64("wins")*10000/champ.GetFloat64("games"))/100,
				))
				champObj.Set("tier", a.NewString(tiers[champ.Get("tierListTier").GetInt("tierRank")-1]))

				tierArray.SetArrayItem(i, champObj)
				i++
			}

			return tierArray
		}

		returnedTierList := a.NewObject()
		returnedTierList.Set("ALL", generateTierObject(tierList))
		returnedTierList.Set("TOP", generateTierObject(topTierList))
		returnedTierList.Set("JUNGLE", generateTierObject(jungleTierList))
		returnedTierList.Set("MID", generateTierObject(midTierList))
		returnedTierList.Set("ADC", generateTierObject(adcTierList))
		returnedTierList.Set("SUP", generateTierObject(supTierList))

		c.Header("Content-Type", "application/json")
		c.String(200, string(returnedTierList.MarshalTo(nil)))
	})

	router.GET("/sample-build", func(c *gin.Context) {
		build, err := GetBuildForChampion(126, "MID", "RANKED_SOLO_5X5")
		if err != nil {
			log.Printf("Failed to get runes: %v", err)
			return
		}

		// runes by popularity
		assetsPop := []Rune{}
		for _, run := range build.Runes.ByPopularity {
			assetsPop = append(assetsPop, runeInfo[run])
		}

		// runes by win-rate
		assetsWr := []Rune{}
		for _, run := range build.Runes.ByWinRate {
			assetsWr = append(assetsWr, runeInfo[run])
		}

		// items by popularity
		itemsPop := []Item{}
		for _, item := range build.Items.ByPopularity {
			itemsPop = append(itemsPop, itemInfo[item])
		}

		// items by win-rate
		itemsWr := []Item{}
		for _, item := range build.Items.ByWinRate {
			itemsWr = append(itemsWr, itemInfo[item])
		}

		// starting items by popularity
		startItemsPop := []Item{}
		for _, item := range build.StartingItems.ByPopularity {
			startItemsPop = append(startItemsPop, itemInfo[item])
		}

		// items by win-rate
		startItemsWr := []Item{}
		for _, item := range build.StartingItems.ByWinRate {
			startItemsWr = append(startItemsWr, itemInfo[item])
		}

		msg, err := CreateWSMessage(CHAMPION_CHANGE, ChampionChange{
			ChampionId:                fmt.Sprint(126),
			ChampionName:              championNames[126],
			RunesByPopularity:         assetsPop,
			RunesByWinRate:            assetsWr,
			ItemsByPopularity:         itemsPop,
			ItemsByWinRate:            itemsWr,
			StartingItemsByPopularity: startItemsPop,
			StartingItemsByWinRate:    startItemsWr,
			Role:                      "Mid",
		})

		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}

		c.JSON(http.StatusOK, msg)
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

	router.POST("/import-items", func(c *gin.Context) {
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

		itemsJson := bodyJson.GetArray("items")
		startingItemsJson := bodyJson.GetArray("starting_items")
		championIdString := string(bodyJson.GetStringBytes("champion_id"))
		role := string(bodyJson.GetStringBytes("role"))

		championId, err := strconv.ParseUint(championIdString, 10, 16)
		if err != nil {
			log.Printf("Failed to parse champion id: %v", err)
			c.Status(http.StatusBadRequest)
			return
		}

		items := []uint16{}
		for _, item := range itemsJson {
			items = append(items, uint16(item.GetInt()))
		}

		startingItems := []uint16{}
		for _, item := range startingItemsJson {
			startingItems = append(startingItems, uint16(item.GetInt()))
		}

		itemsLCU := ItemArrayToObject(startingItems, items, uint16(championId), role)
		itemBytes := itemsLCU.MarshalTo(nil)

		// LCU uses a self-signed certificate, so we need to disable TLS verification.
		// https://developer.riotgames.com/docs/lol#game-client-api_root-certificate-ssl-errors
		lcuClient := &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		}

		r, err := http.NewRequest(http.MethodPost, "https://"+authInfo.Url+"/lol-item-sets/v1/item-sets/"+userInformation.SummonerId+"/sets", bytes.NewBuffer(itemBytes))
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

		log.Println(string(body))
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

	var port string
	if *debug {
		port = "3000"
	} else {
		port = "4246"
	}

	// Open url in browser
	exec.Command("rundll32", "url.dll,FileProtocolHandler", "http://127.0.0.1:"+port).Start()
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
			_, err := io.ReadAll(res.Body)
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
		case "OnJsonApiEvent_lol-champ-select_v1_session", "OnJsonApiEvent_lol-champ-select-legacy_v1_session":
			if string(j.Get("2").GetStringBytes("eventType")) == "Delete" {
				EmitEvent(QUIT_CHAMP_SELECT, nil)
			} else {
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

					// 430 = blink pick; 450 = aram; 900 = arurf.
					queueId := content.GetInt("queueId")

					// the champ select endpoint will handle it, skip.
					if string(j.GetStringBytes("1")) == "OnJsonApiEvent_lol-champ-select-legacy_v1_session" && queueId == 400 {
						return
					}

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

				build, err := GetBuildForChampion(cid, role, queue)
				if err != nil {
					log.Printf("Failed to get runes: %v", err)
					continue
				}

				// runes by popularity
				runesPop := []Rune{}
				for _, run := range build.Runes.ByPopularity {
					runesPop = append(runesPop, runeInfo[run])
				}

				// runes by win-rate
				runesWr := []Rune{}
				for _, run := range build.Runes.ByWinRate {
					runesWr = append(runesWr, runeInfo[run])
				}

				// items by popularity
				itemsPop := []Item{}
				for _, item := range build.Items.ByPopularity {
					itemsPop = append(itemsPop, itemInfo[item])
				}

				// items by win-rate
				itemsWr := []Item{}
				for _, item := range build.Items.ByWinRate {
					itemsWr = append(itemsWr, itemInfo[item])
				}

				// starting items by popularity
				startItemsPop := []Item{}
				for _, item := range build.StartingItems.ByPopularity {
					startItemsPop = append(startItemsPop, itemInfo[item])
				}

				// items by win-rate
				startItemsWr := []Item{}
				for _, item := range build.StartingItems.ByWinRate {
					startItemsWr = append(startItemsWr, itemInfo[item])
				}

				EmitEvent(CHAMPION_CHANGE, ChampionChange{
					ChampionId:                fmt.Sprint(cid),
					ChampionName:              championNames[cid],
					RunesByPopularity:         runesPop,
					RunesByWinRate:            runesWr,
					ItemsByPopularity:         itemsPop,
					ItemsByWinRate:            itemsWr,
					StartingItemsByPopularity: startItemsPop,
					StartingItemsByWinRate:    startItemsWr,
					Role:                      role,
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
			"type":                      eventType,
			"id":                        data.(ChampionChange).ChampionId,
			"name":                      data.(ChampionChange).ChampionName,
			"role":                      data.(ChampionChange).Role,
			"runesByPopularity":         data.(ChampionChange).RunesByPopularity,
			"runesByWinRate":            data.(ChampionChange).RunesByWinRate,
			"itemsByPopularity":         data.(ChampionChange).ItemsByPopularity,
			"itemsByWinRate":            data.(ChampionChange).ItemsByWinRate,
			"startingItemsByPopularity": data.(ChampionChange).StartingItemsByPopularity,
			"startingItemsByWinRate":    data.(ChampionChange).StartingItemsByWinRate,
		}, nil
	case QUIT_CHAMP_SELECT:
		return map[string]interface{}{
			"type": eventType,
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

// Get all non-loopback IP Adresses of this device.
func GetIPAddresses() ([]string, error) {
	addresses, err := net.InterfaceAddrs()
	if err != nil {
		return nil, err
	}
	var ipAddresses []string
	for _, address := range addresses {
		// check the address type and if it is not a loopback the display it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ipAddresses = append(ipAddresses, ipnet.IP.String())
			}
		}
	}
	return ipAddresses, nil
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

lpLoop:
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
					time.Sleep(time.Second * 2)
					continue lpLoop
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

// Get the icon path for every item.
func GetUpdatedItemAssets() (items map[uint16]Item, err error) {
	res, err := http.Get(CDRAGON + "/plugins/rcp-be-lol-game-data/global/default/v1/items.json")
	if err != nil {
		return items, err
	}

	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return items, err
	}

	content, err := fastjson.ParseBytes(body)
	if err != nil {
		return items, err
	}

	items = map[uint16]Item{}
	for _, item := range content.GetArray() {
		itemId := item.Get("id").GetUint()
		itemPath := string(item.GetStringBytes("iconPath"))
		itemPath = CDRAGON +
			strings.ToLower(
				strings.Replace(itemPath, "/lol-game-data/assets/ASSETS/", "/plugins/rcp-be-lol-game-data/global/default/assets/", 1),
			)
		itemName := string(item.GetStringBytes("name"))

		items[uint16(itemId)] = Item{
			Id:    uint16(itemId),
			Asset: itemPath,
			Name:  itemName,
		}
	}

	return items, nil
}

// Get the icon path for every rune.
func GetUpdatedRuneAssets() (runes map[uint16]Rune, err error) {
	res, err := http.Get(CDRAGON + "/plugins/rcp-be-lol-game-data/global/default/v1/perkstyles.json")
	if err != nil {
		return runes, err
	}

	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return runes, err
	}

	content, err := fastjson.ParseBytes(body)
	if err != nil {
		return runes, err
	}

	runes = map[uint16]Rune{}
	for _, run := range content.GetArray("styles") {
		runeId := run.Get("id").GetUint()
		runePath := string(run.GetStringBytes("iconPath"))
		runePath = CDRAGON +
			strings.ToLower(
				strings.Replace(runePath, "/lol-game-data/assets/v1/", "/plugins/rcp-be-lol-game-data/global/default/v1/", 1),
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
func GetBuildForChampion(championId uint16, role string, queue string) (build ChampionBuild, err error) {
	url := BlitzGGUrlForChampion(championId, role, queue)
	res, err := http.Get(url)
	if err != nil {
		return build, err
	}

	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return build, err
	}

	content, err := fastjson.ParseBytes(body)
	if err != nil {
		return build, err
	}

	allRunes := [][]uint16{}
	allItems := [][]uint16{}
	allStartingItems := [][]uint16{}
	winRate := false

	for i := 0; i < 2; i++ {
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

		startingItems := []uint16{}
		completedItems := []uint16{}

		// Get starting items
		selectStartItems := selectedBuild.GetArray("startingItems")
		sort.SliceStable(selectStartItems, func(i, j int) bool {
			if winRate {
				// sort by win-rate:
				return selectStartItems[i].GetInt("wins")/selectStartItems[i].GetInt("games") > selectStartItems[j].GetInt("wins")/selectStartItems[j].GetInt("games")
			} else {
				// sort by popularity
				return selectStartItems[i].GetInt("games") > selectStartItems[j].GetInt("games")
			}
		})

		for _, item := range selectStartItems[0].GetArray("startingItemIds") {
			startingItems = append(startingItems, uint16(item.GetUint()))
		}

		// Get completed items
		selectedItems := selectedBuild.GetArray("completedItems")

		// Sometimes it gives us the first 4 items, sometimes only the
		// first 3. We need to check if we have 4 items or not.
		sort.SliceStable(selectedItems, func(i, j int) bool {
			return selectedItems[i].GetInt("index") > selectedItems[j].GetInt("index")
		})
		lastIndex := selectedItems[0].GetInt("index") + 1

		for i := 0; i < lastIndex; i++ {
			currentIndex := []*fastjson.Value{}
			for _, item := range selectedItems {
				if item.GetInt("index") == i {
					currentIndex = append(currentIndex, item)
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

			completedItems = append(completedItems, uint16(currentIndex[0].GetInt("itemId")))
		}

		mythicIndex := selectedBuild.GetInt("mythicAverageIndex")
		mythicId := selectedBuild.GetUint("mythicId")
		completedItems = append(completedItems[:mythicIndex+1], completedItems[mythicIndex:]...)
		completedItems[mythicIndex] = uint16(mythicId)

		allItems = append(allItems, completedItems)
		allStartingItems = append(allStartingItems, startingItems)
		allRunes = append(allRunes, finalRunes)
		winRate = true
	}

	return ChampionBuild{
		Runes: SortedRunes{
			ByPopularity: allRunes[0],
			ByWinRate:    allRunes[1],
		},
		StartingItems: SortedItems{
			ByPopularity: allStartingItems[0],
			ByWinRate:    allStartingItems[1],
		},
		Items: SortedItems{
			ByPopularity: allItems[0],
			ByWinRate:    allItems[1],
		},
	}, nil
}

// Returns an object accepted by /lol-item-sets/v1/item-sets/{summonerId}/sets
func ItemArrayToObject(startingItems []uint16, items []uint16, championId uint16, role string) (runesObject *fastjson.Value) {
	a := fastjson.Arena{}
	it := a.NewObject()

	var desc string
	if role == "" {
		desc = "ARAM"
	} else {
		desc = role
	}

	desc = string(desc[0]) + strings.ToLower(desc[1:])

	it.Set("title", a.NewString("[GTools] "+championNames[championId]+" "+desc))

	c := a.NewArray()
	c.SetArrayItem(0, a.NewNumberInt(int(championId)))
	it.Set("associatedChampions", c)

	// Assign to both SR and ARAM
	maps := a.NewArray()
	maps.SetArrayItem(0, a.NewNumberInt(11))
	if desc == "ARAM" {
		maps.SetArrayItem(1, a.NewNumberInt(12))
	}
	it.Set("associatedMaps", a.NewArray())

	blocks := a.NewArray()

	// starting items
	startBlock := a.NewObject()
	itemsObject := a.NewArray()
	for i, item := range startingItems {
		itemObject := a.NewObject()
		itemObject.Set("id", a.NewString(fmt.Sprint(item)))
		itemObject.Set("count", a.NewNumberInt(1))
		itemsObject.SetArrayItem(i, itemObject)
	}
	startBlock.Set("items", itemsObject)
	startBlock.Set("type", a.NewString("Starting Items"))

	// completed items
	completedBlock := a.NewObject()
	itemsObject = a.NewArray()
	for i, item := range items {
		itemObject := a.NewObject()
		itemObject.Set("id", a.NewString(fmt.Sprint(item)))
		itemObject.Set("count", a.NewNumberInt(1))
		itemsObject.SetArrayItem(i, itemObject)
	}
	completedBlock.Set("items", itemsObject)
	completedBlock.Set("type", a.NewString("Full Items"))

	blocks.SetArrayItem(0, startBlock)
	blocks.SetArrayItem(1, completedBlock)

	it.Set("blocks", blocks)

	return it
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

	desc = string(desc[0]) + strings.ToLower(desc[1:])

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
