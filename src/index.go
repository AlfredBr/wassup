package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

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
const oneMonth = 60 * 60 * 24 * 30

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
func getUserMessage(w http.ResponseWriter, r *http.Request) (UserMessage, error) {
	var userMessage UserMessage
	if r.Body != nil {
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&userMessage); err != nil {
			return userMessage, errors.New("no message")
		}
	}
	return userMessage, nil
}
func setCookie(w http.ResponseWriter, r *http.Request, name string, value string) {
	cookie := http.Cookie{
		Name:   name,
		Value:  value,
		MaxAge: oneMonth,
	}
	http.SetCookie(w, &cookie)
}
func cooldownTest(w http.ResponseWriter, r *http.Request) bool {
	var then int64
	now := time.Now().Unix()
	if cookie, err := r.Cookie("userDate"); err == nil {
		if then, err = strconv.ParseInt(cookie.Value, 10, 64); err == nil {
			cooldown := (now - then) >= 1
			//fmt.Printf("now= %d\nthen=%d\n", now, then)
			return cooldown
		}
	}
	return false
}
func cooldownError(w http.ResponseWriter, r *http.Request) {
	const errMsg = "403 forbidden: cooldown failed"
	log.Println(errMsg)
	http.Error(w, errMsg, http.StatusForbidden)
}
func uniquenessTest(w http.ResponseWriter, r *http.Request, userMessage UserMessage) bool {
	currentMessage := userMessage.Message
	if cookie, err := r.Cookie("userMessage"); err == nil {
		previousMessage := cookie.Value
		unique := previousMessage != currentMessage
		//fmt.Printf("prev=%s\ncurr=%s\nuniq=%v\n", previousMessage, currentMessage, unique)
		return unique
	}
	return false
}
func uniquenessError(w http.ResponseWriter, r *http.Request, userMessage UserMessage) {
	currentMessage := userMessage.Message
	cookie, err := r.Cookie("userMessage")
	if err == nil {
		errMsg := fmt.Sprintf("403 forbidden: message not unique because '%s' = '%s'", cookie.Value, currentMessage)
		log.Println(errMsg)
		http.Error(w, errMsg, http.StatusForbidden)
	}
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

	userMessage, err := getUserMessage(w, r)
	if err != nil {
		return
	}
	if len(userMessage.Message) > 0 {
		userDate := strconv.FormatInt(time.Now().Unix(), 10)
		setCookie(w, r, "userDate", userDate)
		if strings.HasPrefix(userMessage.Message, "/n") {
			// todo: custom symbol command
			return
		}
		if strings.HasPrefix(userMessage.Message, "/reset") {
			// todo: reset command
			return
		}
		if strings.HasPrefix(userMessage.Message, "/info") {
			// todo: info command
			return
		}
		if strings.HasPrefix(userMessage.Message, "/") {
			// unknown command
			return
		}
		if !cooldownTest(w, r) {
			cooldownError(w, r)
			return
		}
		if !uniquenessTest(w, r, userMessage) {
			uniquenessError(w, r, userMessage)
			return
		}
		userId := getUserId(w, r)
		userSymbol := getUserSymbol(userId)
		setCookie(w, r, "userMessage", userMessage.Message)
		userMessageOut := UserMessage{userId, userMessage.Message, userSymbol}
		fmt.Printf("/UserMessage: %+v\n", userMessageOut)
		userMessages = append(userMessages, userMessageOut)
		// todo: send broadcast message via websocket
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
	port, err := strconv.ParseInt(portEnv, 10, 32)
	if err != nil || port == 0 {
		port = defaultPort
	}
	addr := fmt.Sprintf(":%v", port)
	url := fmt.Sprintf("http://localhost%s", addr)
	log.Printf("listening: %s\n", url)
	log.Fatal(http.ListenAndServe(addr, nil))
}
