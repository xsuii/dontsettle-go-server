// JavaScript Document
var wsUriTest = "ws://echo.websocket.org"
var wsuri = "ws://xsuii.meibu.net:8001/chat"
var origin = "http://xsuii.meibu.net"

var dbName = "history";
var tbName = "h" + localStorage.uid;
var db = window.openDatabase(dbName, "1.0", "history store", 1000);
var talks;
var f = document.getElementById("file");

function init() {
	onWebSocket();
}

function upFile() {
	console.log("file upload fire");
	var reader = new FileReader();
	reader.onloadend = function(bytes) {
		fName = f.value.substring(f.value.lastIndexOf('\\') + 1);
		console.log("send file:", fName);
		doSend(JSON.stringify({
			"Type": "F",
			"ToName": "0",
			"Body": fName
		}));
		doSend(bytes.target.result);
	};
	reader.readAsArrayBuffer(f.files[0]);
}

// database operate

function createTable(tx) {
	console.log("create table", tbName);
	tx.executeSql('CREATE TABLE IF NOT EXISTS ' + tbName + '(talks)');
}

function addHistory(tx) {
	console.log("add history", tbName);
	tx.executeSql('CREATE TABLE IF NOT EXISTS ' + tbName + '(talks)');
	tx.executeSql('INSERT INTO ' + tbName + ' VALUES("' + talks + '")');
}

function showHistoryDB(tx) {
	console.log("check history");
	tx.executeSql('SELECT * FROM ' + tbName, [], showHistorySuccess, errorCB);
}

function clearHistoryDB(tx) {
	console.log("clear history");
	tx.executeSql('DROP TABLE IF EXISTS ' + tbName);
	tx.executeSql('CREATE TABLE IF NOT EXISTS ' + tbName + '(talks)');
	document.getElementById("chatHistory").innerHTML = '';
}

function showHistorySuccess(tx, results) {
	var len = results.rows.length;
	var para = document.getElementById("chatHistory");
	if (len == 0) {
		para.innerHTML = "no history";
		return;
	}
	para.innerHTML = '';
	console.log(tbName + " table: " + len + " rows found.");
	for (var i = 0; i < len; i++) {
		console.log("Row = " + i + " history = " + results.rows.item(i).talks);
		var pre = document.createElement("p");
		pre.innerHTML = results.rows.item(i).talks;
		para.appendChild(pre);
		keepScrollButtom(para);
	}
}

function errorCB(tx, err) {
	alert("Error processing SQL: " + err);
}

function successCB() {
	console.log("database excute success!")
}

function showHistory() {
	db.transaction(showHistoryDB, errorCB);
}

function clearHistory() {
	db.transaction(clearHistoryDB, errorCB);
}

function chatError(err) {
	var para = document.getElementById("chatError");
	para.innerHTML = err;
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
	document.getElementById("author").innerHTML = "Author:" + localStorage.username;
	document.getElementById("uid").innerHTML = "UID:" + localStorage.uid;
	doSend(localStorage.uid + "+" + localStorage.username); // send uid as identify
	//db.transaction(createTable, errorCB, successCB);
}

function onClose(evt) {
	console.log("DISCONNECTED");
}

function onMessage(evt) {
	talks = evt.data
	console.log("RESPONSE: " + talks);

	var para = document.getElementById("messageBox");
	var pre = document.createElement("p");
	pre.innerHTML = talks;
	para.appendChild(pre);
	keepScrollButtom(para);

	db.transaction(addHistory, errorCB, successCB);
}

function onError(evt) {
	console.log("ERROR: " + evt.data);
}

function doSend(message) {
	console.log("SENT: " + message);
	websocket.send(message);
}

function sendMessage() {
	var msg = document.getElementById("message").value;
	var one = document.getElementById("one").value;
	var group = document.getElementById("group").value;
	if (one) {
		var para = document.getElementById("messageBox");
		var pre = document.createElement("p");
		pre.innerHTML = "[S]" + localStorage.username + ":" + msg;
		para.appendChild(pre);
		keepScrollButtom(para);
		//msg = "S+" + one + "+" + msg; // single chat
		msg = JSON.stringify({
			"Type": "S",
			"ToName": one,
			"Body": msg
		});
	} else if (group) {
		//msg = "G+" + group + "+" + msg; // group chat
		msg = JSON.stringify({
			"Type": "G",
			"ToName": group,
			"Body": msg
		});
	} else {
		//msg = "B+" + msg; // broadcast chat
		msg = JSON.stringify({
			"Type": "B",
			"ToName": "broadcast",
			"Body": msg
		});
	}
	doSend(msg);
}

function keepScrollButtom(node) {
	if (node.scrollHeight > node.clientHeight) {
		node.scrollTop = node.scrollHeight - node.clientHeight;
	}
}

window.addEventListener("load", init, false);