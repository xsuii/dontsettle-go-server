// app
window.requestFileSystem = window.requestFileSystem || window.webkitRequestFileSystem;

// app opcode
var OpMaster = 0
var OpLogin = 1
var OpRegister = 2
var OpChat = 3
var OpFileTransfer = 4

var MASTER = 10000  // app server's system id, it may be good to reserve some id

var fs = null;
var wsuri = "ws://xsuii.meibu.net:8001/mlogin";
var wsbk = "ws://172.18.19.46:8001/mlogin";

function init() {
	onWebSocket();
}

///////////////////////////// websocket ///////////////////////////

function onWebSocket() {
	websocket = new WebSocket(wsbk);
	console.log(websocket)
	websocket.onopen = function(evt) {
		console.log("CONNECTED")
	};

	websocket.onclose = function(evt) {
		console.log("DISCONNECTED")
	};

	websocket.onmessage = function(evt) {
		onMessage(evt)
	};

	websocket.onerror = function(evt) {
		console.log(evt)
		console.log(evt.data)
	};
}

function onMessage(evt) {
	console.log("RESPONSE: " + evt.data);
	msg = JSON.parse(evt.data)
	console.log("OpCode:" + msg.OpCode);
	console.log("check body type:", typeof(msg.Body));
	switch (msg.OpCode) {
		case OpMaster:
			console.log(msg);
			break;
		case OpLogin:
			if (msg.Body != "0") {
				console.log("login success with uid :", msg.Body);
				// initial user data
				localStorage.uid = msg.Body;
				tbName = "h" + msg.Body; // database table name begins with "h"(history)
				gameModel.toChatView();
			} else {
				localStorage.username = ""; //
				loginError("user name or userpassword error!");
			}
			break;
		case OpRegister:
			break;
		case OpChat:
			break;
	}
	// if login success, recieve string "0"; otherwise recieve "uid+username" which will store later
}

/////////////////////////////////////////////////////////////////////

function doSend(message) {
	console.log("SENT: " + message);
	websocket.send(message);
}

function loginError(err) {

}

function chatError(err) {
	var para = document.getElementById("chatError");
	para.innerHTML = err;
}

function keepScrollButtom(node) {
	if (node.scrollHeight > node.clientHeight) {
		node.scrollTop = node.scrollHeight - node.clientHeight;
	}
}

///////////////////////////////////////////////////////////////////

window.addEventListener("load", init, false);