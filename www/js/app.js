// app
window.requestFileSystem = window.requestFileSystem || window.webkitRequestFileSystem;

// app opcode
var OpNull = 0
var OpMaster = 1 // this present master's message, include bad-package...
var OpLogin = 2
var OpRegister = 3
var OpChat = 4
var OpFileTransfer = 5
var OpFileUp = 6
var OpFileDown = 7
// special id
var NullId = 0
var MasterId = 10000 // app server's system id, it may be good to reserve some id
var BroadCastId = 10001
// forward type
var FwGroup = 1
var FwSingle = 2
var FwBroadcast = 3
var FWT = {};
FWT[FwSingle] = "single";
FWT[FwGroup] = "group";
FWT[FwBroadcast] = "broadcast";

var _userId_;
var _userName_;

var fs = null;
var wsuri = "ws://xsuii.meibu.net:8001/login";
var wsbk = "ws://localhost:8001/login";
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
		console.log("file content:", bytes);
		var p = new Pack(sendTo, fName, OpFileUp, FwSingle)
		p.send();
		console.log(bytes.target.result.toString());
		doSend(bytes.target.result);
	};
	reader.readAsArrayBuffer(f.files[0]);
}

function addFileNode(file) {
	console.log("add file node.");
	var para = document.getElementById("messageBox");
	var pre = document.createElement("p");
	pre.innerHTML = "[" + FWT[file.ForwardType] + "]" + file.Sender + ":" + file.Body;
	pre.style.fontStyle = "italic"; // file node
	pre.style.fontWeight = "bolder";
	pre.style.color = "#FF0087";
	pre.style.backgroundColor = "#D5D5D5"
	pre.style.padding = "5px";
	pre.setAttribute("sender", file.Sender);
	pre.setAttribute("receiver", file.Receiver);
	pre.setAttribute("filename", file.Body);
	pre.setAttribute("timestamp", file.TimeStamp);
	pre.onclick = function() { // send download file request
		var r = confirm("sure download?  " + this.getAttribute("filename"));
		if (r == true) {
			var body = {
				"FSender": Number(this.getAttribute("sender")),
				"FReceiver": Number(this.getAttribute("receiver")),
				"FileName": this.getAttribute("filename"),
				"TimeStamp": Number(this.getAttribute("timestamp"))
			};
			console.log("file ticket : ", body);
			var p = new Pack(
				MasterId,
				JSON.stringify(body),
				OpFileDown,
				FwSingle);
			p.send();
		} else {
			return;
		}
	}
	para.appendChild(pre);
	keepScrollButtom(para);

	db.transaction(addHistory, errorCB, successCB);
}

function receiveFile(msg) {
	console.log("Receive file")
	file = UnPack(msg.Body);
	//file = JSON.parse(msg.Body);
	console.log("After unpack file:", file);
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

			var blob = new Blob([file.Body], {	// should handle different type of file, png, jpg ...
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
	websocket = new WebSocket(wsbk);
	console.log(websocket)
	websocket.onopen = function(evt) {
		console.log("CONNECTED:", evt)
	};

	websocket.onclose = function(evt) {
		console.log("DISCONNECTED:", evt)
	};

	websocket.onmessage = function(evt) {
		onMessage(evt)
	};

	websocket.onerror = function(evt) {
		console.log("ERROR:", evt)
	};
}

function onMessage(evt) {
	console.log("OnMessage:", evt)
	msg = UnPack(evt.data);
	console.log("After Unpack Message:", msg);
	switch (msg.OpCode) {
		case OpMaster:
			console.log(msg);
			break;
		case OpLogin:
			if (msg.Body != "0") {
				console.log("login success with uid :", msg.Body);
				// initial user data
				_userId_ = parseInt(msg.Body);
				tbName = "h" + msg.Body; // database table name begins with "h"(history)
				gameModel.toChatView();
			} else {
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
			break;
		case OpFileUp:
			addFileNode(msg);
			break;
		case OpFileDown:
			receiveFile(msg);
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

function Pack(reciever, body, opcode, forwardtype) {
	self = this;
	self.Sender = Number(_userId_);
	self.Receiver = Number(reciever);
	self.Body = window.btoa(body); // base64 encode; [bug:chinese unsurport]
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

// JSON -> struct -> 

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