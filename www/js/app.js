/*
 
 */
// Html5 filesystem 
window.requestFileSystem = window.requestFileSystem || window.webkitRequestFileSystem;

// app opcode
OpNull = 0
OpMaster = 1 // this present master's message, include bad-package...
OpLogin = 2
OpRegister = 3
OpChat = 4
OpFileTransfer = 5
OpFileUpld = 6
OpFileDownld = 7
OpFileUpldReq = 8
OpFileDownldReq = 9
OpChatToOne = 10
OpChatToMuti = 11
OpChatBroadcast = 12
OpFileUpldReqAckOk = 13
OpError = 14
OpFileTicket = 15
OpFileDownldReqAckOk = 16
OpFileUpldDone = 17

// error code
ErrFileUpReqAck = 1
ErrOperation = 2
ErrBadPackage = 3

// special id
var NullId = 0
var MasterId = 10000 // app server's system id, it may be good to reserve some id
var BroadCastId = 10001

var SEQ_LENGTH = 5;

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
var pack; // messages recieve from server
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
	var err = '';
	switch (e.code) {
		case FileError.QUOTA_EXCEEDED_ERR:
			err = 'QUOTA_EXCEEDED_ERR';
			break;
		case FileError.NOT_FOUND_ERR:
			err = 'NOT_FOUND_ERR';
			break;
		case FileError.SECURITY_ERR:
			err = 'SECURITY_ERR';
			break;
		case FileError.INVALID_MODIFICATION_ERR:
			err = 'INVALID_MODIFICATION_ERR';
			break;
		case FileError.INVALID_STATE_ERR:
			err = 'INVALID_STATE_ERR';
			break;
		default:
			err = 'UNKNOWN';
	}
	document.getElementById("chatError").innerHTML = 'Error: ' + err;
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

function upldFileReq() {
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
	var p = new Package(sendTo, JSON.stringify(fileInfo), OpFileUpldReq) // send file title
	p.send();
}

function addFileTask(file) {
	console.log("add file task.");
	var jTicket = JSON.parse(file.Body);
	FileTask[jTicket.TaskId] = jTicket; // add task
	console.log("Show file task:", FileTask);
	var para = document.getElementById("messageBox");
	var pre = document.createElement("p");
	pre.innerHTML = "[" + FWT[OpChatToOne] + "]" + file.Sender + ":" + jTicket.FileInfo.FileName;
	pre.style.fontStyle = "italic"; // file node
	pre.style.fontWeight = "bolder";
	pre.style.color = "#FF0087";
	pre.style.backgroundColor = "#D5D5D5";
	pre.style.padding = "5px";
	pre.setAttribute("fileTicket", file.Body);
	pre.onclick = function() { // send download file request
		var r = confirm("sure download? " + jTicket.FileInfo.FileName + jTicket.FileInfo.FileSize);
		if (r == true) {
			var p = new Package(
				MasterId,
				jTicket.TaskId,
				OpFileDownldReq);
			p.send();
		} else {
			return;
		}
	}
	para.appendChild(pre);
	keepScrollButtom(para);

	db.transaction(addHistory, errorCB, successCB);
}


// [TODO] Actually this should stop sending when 
// the network problem or wrong sending pointing by
// server or client itself . And most important 
// thing is this operation is under control. It should
// not happen.

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
		console.log("show head 100:", content.slice(0, 100));

		for (var i = 0; i < Math.floor(file.size / SEQ_LENGTH); i++) {
			var fileSequence = {
				TaskId: taskId,
				SeqNum: i,
				SeqContent: content.slice(i * SEQ_LENGTH, (i + 1) * SEQ_LENGTH),
				SeqSize: SEQ_LENGTH
			};
			console.log("Show Seq:", fileSequence);
			var p = new Package(sendTo, JSON.stringify(fileSequence), OpFileUpld) // send file title
			setTimeout((function(pkg) {
				return function() {
					console.log("Show Pack:", pkg);
					console.log("Show Seq:", window.atob(pkg.Body));
					pkg.send();
				}
			})(p), i * 1000);
		}
		if (file.size % SEQ_LENGTH > 0) {
			var fileSequence = {
				TaskId: taskId,
				SeqNum: i,
				SeqContent: content.slice(i * SEQ_LENGTH, i * SEQ_LENGTH + file.size % SEQ_LENGTH),
				SeqSize: file.size % SEQ_LENGTH
			};
			console.log("Show Seq:", fileSequence);
			var p = new Package(sendTo, JSON.stringify(fileSequence), OpFileUpld) // send file title
			setTimeout((function(pkg) {
				return function() {
					console.log("Show pack:", window.atob(pkg.Body));
					pkg.send();
				}
			})(p), i * 1000);
		};
		// download end
		var p = new Package(sendTo, taskId, OpFileUpldDone)
		setTimeout((function(pkg) {
			return function() {
				console.log("Show pack:", window.atob(pkg.Body));
				pkg.send();
			}
		})(p), (i + 1) * 1000);
	};
	reader.readAsText(file);
	console.log("upload file end.")
}

// [TODO] check whether file exsist, if is, then throw a message asking changing

function createFile(taskId) {
	console.log("Start create file(name&taskid):", FileTask[taskId].FileInfo.FileName, taskId);
	fs.root.getFile(FileTask[taskId].FileInfo.FileName, {
		create: true
	}, function(fileEntry) {
		fileEntry.createWriter(function(fileWriter) {
			fileWriter.truncate(FileTask[taskId].FileInfo.FileSize);
			fileWriter.onwriteend = function(e) {
				console.log("Create file done.");
				var p = new Package(
					MasterId,
					taskId,
					OpFileDownld);
				p.send();
			};

			fileWriter.onerror = function(e) {
				console.log("Create file error.");
			};
		}, errorFile);
	}, errorFile);
}

// Use file taskId(the FileTask map's key,UUID) to decide which file to create/write
// write asynchronous

function writeFile(jFileSeq) {
	var fileSeq = JSON.parse(jFileSeq);
	var fileTask = FileTask[fileSeq.TaskId];
	console.log("Write to file:", fileTask.FileInfo.FileName);
	console.log("Show file sequence:", fileSeq);
	fs.root.getFile(fileTask.FileInfo.FileName, {
		create: false
	}, function(fileEntry) {
		console.log("Create writer.");
		fileEntry.createWriter(function(fileWriter) {
			var off = fileSeq.SeqNum * SEQ_LENGTH;
			console.log("Writing Seq:", fileSeq)
			console.log("Seek to", off);
			fileWriter.seek(fileSeq.SeqNum * SEQ_LENGTH);
			fileWriter.onwriteend = function(e) {
				console.log("write file done");
			};

			fileWriter.onerror = function(e) {
				console.log("write file error");
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
	console.log("add history {" + pack.Body + "} to :" + tbName);
	tx.executeSql('CREATE TABLE IF NOT EXISTS ' + tbName + '(talks)');
	tx.executeSql('INSERT INTO ' + tbName + ' VALUES("' + pack.Body + '")');
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
	pack = UnPack(evt.data);
	console.log("After Unpack Message:", pack);
	switch (pack.OpCode) {
		case OpMaster:
			console.log(pack);
			break;
		case OpLogin:
			if (pack.Body != "0") {
				console.log("login success with uid :", pack.Body);
				// initial user data
				_userId_ = parseInt(pack.Body);
				tbName = "h" + pack.Body; // database table name begins with "h"(history)
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
			gameModel.addChats(pack); // show to scroll
			db.transaction(addHistory, errorCB, successCB);
			break;
		case OpFileTransfer:
			break;
		case OpFileTicket:
			addFileTask(pack);
			break;
		case OpFileUpldReqAckOk:
			console.log("Upload file ready.");
			uploadFileInPiece(pack.Body);
			break;
		case OpFileDownldReqAckOk:
			console.log("Recieve download ACK.")
			createFile(pack.Body);
			break;
		case OpFileDownld:
			writeFile(pack.Body);
			break;
		case OpError:
			console.log("Get error message.")
			errorHandler(pack.Body);
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
		case ErrBadPackage:
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
	this.Sender = Number(_userId_);
	this.Reciever = Number(reciever);
	this.Body = window.btoa(body); // base64 encode; [bug:chinese unsurport]
	this.TimeStamp = Math.round(Date.now() / 1000); // Unix timestamp
	this.OpCode = opcode;

	this.send = function() {
		console.log("send package");
		doSend(JSON.stringify({
			"Sender": self.Sender,
			"Reciever": Number(reciever),
			"Body": window.btoa(body),
			"TimeStamp": self.TimeStamp, // [TODO]this would be a bug.
			"OpCode": opcode,
		}));
	}
}

// JSON -> struct -> 

function UnPack(pack) {
	var p = JSON.parse(pack);
	p.Body = window.atob(p.Body); // base64 decode(should consider client-end surport)
	return p
}

function loginError(str) {
	console.log("[Login Error]", str);
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