// https://jquery.com/
// https://nodejs.org/en/
// http://expressjs.com/
// https://socket.io/
// https://github.com/socketio/socket.io
// https://socket.io/docs/v4/index.html
// https://socket.io/docs/v3/client-installation/
// https://socket.io/docs/v3/client-initialization/
// https://socket.io/docs/v3/emitting-events/
// https://nodejs.org/docs/latest/api/events.html#events_events
// https://www.w3schools.com/charsets/ref_utf_geometric.asp
// https://www.w3schools.com/charsets/ref_emoji.asp
// https://docs.microsoft.com/en-us/azure/app-service/configure-language-nodejs?pivots=platform-linux
// https://socket.io/docs/v3/emit-cheatsheet/index.html
// https://www.tutorialspoint.com/expressjs/expressjs_cookies.htm#:~:text=To%20set%20a%20new%20cookie%2C%20let%20us%20define,%27express%27%29.send%28%27cookie%20set%27%29%3B%20%2F%2FSets%20name%20%3D%20express%20%7D%29%3B%20app.listen%283000%29%3B
// https://stackoverflow.com/questions/16209145/how-to-set-cookie-in-node-js-using-express-framework
// http://expressjs.com/en/5x/api.html#res.cookie
// http://expressjs.com/en/5x/api.html#req.cookies

// To upgrade NodeJS (on Linux)
// curl -fsSL https://deb.nodesource.com/setup_current.x | sudo -E bash -
// sudo apt-get install -y nodejs

// $ npm install express --save
// $ npm install socket.io --save
// $ npm install cookie-parser --save
// $ npm install uuid --save

"use strict"
console.log("--JavaScript/Node/Express--");
const uuid = require('uuid');
const port = process.env.PORT || 3000;
const express = require('express');
const app = express();
const server = require('http').createServer(app);
const io = require('socket.io')(server);
const cookieParser = require('cookie-parser');
const users = [];
var messages = [];

io.on("connection", (socket) => {
    //console.log(`connection: socket=${socket.id}`);
    socket.on('disconnect', () => {
        console.log(`disconnect: socket=${socket.id}`);
    });
});

var getUserId = function(req, res) {
    var userId = "";
    if (!!req.cookies.userId) {
        userId = req.cookies.userId;
    }
    else {
        userId = uuid.v4();
        res.cookie("userId", userId);
    }
    if (users.indexOf(userId) === -1) {
        users.push(userId);
    }
    //console.log(`getUserId() returns '${userId}'`)
    return userId;
};

var getUserSymbol = function(req, res) {
    var userSymbol = "";
    if (!!req.cookies.userSymbol)
    {    
        userSymbol = req.cookies.userSymbol;
    }
    else {
        const userId = getUserId(req, res);
        const symbols = ["🔴", "🟡", "🟢", "🟣", "🔵", "🟠", "🟤", "⚪️"];
        var i = users.indexOf(userId);
        userSymbol = symbols[i % symbols.length];
        res.cookie("userSymbol", userSymbol);
    }
    //console.log(`getUserSymbol() returns '${userSymbol}'`)
    return userSymbol;
};

var getUserMessage = function(req, res) {
    var userMessage = "";
    if (req.body.message?.length > 0) {
        userMessage = req.body.message;
    }    
    //console.log(`getUserMessage() returns '${userMessage}'`)
    return userMessage;
};

app.use(express.json());
app.use(express.urlencoded({
    extended: true
}));
app.use(cookieParser());
app.use(express.static('.'));

app.get('/', (req, res) => {
    console.log(`root: ${req.cookies.userId}`);
});
app.post('/UserMessage', (req, res) => {
    const userId = getUserId(req, res);
    const userSymbol = getUserSymbol(req, res);
    const userMessage = getUserMessage(req, res);
    
    if (userMessage?.length > 0) {
        const timespan = (((new Date()).getTime()) - req.cookies.userDate);
        const cooldown = req.cookies.userDate ? timespan >= 1000 : true;
        if (!cooldown) {
            const errMsg = `forbidden: no cooldown because previous message was ${timespan}ms ago`;
            console.log(errMsg);
            res.status(403).send(errMsg);
            return;
        }
        const unique = req.cookies.userMessage !== userMessage;
        if (!unique) {
            const errMsg = `forbidden: not unique because '${req.cookies.userMessage}' = '${req.body.message}'`;
            console.log(errMsg);
            res.status(403).send(errMsg);
            return;
        }
        if (req.body.message.toLowerCase().indexOf("/n") == 0) {
            res.cookie("userSymbol", req.body.message.substring(2).trim());
            res.send("");
            return;
        }      
        if (req.body.message.toLowerCase().indexOf("/info") == 0) {
            console.log(`/info userId=${userId}, userSymbol=${userSymbol}`);
            res.send("");
            return;
        }      
        if (req.body.message.toLowerCase().indexOf("/") == 0) {
            res.send("");
            return;
        }
        res.cookie("userMessage", userMessage);
        res.cookie("userDate", (new Date()).getTime());
        const userMessageOut = {
            userId: userId,
            message: userMessage,
            symbol: userSymbol
        };
        console.log(`/UserMessage ${JSON.stringify(userMessageOut)}`)
        messages.push(userMessageOut);
        io.sockets.emit("broadcast");
    }
    messages = messages.slice(-8);
    res.send(messages);
});

server.listen(port, () => {
    console.log(`listening: http://localhost:${port}`);
});