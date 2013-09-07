        var wsUriTest = "ws://echo.websocket.org"
        var wsuri = "ws://xsuii.meibu.net:8001/login"
        var origin = "http://xsuii.meibu.net"
        var output;

        function init() {
            output = document.getElementById("output");
            testWebSocket();
        }

        function testWebSocket() {
            //websocket = new plugins.WebSocket(wsUriTest);
            websocket = new plugins.WebSocket(wsuri, "", origin);
            websocket.onopen = function(evt) { onOpen(evt) };
            websocket.onclose = function(evt) { onClose(evt) };
            websocket.onmessage = function(evt) { onMessage(evt) };
            websocket.onerror = function(evt) { onError(evt) };
        }

        function onOpen(evt) {
                writeToScreen("CONNECTED");
        }

        function onClose(evt) {
                writeToScreen("DISCONNECTED");
        }

        function onMessage(evt) {
                writeToScreen('<span style="color: blue;">RESPONSE: ' + evt + '</span>');
                var isIn = parseInt(evt);
                if(isIn) {
                    window.location.assign("chat.html");
					localStorage.uid = isIn;	// store uid for keep connection binding while page switch
                }else{
                    loginError("user name or userpassword error!");
                }
        }

        function onError(evt) {
                writeToScreen('<span style="color: red;">ERROR: ' + evt + '</span>');
        }

        function doSend(message) {
                writeToScreen("SENT: " + message);
                websocket.send(message);
        }

        function sendLogin() {
            var user = document.getElementById("username").value;
            var passwd = document.getElementById("password").value;
            doSend(user + "+" +passwd);
        }
		
		function loginError(err) {
			var para = document.getElementById("loginError");
			para.innerHTML = err;
		}

        function writeToScreen(message) {
                var pre = document.createElement("p");
                pre.style.wordWrap = "break-word";
                pre.innerHTML = message;
                output.appendChild(pre);
        }

        window.addEventListener("load", init, false);