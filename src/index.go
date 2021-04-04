package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/pkg/math"
)

// https://blog.golang.org/json
// https://github.com/gorilla/websocket/blob/master/examples/chat/main.go // golang example
// https://github.com/googollee/go-socket.io
// https://scene-si.org/2017/09/27/things-to-know-about-http-in-go/
// https://dev.to/bcanseco/request-body-encoding-json-x-www-form-urlencoded-ad9

const maxMsgs = 8
const defaultPort = 3000

var userSymbols map[string]string
var symbolIndex = 0
var userMessages []UserMessage

type UserMessage struct {
	UserId  string `json:"userId"`
	Message string `json:"message"`
	Symbol  string `json:"symbol"`
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func fileHandler(file string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, file)
	}
}
func getUserId(w http.ResponseWriter, r *http.Request) string {
	cookieIn, err := r.Cookie("userId")
	if err == nil {
		return cookieIn.Value
	}
	userId := uuid.New().String()
	oneMonth := 60 * 60 * 24 * 30
	cookieOut := http.Cookie{
		Name:   "userId",
		Value:  userId,
		MaxAge: oneMonth,
	}
	http.SetCookie(w, &cookieOut)
	return userId
}
func getUserSymbol(userId string) string {
	symbols := []string{"🔴", "🟡", "🟢", "🟣", "🔵", "🟠", "🟤", "⚪️"}
	symbol := userSymbols[userId]
	if symbol == "" {
		symbol = symbols[symbolIndex%len(symbols)]
		userSymbols[userId] = symbol
		symbolIndex++
	}
	return symbol
}
func getUserMessage(w http.ResponseWriter, r *http.Request) string {
	if r.Body == nil {
		return ""
	}
	var userMessage UserMessage
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&userMessage); err != nil {
		return ""
	}
	return userMessage.Message
}
func userMessageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "403 not suported", http.StatusForbidden)
		return
	}
	if r.URL.Path != "/UserMessage" {
		http.Error(w, "404 not found", http.StatusNotFound)
		return
	}

	userMessage := getUserMessage(w, r)
	if len(userMessage) > 0 {
		userId := getUserId(w, r)
		userSymbol := getUserSymbol(userId)
		userMessageOut := UserMessage{userId, userMessage, userSymbol}
		fmt.Printf("/UserMessage: %+v\n", userMessageOut)
		userMessages = append(userMessages, userMessageOut)
	}
	userMessages = userMessages[math.Max(0, len(userMessages)-maxMsgs):]

	if json, err := json.Marshal(userMessages); err == nil {
		w.Header().Set("Content-Type", "application/json")
		w.Write(json)
	}
}
func websocketHandler(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Client Connected")
	err = ws.WriteMessage(1, []byte("broadcast"))
	if err != nil {
		log.Fatal(err)
	}
}
func main() {
	http.HandleFunc("/", fileHandler("index.html"))
	http.HandleFunc("/index.css", fileHandler("index.css"))
	http.HandleFunc("/UserMessage", userMessageHandler)
	http.HandleFunc("/ws", websocketHandler)

	userSymbols = make(map[string]string)

	portEnv := os.Getenv("PORT")
	port, err := strconv.Atoi(portEnv)
	if err != nil || port == 0 {
		port = defaultPort
	}
	addr := fmt.Sprintf(":%v", port)
	url := fmt.Sprintf("http://localhost%s", addr)
	fmt.Printf("listening: %s\n", url)
	log.Fatal(http.ListenAndServe(addr, nil))
}
