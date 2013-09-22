/*
    A router using sammyjs for browser's anchor registering.
*/
var currentView = gameModel.identifyView;
console.log("currentView", currentView());

Sammy(function() {
    this.get('#login', function() {
        switchViewTo(gameModel.identifyView);
        gameModel.registerData(null); // disable register page
        gameModel.loginData({
            lUserName: "",
            lUserPassWd: "",
            loginError: ""
        });
    });

    this.get('#register', function() { // seems like it can't just use '#:register'
        gameModel.loginData(null); // disable login page
        gameModel.registerData({
            rUserName: "",
            rEmail: "",
            rUserPassWd: "",
            rConfirmPassWd: ""
        });
    });

    this.get('#chat', function() {
        console.log("go chat");
        switchViewTo(gameModel.chatView);
        gameModel.userInfo({
            author: localStorage.username,
            uid: localStorage.uid
        });
    });

    this.get('', function() { //  means that the empty client-side URL will be treated the same as #login, i.e., it will load and display the Inbox.
        this.app.runRoute('get', '#login')
    });
}).run();

function switchViewTo(view) {
    currentView(false);
    view(true);
    currentView = view;
}