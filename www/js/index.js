// JavaScript Document
var wsUriTest = "ws://echo.websocket.org"
var wsuri = "ws://xsuii.meibu.net:8001/login"
var origin = "http://xsuii.meibu.net"

function init() {
	output = document.getElementById("output");
	onWebSocket();
}

function onWebSocket() {
	//websocket = new plugins.WebSocket(wsUriTest);
	websocket = new WebSocket(wsuri);
	websocket.onopen = function(evt) {
		onOpen(evt)
	};
	websocket.onclose = function(evt) {
		onClose(evt)
	};
	websocket.onmessage = function(evt) {
		onMessage(evt)
	};
	websocket.onerror = function(evt) {
		onError(evt)
	};
}

function onOpen(evt) {
	console.log("CONNECTED");
}

function onClose(evt) {
	console.log("DISCONNECTED");
}

function onMessage(evt) {
	console.log("RESPONSE: " + evt.data);

	// if login success, recieve string "0"; otherwise recieve "uid+username" which will store later
	if (evt.data != "0") {
		localStorage.uid = evt.data;
		localStorage.username = user;
		window.location.assign("chat.html");
	} else {
		loginError("user name or userpassword error!");
	}
}

function onError(evt) {
	console.log("ERROR: " + evt.data);
}

function doSend(message) {
	console.log("SENT: " + message);
	websocket.send(message);
}

function sendLogin() {
	user = document.getElementById("username").value;
	var passwd = document.getElementById("password").value;
	//doSend(user + "+" + passwd);
	doSend( JSON.stringify(
		{ "Username":user,
		  "Userpasswd":passwd }
	));
}

function loginError(err) {
	var para = document.getElementById("loginError");
	para.innerHTML = err;
}

window.addEventListener("load", init, false);