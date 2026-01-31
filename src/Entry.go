//go:build ecmascript

package main

import (
	"syscall/js"

	"github.com/ScriptTiger/jsGo"
)

const (
	// App name used for titling various things
	appName = "TigerShare"

	// Chunk size for file transfer
	chunkSize = 64 * 1024
)

var (

	// Location elements
	urlOrigin, url, urlFull, pid, tid string

	// Global JS objects
	app, peer js.Value

	// File tracking
	file js.Value
	fileName string
	fileSize int

	// Status tracking
	hasPage, connected bool
	destroyed = true

	// TURN settings
	turnUrl, turnUser, turnCred, policy string
)

// Set up app, making sure it has everything it needs to proceed
func main() {

	// Get app location in DOM
	app = jsGo.GetElementById("app")

	// Check URL query strings
	hasPID := jsGo.Params.Call("has", "pid").Bool()
	hasTID := jsGo.Params.Call("has", "tid").Bool()
	hasTurnUrl := jsGo.Params.Call("has", "turnurl").Bool()
	hasTurnUser := jsGo.Params.Call("has", "turnuser").Bool()
	hasTurnCred := jsGo.Params.Call("has", "turncred").Bool()
	hasPolicy  := jsGo.Params.Call("has", "policy").Bool()

	// Store regular strings directly
	if hasPID {pid = jsGo.Params.Call("get", "pid").String()}
	if hasTID {tid = jsGo.Params.Call("get", "tid").String()}
	if hasPolicy {policy = jsGo.Params.Call("get", "policy").String()}

	// Decode base64 URL-safe strings back to regular strings and store them
	if hasTurnUrl {turnUrl = urlToString(jsGo.Params.Call("get", "turnurl").String())}
	if hasTurnUser {turnUser = urlToString(jsGo.Params.Call("get", "turnuser").String())}
	if hasTurnCred {turnCred = urlToString(jsGo.Params.Call("get", "turncred").String())}

	// Capture URL
	jsURL := jsGo.URL.New(jsGo.Location.Get("href"))
	urlOrigin = jsURL.Call("toString").String()
	jsURL.Set("search", "")
	url = jsURL.Call("toString").String()
	urlFull = url

	// Wipe current query strings without reloading
	jsGo.History.Call("replaceState", nil, nil, url)

	// If no PID and/or TID given, present user to start serving a download
	if !(hasPID && hasTID) {

		// Prepare page
		jsGo.Document.Set("title", appName)
		app.Set("innerHTML", nil)

		// TURN settings container
		var turnSettingsVisible bool
		turnSettings := jsGo.CreateElement("div")
		turnSettings.Set("hidden", true)

		// TURN URL input
		turnUrlLabel := jsGo.CreateElement("label")
		turnUrlLabel.Set("textContent", "URL:")
		turnSettings.Call("appendChild", turnUrlLabel)
		turnSettings.Call("appendChild", jsGo.CreateElement("br"))
		turnUrlField := jsGo.CreateElement("input")
		if hasTurnUrl {turnUrlField.Set("value", turnUrl)}
		turnSettings.Call("appendChild", turnUrlField)
		turnSettings.Call("appendChild", jsGo.CreateElement("br"))

		// TURN user input
		turnUserLabel := jsGo.CreateElement("label")
		turnUserLabel.Set("textContent", "User Name:")
		turnSettings.Call("appendChild", turnUserLabel)
		turnSettings.Call("appendChild", jsGo.CreateElement("br"))
		turnUserField := jsGo.CreateElement("input")
		if hasTurnUser {turnUserField.Set("value", turnUser)}
		turnSettings.Call("appendChild", turnUserField)
		turnSettings.Call("appendChild", jsGo.CreateElement("br"))

		// TURN credential input
		turnCredLabel := jsGo.CreateElement("label")
		turnCredLabel.Set("textContent", "Credential:")
		turnSettings.Call("appendChild", turnCredLabel)
		turnSettings.Call("appendChild", jsGo.CreateElement("br"))
		turnCredField := jsGo.CreateElement("input")
		turnCredField.Set("type", "password")
		if hasTurnCred {turnCredField.Set("value", turnCred)}
		turnSettings.Call("appendChild", turnCredField)
		turnSettings.Call("appendChild", jsGo.CreateElement("br"))

		// TURN policy selection
		policyLabel := jsGo.CreateElement("label")
		policyLabel.Set("textContent", "ICE Transport Policy:")
		turnSettings.Call("appendChild", policyLabel)
		turnSettings.Call("appendChild", jsGo.CreateElement("br"))
		policySelect := jsGo.CreateElement("select")
		policyAll := jsGo.CreateElement("option")
		policyAll.Set("value", "all")
		policyAll.Set("textContent", "All")
		policyAll.Set("title", "Select the best network path of all candidates, even if it is not through TURN")
		policySelect.Call("appendChild", policyAll)
		policyRelay := jsGo.CreateElement("option")
		policyRelay.Set("value", "relay")
		policyRelay.Set("textContent", "Relay")
		policyRelay.Set("title", "Select the best network path of only relay candidates to ensure only TURN is used")
		policySelect.Call("appendChild", policyRelay)
		if hasPolicy {policySelect.Set("value", policy)
		} else {policySelect.Set("value", "all")}
		turnSettings.Call("appendChild", policySelect)
		turnSettings.Call("appendChild", jsGo.CreateElement("br"))

		// Append the TURN settings container to the app container
		appAppendChild(turnSettings)

		// Toggle TURN settings
		turnButton := jsGo.CreateButton("TURN Settings", func() {
			if turnSettingsVisible{
				turnSettings.Set("hidden", true)
				turnSettingsVisible = false
			} else {
				turnSettings.Set("hidden", false)
				turnSettingsVisible = true
			}
		})
		appAppendChild(turnButton)

		// Three-ling break under TURN settings
		appAppendChild(jsGo.CreateElement("br"))
		appAppendChild(jsGo.CreateElement("br"))
		appAppendChild(jsGo.CreateElement("br"))

		// File button to select file
		appAppendChild(
			jsGo.CreateLoadFileButton("File", "*/*", false, func(files js.Value) {

				// Capture TURN settings
				turnUrl = turnUrlField.Get("value").String()
				turnUser = turnUserField.Get("value").String()
				turnCred = turnCredField.Get("value").String()
				policy = policySelect.Get("value").String()

				// If TURN settings given, capture to complete URL and log
				if turnUrl != "" && turnUser != "" && turnCred != "" {
					urlFull = url+
						"?turnurl="+stringToUrl(turnUrl)+
						"&turnuser="+stringToUrl(turnUser)+
						"&turncred="+stringToUrl(turnCred)+
						"&policy="+policy
					jsGo.Log("Full URL (THIS URL CONTAINS YOUR PERSONAL TURN ACCOUNT, SO USE AT YOUR OWN RISK): "+urlFull)
				}

				// Capture file and metadata
				file = files.Index(0)
				fileName = file.Get("name").String()
				fileSize = file.Get("size").Int()

				// Load required JS libraries and start serving the file
				jsGo.LoadJS("https://cdn.jsdelivr.net/npm/peerjs@1.5.5/dist/peerjs.min.js", func() {
					jsGo.LoadJS("https://cdn.jsdelivr.net/npm/qrcodejs@1.0.0/qrcode.min.js", func() {
						server()
					})
				})
			}),
		)

	// If there was both a PID and TID given, allow client to attempt downloading the file
	} else {
		// Load PeerJS and proceed to download page
		jsGo.LoadJS("https://cdn.jsdelivr.net/npm/peerjs@1.5.5/dist/peerjs.min.js", func() {
			client()
		})
	}
}
