package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/google/uuid"
	"github.com/pkg/math"
)

// https://github.com/gofiber/fiber
// https://yalantis.com/blog/how-to-build-websockets-in-go/
// https://yalantis.com/blog/golang-vs-nodejs-comparison/

// https://blog.golang.org/json
// https://github.com/gorilla/websocket/blob/master/examples/chat/main.go
// https://github.com/googollee/go-socket.io
// https://scene-si.org/2017/09/27/things-to-know-about-http-in-go/
// https://dev.to/bcanseco/request-body-encoding-json-x-www-form-urlencoded-ad9

const maxMsgs = 8
const defaultPortNum = 3000

var userSymbols map[string]string
var symbolIndex = 0
var userMessages []UserMessage

type UserMessage struct {
	UserId  string `json:"userId"`
	Message string `json:"message"`
	Symbol  string `json:"symbol"`
}

func getUserId(ctx *fiber.Ctx) string {
	var userMessage UserMessage
	ctx.BodyParser(&userMessage)
	userId := ctx.Cookies("userId")
	if len(userId) > 0 {
		return userId
	}
	userId = uuid.New().String()
	setCookie(ctx, "userId", userId)
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
func getUserMessage(ctx *fiber.Ctx) (UserMessage, error) {
	var userMessage UserMessage
	err := ctx.BodyParser(&userMessage)
	if err != nil {
		errMsg := "error cannot parse"
		ctx.Status(fiber.StatusBadRequest).SendString(errMsg)
		return userMessage, errors.New(errMsg) // TODO: what is correct here?
	}
	if userMessage.Message == "" {
		return userMessage, errors.New("no message")
	}
	return userMessage, nil
}
func setCookie(ctx *fiber.Ctx, name string, value string) {
	cookie := new(fiber.Cookie)
	cookie.Name = name
	cookie.Value = value
	cookie.Expires = time.Now().Add(24 * time.Hour)
	ctx.Cookie(cookie)
}
func cooldownTest(ctx *fiber.Ctx) bool {
	now := time.Now().Unix()
	if cookie := ctx.Cookies("userDate"); cookie != "" {
		if then, err := strconv.ParseInt(cookie, 10, 64); err == nil {
			cooldown := (now - then) >= 1
			//fmt.Printf("now= %d\nthen=%d\n", now, then)
			return cooldown
		}
	}
	return false
}
func cooldownError(ctx *fiber.Ctx) {
	const errMsg = "403 forbidden: cooldown failed"
	log.Println(errMsg)
	ctx.Status(fiber.StatusForbidden).SendString(errMsg)
}
func uniquenessTest(ctx *fiber.Ctx, userMessage UserMessage) bool {
	currentMessage := userMessage.Message
	if cookie := ctx.Cookies("userMessage"); cookie != "" {
		previousMessage := cookie
		unique := previousMessage != currentMessage
		//fmt.Printf("prev=%s\ncurr=%s\nuniq=%v\n", previousMessage, currentMessage, unique)
		return unique
	}
	return false
}
func uniquenessError(ctx *fiber.Ctx, userMessage UserMessage) {
	currentMessage := userMessage.Message
	previousMessage := ctx.Cookies("userMessage")
	if previousMessage != "" {
		errMsg := fmt.Sprintf("403 forbidden: message not unique because '%s' = '%s'", previousMessage, currentMessage)
		log.Println(errMsg)
		ctx.Status(fiber.StatusForbidden).SendString(errMsg)
	}
}
func userMessageHandler(ctx *fiber.Ctx) error {
	ctx.Method()
	if ctx.Method() != "POST" {
		ctx.Status(fiber.StatusForbidden)
		return errors.New("403 not suported")
	}
	if ctx.Path() != "/UserMessage" {
		ctx.Status(fiber.StatusNotFound)
		return errors.New("404 not found")
	}

	userMessage, err := getUserMessage(ctx)
	if err != nil {
		return err
	}
	if len(userMessage.Message) > 0 {
		userDate := strconv.FormatInt(time.Now().Unix(), 10)
		setCookie(ctx, "userDate", userDate)
		if strings.HasPrefix(userMessage.Message, "/n") {
			// todo: custom symbol command
			return nil
		}
		if strings.HasPrefix(userMessage.Message, "/reset") {
			// todo: reset command
			return nil
		}
		if strings.HasPrefix(userMessage.Message, "/info") {
			// todo: info command
			return nil
		}
		if strings.HasPrefix(userMessage.Message, "/") {
			// unknown command
			return nil
		}
		if !cooldownTest(ctx) {
			cooldownError(ctx)
			return nil
		}
		if !uniquenessTest(ctx, userMessage) {
			uniquenessError(ctx, userMessage)
			return nil
		}
		userId := getUserId(ctx)
		userSymbol := getUserSymbol(userId)
		setCookie(ctx, "userMessage", userMessage.Message)
		userMessageOut := UserMessage{userId, userMessage.Message, userSymbol}
		fmt.Printf("/UserMessage: %+v\n", userMessageOut)
		userMessages = append(userMessages, userMessageOut)
		// todo: send broadcast message via websocket
	}

	userMessages = userMessages[math.Max(0, len(userMessages)-maxMsgs):]
	if json, err := json.Marshal(userMessages); err == nil {
		ctx.Send(json)
	}

	return nil
}
func getPort() int {
	portEnv := os.Getenv("PORT")
	portNum, err := strconv.ParseInt(portEnv, 10, 32)
	if err != nil || portNum == 0 {
		portNum = defaultPortNum
	}
	return int(portNum)
}
func main() {
	log.Println("--Golang/GoFiber--")

	userSymbols = make(map[string]string)

	app := fiber.New()
	app.Use(logger.New())
	app.Static("/", ".")
	app.Post("/UserMessage", userMessageHandler)

	addr := fmt.Sprintf(":%v", getPort())
	url := fmt.Sprintf("http://localhost%s", addr)
	log.Printf("listening: %s\n", url)
	log.Fatal(app.Listen(addr))
}
