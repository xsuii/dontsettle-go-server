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
		doSend(JSON.stringify({
			"Sender": localStorage.username,
			"Receiver": sendTo,
			"Body": fName,
			"DateTime": t.toUTCString(),
			"OpCode": OpFileTransfer,
			"DstT": "S"
		}));
		console.log(bytes.target.result.toString());
		doSend(bytes.target.result);
	};
	reader.readAsArrayBuffer(f.files[0]);
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
			gameModel.addChats(msg);
			db.transaction(addHistory, errorCB, successCB);
			break;
		case OpFileTransfer:
			var para = document.getElementById("messageBox");
			var pre = document.createElement("p");
			pre.innerHTML = "[" + msg.DstT + "]" + msg.Sender + ":" + msg.Body;
			if (msg.Sender != "MASTER") {
				console.log("add file node")
				pre.style.fontStyle = "italic"; // file node
				pre.style.fontWeight = "bolder";
				pre.style.color = "#FF0087";
				pre.style.backgroundColor = "#D5D5D5"
				pre.style.padding = "5px";
				pre.setAttribute("Sender", msg.Sender);
				pre.setAttribute("Receiver", msg.Receiver);
				pre.setAttribute("filename", msg.Body);
				pre.setAttribute("datetime", msg.DateTime);
				pre.setAttribute("opcode", msg.OpCode);
				pre.setAttribute("dstt", msg.DstT);
				pre.onclick = function() { // send download file request
					console.log(this.getAttribute("filename"));
					var r = confirm("sure download?  " + this.getAttribute("filename"));
					if (r == true) {
						p = {
							"Sender": this.getAttribute("Sender"),
							"Receiver": this.getAttribute("Receiver"),
							"Body": this.getAttribute("filename"),
							"DateTime": this.getAttribute("dateTime"),
							"OpCode": parseInt(this.getAttribute("opcode")),
							"DstT": this.getAttribute("dstt")
						};
						doSend(JSON.stringify(p))
					} else {
						return;
					}
				}
			} else if (msg.Sender == "MASTER") { // recieve file
				file = JSON.parse(msg.Body);
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
				return;
			}
			para.appendChild(pre);
			keepScrollButtom(para);

			db.transaction(addHistory, errorCB, successCB);
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