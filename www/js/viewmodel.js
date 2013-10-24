function chats(who, msg) {
    var self = this;
    self.who = who;
    self.msg = msg;
}

function chatHistory(msg) {
    var self = this;
    self.msg = msg;
}

function gameViewModel() {
    var self = this;
    self.identifyView = ko.observable(true);
    self.chatView = ko.observable(false);

    self.loginData = ko.observable({
        lUserName: "",
        lUserPassWd: "",
        loginError: ""
    });

    self.registerData = ko.observable({
        rUserName: "",
        rEmail: "",
        rUserPassWd: "",
        rConfirmPassWd: ""
    });

    self.msgBox = ko.observableArray([]);
    self.historyBox = ko.observableArray([]);

    self.userInfo = ko.observable({
        author: "",
        uid: ""
    });

    self.sendBox = ko.observable({
        sendMsg: "",
        toOne: "100002",
        toGroup: ""
    });

    // request login
    self.login = function(login) {
        console.log("call login and send identify :", login);
        _userName_ = login.lUserName; // store for later use, if not true then set it to null in app.js-onMessage
        doSend(JSON.stringify({ // websocket send
            "Username": login.lUserName,
            "Userpasswd": login.lUserPassWd
        }));
    };

    self.sendMessage = function(msg) {
        console.log("send message", msg);
        if (msg.toOne != "") {
            opcode = OpChatToOne;
            rc = msg.toOne;
        } else if (msg.toGroup != "") {
            opcode = OpChatToMuti;
            rc = msg.toGroup;
        } else {
            opcode = OpChatBroadcast;
            rc = BroadCastId;
        }
        console.log("rc type:", typeof(rc))
        var p = new Package(rc, msg.sendMsg, opcode);
        console.log("Message package:", p);
        p.send();
    };

    self.addChats = function(chat) {
        self.msgBox.push(new chats(chat.Sender, chat.Body));
    };

    self.showHistory = function(history) {
        console.log("show history:", history);
        getHistory();
    };

    self.clearHistory = function() {
        dropHistory();
    };

    // hash anchor definition 
    self.toLoginView = function() {
        location.hash = "login";
    };

    self.toRegisterView = function() {
        location.hash = "register";
    };

    self.toChatView = function() {
        location.hash = "chat";
    };

    // initial view
    self.registerData(null); // disable register view
}

var gameModel = new gameViewModel;
ko.applyBindings(gameModel);

// for this hiding method is burdensome
// another solution is to use knockout's visible-binding, 
// to set current visiable and pre-view invisible