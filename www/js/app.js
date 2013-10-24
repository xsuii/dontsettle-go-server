/*
 
 */
// Html5 filesystem 
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
var OpFileUpReq = 8
var OpFileDownReq = 9
var OpChatToOne = 10
var OpChatToMuti = 11
var OpChatBroadcast = 12
var OpFileUpReqAckOk = 13
var OpError = 14

// error code
var ErrFileUpReqAck = 1

// special id
var NullId = 0
var MasterId = 10000 // app server's system id, it may be good to reserve some id
var BroadCastId = 10001

var SEQ_LENGTH = 10;

var FWT = {};
FWT[OpChatToOne] = "single";
FWT[OpChatToMuti] = "group";
FWT[OpChatBroadcast] = "broadcast";

var _userId_;
var _userName_;

var fs = null;
var wsuri = "ws://xsuii.meibu.net:8001/login";
var wsbk = "ws://localhost:8001/login";
var dbName = "dontsettle";
var tbName;
var msg; // messages recieve from server
var db = window.openDatabase(dbName, "1.0", "history store", 1000);

// A map to own file task as a file manager.{ key:UUID, value:filename }
var FileTask = {};

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

function ab2str(buf) {
	return String.fromCharCode.apply(null, new Uint16Array(buf));
}

function str2ab(str) {
	var buf = new ArrayBuffer(str.length * 2); // 2 bytes for each char
	var bufView = new Uint16Array(buf);
	for (var i = 0, strLen = str.length; i < strLen; i++) {
		bufView[i] = str.charCodeAt(i);
	}
	return buf;
}

function upFileReq() {
	console.log("Send file upload request.");
	var f = document.getElementById("file");
	var sendTo = document.getElementById("one").value;
	var file = f.files[0];
	var fName = file.name;
	if (sendTo == null || sendTo == "") {
		document.getElementById("chatError").innerHTML = "Please fill up the one you send to";
		return;
	}

	var fileInfo = {
		"FileName": fName,
		"FileSize": file.size
	};
	console.log("file info:", fileInfo);
	var p = new Package(sendTo, JSON.stringify(fileInfo), OpFileUpReq) // send file title
	p.send();
}

function uploadFileInPiece(taskId) {
	console.log("Start sending file in pieces.");
	var f = document.getElementById("file");
	var sendTo = document.getElementById("one").value;
	var file = f.files[0];
	var fName = file.name;
	if (sendTo == null || sendTo == "") {
		document.getElementById("chatError").innerHTML = "Please fill up the one you send to";
		return;
	}

	var reader = new FileReader();
	reader.onloadend = function(evt) {
		var content = evt.target.result;
		console.log(content.slice(0, 10));

		var p = new Package(sendTo, "", OpFileUp) // send file title
		var fileSequence = {
			TaskId: taskId
		};
		console.log("show head 100:", content.slice(0, 100));

		// <=1024; >1024 && size%1024==0; >1024 && size%1024>0
		if (file.size < SEQ_LENGTH) {
			fileSequence["SeqNum"] = 0;
			fileSequence["SeqContent"] = content.slice(0, file.size);
			fileSequence["SeqSize"] = file.size;
			console.log(fileSequence);
			p.Body = window.btoa(JSON.stringify(fileSequence));
			p.send();
		} else {
			for (var i = 0; i < file.size / SEQ_LENGTH - 1; i++) {
				fileSequence["SeqNum"] = i;
				fileSequence["SeqContent"] = content.slice(i * SEQ_LENGTH, (i + 1) * SEQ_LENGTH);
				fileSequence["SeqSize"] = SEQ_LENGTH;
				console.log(fileSequence);
				p.Body = window.btoa(JSON.stringify(fileSequence));
				p.send();
			}
			if (file.size % SEQ_LENGTH > 0) {
				fileSequence["SeqNum"] = i;
				fileSequence["SeqContent"] = content.slice(i * SEQ_LENGTH, i * SEQ_LENGTH + file.size % SEQ_LENGTH);
				fileSequence["SeqSize"] = file.size % SEQ_LENGTH;
				console.log(fileSequence);
				p.Body = window.btoa(JSON.stringify(fileSequence));
				p.send();
			};
		}
	};
	reader.readAsText(file);
	console.log("upload file end.")
}

function addFileNode(file) {
	console.log("add file node.");
	jTicket = JSON.parse(file.Body);
	FileTask[jTicket.TaskId] = jTicket.FileInfo.FileName; // add task
	console.log("Show file task:", FileTask)
	var para = document.getElementById("messageBox");
	var pre = document.createElement("p");
	pre.innerHTML = "[" + FWT[OpChatToOne] + "]" + file.Sender + ":" + jTicket.FileInfo.FileName;
	pre.style.fontStyle = "italic"; // file node
	pre.style.fontWeight = "bolder";
	pre.style.color = "#FF0087";
	pre.style.backgroundColor = "#D5D5D5"
	pre.style.padding = "5px";
	pre.setAttribute("fileTicket", file.Body);
	pre.onclick = function() { // send download file request
		var body = this.getAttribute("fileTicket");
		var r = confirm("sure download? " + jTicket.FileInfo.FileName + jTicket.FileInfo.FileSize);
		if (r == true) {
			console.log("Show file ticket : ", body);
			var p = new Package(
				MasterId,
				body,
				OpFileDownReq);
			p.send();
		} else {
			return;
		}
	}
	para.appendChild(pre);
	keepScrollButtom(para);

	db.transaction(addHistory, errorCB, successCB);
}

// Use file taskId(the FileTask map's key,UUID) to decide which file to create/write

function receiveFile(msg) {
	fileSeq = JSON.parse(msg.Body);
	fileName = FileTask[fileSeq.TaskId];
	console.log("Receive file:", fileName);
	console.log("Show file sequence:", fileSeq);
	fs.root.getFile(fileName, {
		create: true
	}, function(fileEntry) {
		fileEntry.createWriter(function(fileWriter) {
			fileWriter.seek(fileWriter.length);
			fileWriter.onwriteend = function(e) {
				console.log("write file done");
			};

			fileWriter.onerror = function(e) {
				console.log("write error");
			};

			var blob = new Blob([fileSeq.SeqContent], { // should handle different type of file, png, jpg ...
				type: "text/plain"
			});
			fileWriter.write(blob);
		}, errorFile);
	}, errorFile);
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
	console.log("Error processing SQL: " + err);
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
				console.log("create history table : ", tbName);
				gameModel.toChatView();
			} else {
				loginError("user name or userpassword error!");
			}
			break;
		case OpRegister:
			break;
		case OpChatToOne:
		case OpChatToMuti:
		case OpChatBroadcast:
			gameModel.addChats(msg); // show to scroll
			db.transaction(addHistory, errorCB, successCB);
			break;
		case OpFileTransfer:
			break;
		case OpFileUpReq:
			addFileNode(msg);
			break;
		case OpFileUpReqAckOk:
			console.log("Upload file ready.");
			uploadFileInPiece(msg.Body);
			break;
		case OpFileDown:
			receiveFile(msg);
			break;
		case OpError:
			console.log("Get error message.")
			errorHandler(msg.Body);
			break;
		default:
			console.log("Unknown message receive.");
	}
	// if login success, recieve string "0"; otherwise recieve "uid+username" which will store later
}

// handle the error response from server.

function errorHandler(err) {
	console.log(err);
	err = JSON.parse(err);
	switch (err.Code) {
		case ErrFileUpReqAck:
			alert("[Error:" + err.Code + "] " + err.Message);
			break;
		default:
			console.log("Unknown error.")
	}
}

/////////////////////////////////////////////////////////////////////

function doSend(message) {
	console.log("SENT: " + message);
	websocket.send(message);
}

// pack message(Object)

function Package(reciever, body, opcode) {
	self = this;
	self.Sender = Number(_userId_);
	self.Reciever = Number(reciever);
	self.Body = window.btoa(body); // base64 encode; [bug:chinese unsurport]
	self.TimeStamp = Math.round(Date.now() / 1000); // Unix timestamp
	self.OpCode = opcode;

	self.send = function() {
		console.log("send package");
		doSend(JSON.stringify({
			"Sender": self.Sender,
			"Reciever": self.Reciever,
			"Body": self.Body,
			"TimeStamp": self.TimeStamp,
			"OpCode": self.OpCode,
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