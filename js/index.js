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
const symbols = ["🔴", "🟡", "🟢", "🟣", "🔵", "🟠", "🟤", "⚪️"];
var messages = [];

io.on("connection", (socket) => {
    console.log(`connection: socket=${socket.id}`);
    socket.on('disconnect', () => {
        console.log(`disconnect: socket=${socket.id}`);
    });
});

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
    if (!req.body.userId || !req.cookies.userId) {
        req.body.userId = uuid.v4();
    } else {
        req.body.userId == req.cookies.userId;
    }
    res.cookie("userId", req.body.userId);
    console.log("post: /UserMessage " + JSON.stringify(req.body));
    if (users.indexOf(req.body.userId) === -1) {
        users.push(req.body.userId);
    }
    if (req.body.message?.length > 0) {
        const timespan = (((new Date()).getTime()) - req.cookies.userDate);
        const cooldown = req.cookies.userDate ? timespan >= 1000 : true;
        if (!cooldown) {
            const errMsg = `fail: no cooldown because previous message was ${timespan}ms ago`;
            console.log(errMsg);
            res.status(403).send(errMsg);
            return;
        }
        const unique = req.cookies.userMessage !== req.body.message;
        if (!unique) {
            const errMsg = `fail: not unique because '${req.cookies.userMessage}' === '${req.body.message}'`;
            console.log(errMsg);
            res.status(403).send(errMsg);
            return;
        }
        if (req.body.message.toLowerCase().indexOf("/n") >= 0) {
            res.cookie("userName", req.body.message.substring(2).trim());
            res.send("");
            return;
        }
        res.cookie("userMessage", req.body.message);
        res.cookie("userDate", (new Date()).getTime());
        var i = users.indexOf(req.body.userId);
        req.body.symbol = req.cookies.userName || symbols[i % symbols.length];
        messages.push(req.body);
        io.sockets.emit("broadcast");
    }
    messages = messages.slice(-8);
    res.send(messages);
});

server.listen(port, () => {
    console.log(`listening: http://localhost:${port}`);
});