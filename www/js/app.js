// app
window.requestFileSystem = window.requestFileSystem || window.webkitRequestFileSystem;

// app opcode
var OpNull = 0
var OpMaster = 1 // this present master's message, include bad-package...
var OpLogin = 2
var OpRegister = 3
var OpChat = 4
var OpFileTransfer = 5
// special id
var NullId = 0
var MasterId = 10000 // app server's system id, it may be good to reserve some id
var BroadCastId = 10001
// forward type
var FwGroup = 1
var FwSingle = 2
var FwBroadcast = 3

var fs = null;
var wsuri = "ws://xsuii.meibu.net:8001/login";
var wsbk = "ws://172.18.19.46:8001/login";
var dbName = "dontsettle";
var tbName;
var msg; // messages recieve from server
var db = window.openDatabase(dbName, "1.0", "history store", 1000);

function init() {
	if (window.requestFileSystem) {
		initFS();
	}
	onWebSocket();
}

//////////////////////////////　File /////////////////////////////

function errorFile(e) {
	var msg = '';
	switch (e.code) {
		case FileError.QUOTA_EXCEEDED_ERR:
			msg = 'QUOTA_EXCEEDED_ERR';
			break;
		case FileError.NOT_FOUND_ERR:
			msg = 'NOT_FOUND_ERR';
			break;
		case FileError.SECURITY_ERR:
			msg = 'SECURITY_ERR';
			break;
		case FileError.INVALID_MODIFICATION_ERR:
			msg = 'INVALID_MODIFICATION_ERR';
			break;
		case FileError.INVALID_STATE_ERR:
			msg = 'INVALID_STATE_ERR';
			break;
	}
	document.getElementById("chatError").innerHTML = 'Error: ' + msg;
}

function initFS() {
	console.log("init filesystem")
	window.requestFileSystem(window.TEMPORARY, 1024 * 1024, function(filesystem) {
		fs = filesystem;
		console.log(fs)
	}, errorFile);
}

function upFile() {
	console.log("file upload fire");
	var f = document.getElementById("file");
	var sendTo = document.getElementById("one").value;
	if (sendTo == null || sendTo == "") {
		document.getElementById("chatError").innerHTML = "Please fill up the one you send to";
		return;
	}
	var reader = new FileReader();
	var t = new Date();
	reader.onloadend = function(bytes) {
		fName = f.value.substring(f.value.lastIndexOf('\\') + 1);
		console.log("send file:", fName);
		var p = new Pack(localStorage.uid, sendTo, fName, OpFileTransfer, FwSingle)
		p.send();
		console.log(bytes.target.result.toString());
		doSend(bytes.target.result);
	};
	reader.readAsArrayBuffer(f.files[0]);
}

function addFileNode(file) {
	var para = document.getElementById("messageBox");
	var pre = document.createElement("p");
	pre.innerHTML = "[" + file.ForwardType + "]" + file.Sender + ":" + file.Body;
	console.log("add file node")
	pre.style.fontStyle = "italic"; // file node
	pre.style.fontWeight = "bolder";
	pre.style.color = "#FF0087";
	pre.style.backgroundColor = "#D5D5D5"
	pre.style.padding = "5px";
	pre.setAttribute("Sender", file.Sender);
	pre.setAttribute("Receiver", file.Receiver);
	pre.setAttribute("filename", file.Body);
	pre.setAttribute("datetime", file.DateTime);
	pre.setAttribute("opcode", file.OpCode);
	pre.setAttribute("forwardtype", file.ForwardType);
	pre.onclick = function() { // send download file request
		console.log(this.getAttribute("filename"));
		var r = confirm("sure download?  " + this.getAttribute("filename"));
		if (r == true) {
			/*p = {
				"Sender": this.getAttribute("Sender"),
				"Receiver": this.getAttribute("Receiver"),
				"Body": this.getAttribute("filename"),
				"DateTime": this.getAttribute("dateTime"),
				"OpCode": parseInt(this.getAttribute("opcode")),
				"ForwardType": this.getAttribute("forwardtype")
			};
			doSend(JSON.stringify(p))*/
			var p = new Pack(this.getAttribute("Sender"),
				this.getAttribute("Receiver"),
				this.getAttribute("filename"),
				this.getAttribute("dateTime"),
				parseInt(this.getAttribute("opcode")),
				this.getAttribute("forwardtype"));
			p.send();
		} else {
			return;
		}
	}
	para.appendChild(pre);
	keepScrollButtom(para);

	db.transaction(addHistory, errorCB, successCB);
}

function recieveFile(msg) {
	file = UnPack(msg.Body);
	console.log(fs, "begin to recieve file", file.FileName);
	fs.root.getFile(file.FileName, {
		create: true
	}, function(fileEntry) {
		fileEntry.createWriter(function(fileWriter) {
			console.log("file_writer_test_s");
			fileWriter.onwriteend = function(e) {
				console.log("get file done");
			};

			fileWriter.onerror = function(e) {
				console.log("write error");
			};

			var blob = new Blob([file.Body], {
				type: "text/plain"
			});
			fileWriter.write(blob);
		}, errorFile);
	}, errorFile);
	console.log("file_writer_test_e");
}

////////////////////// database operate  ///////////////////////

function createTable(tx) {
	console.log("create table", tbName);
	tx.executeSql('CREATE TABLE IF NOT EXISTS ' + tbName + '(talks)');
}

function addHistory(tx) {
	console.log("add history {" + msg.Body + "} to :" + tbName);
	tx.executeSql('CREATE TABLE IF NOT EXISTS ' + tbName + '(talks)');
	tx.executeSql('INSERT INTO ' + tbName + ' VALUES("' + msg.Body + '")');
}

function getHistoryDB(tx) {
	console.log("check history");
	tx.executeSql('SELECT * FROM ' + tbName, [], getHistorySuccess, errorCB);
}

function dropHistoryDB(tx) {
	console.log("clear history");
	tx.executeSql('DROP TABLE IF EXISTS ' + tbName);
	tx.executeSql('CREATE TABLE IF NOT EXISTS ' + tbName + '(talks)');
	gameModel.historyBox.removeAll();
}

function getHistorySuccess(tx, results) {
	var len = results.rows.length;
	var para = document.getElementById("chatHistory");
	if (len == 0) {

		return;
	}
	para.innerHTML = '';
	console.log(tbName + " table: " + len + " rows found.");
	for (var i = 0; i < len; i++) {
		console.log("Row = " + i + " history = " + results.rows.item(i).talks);
		gameModel.historyBox.push(new chatHistory(results.rows.item(i).talks));
	}
}

function errorCB(tx, err) {
	alert("Error processing SQL: " + err);
}

function successCB() {
	console.log("database excute success!")
}

function getHistory() {
	db.transaction(getHistoryDB, errorCB);
}

function dropHistory() {
	db.transaction(dropHistoryDB, errorCB);
}

///////////////////////////// websocket ///////////////////////////

function onWebSocket() {
	websocket = new WebSocket(wsuri);
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
	console.log("OnMessage:", evt)
	console.log("RESPONSE: " + evt.data);
	msg = UnPack(evt.data);
	console.log("after unpack :", msg);
	console.log("OpCode :" + msg.OpCode);
	switch (msg.OpCode) {
		case OpMaster:
			console.log(msg);
			break;
		case OpLogin:
			if (msg.Body != "0") {
				console.log("login success with uid :", msg.Body);
				// initial user data
				localStorage.uid = parseInt(msg.Body);
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
			gameModel.addChats(msg); // show to scroll
			db.transaction(addHistory, errorCB, successCB);
			break;
		case OpFileTransfer:
			if (msg.Sender != MasterId) {
				addFileNode(msg)
			} else if (msg.Sender == MasterId) { // recieve file
				recieveFile(msg)
			}
			break;
	}
	// if login success, recieve string "0"; otherwise recieve "uid+username" which will store later
}

/////////////////////////////////////////////////////////////////////

function doSend(message) {
	console.log("SENT: " + message);
	websocket.send(message);
}

// pack message(Object)

function Pack(sender, reciever, body, opcode, forwardtype) {
	self = this;
	self.Sender = sender;
	self.Receiver = reciever;
	self.Body = window.btoa(body);
	self.TimeStamp = Math.round(Date.now() / 1000); // Unix timestamp
	self.OpCode = opcode;
	self.ForwardType = forwardtype;

	self.send = function() {
		console.log("send package");
		doSend(JSON.stringify({
			"Sender": self.Sender,
			"Receiver": self.Receiver,
			"Body": self.Body,
			"TimeStamp": self.TimeStamp,
			"OpCode": self.OpCode,
			"ForwardType": self.ForwardType,
		}));
	}
}

function UnPack(pack) {
	var p = JSON.parse(pack);
	p.Body = window.atob(p.Body); // base64 decode(should consider client-end surport)
	return p
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