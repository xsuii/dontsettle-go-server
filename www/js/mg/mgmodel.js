function managerViewModel() {
    var self = this;
    self.identifyView = ko.observable(true);

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
        toOne: "",
        toGroup: ""
    });

    // request login
    self.login = function(login) {
        console.log("call login and send identify :", login);
        localStorage.username = login.lUserName; // store for later use, if not true then set it to null in app.js-onMessage
        doSend(JSON.stringify({ // websocket send
            "Username": login.lUserName,
            "Userpasswd": login.lUserPassWd
        }));
    };

    self.sendMessage = function(msg) {
        console.log("send message", msg);
        var t = new Date();
        var pack = {
            "Sender": localStorage.username,
            "DateTime": t.toUTCString(),
            "OpCode": OpChat,
            "Body": msg.sendMsg,
        };
        if (msg.toOne != "") {
            pack["DstT"] = "S";
            pack["Receiver"] = msg.toOne;
        } else if (msg.toGroup != "") {
            pack["DstT"] = "G";
            pack["Receiver"] = msg.toGroup;
        } else {
            pack["DstT"] = "B";
            pack["Receiver"] = "broadcast";
        }
        doSend(JSON.stringify(pack));
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
}

var mgModel = new managerViewModel;
ko.applyBindings(mgModel);

// for this hiding method is burdensome
// another solution is to use knockout's visible-binding, 
// to set current visiable and pre-view invisible