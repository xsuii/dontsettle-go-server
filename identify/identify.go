/*
 * project login & register module
 *
 */

package identify

import (
	"code.google.com/p/go.net/websocket"
	"database/sql"
	_ "github.com/Go-SQL-Driver/MySQL"
	"log"
	"strconv"
	"strings"
)

// login message
var ()

func Login(ws *websocket.Conn) {
	log.Println(" # USER LOGIN # ")
	log.Println("client :", ws.Request().RemoteAddr)
	var reply string
	var effect int
	var uid int
	var username string

	db, err := sql.Open("mysql", "root:mrp520@/game") // connect database
	checkErr(err)
	log.Println("mysql connect success . . .")
	defer func() {
		err = db.Close()
		checkErr(err)
		log.Println("close database . . .")
	}()

	for { // keep until login success
		// get login imformation from client
		err = websocket.Message.Receive(ws, &reply)
		checkErr(err)

		temp := strings.Split(reply, "+")

		log.Println("Receive login message : [ Username:", temp[0], " ]  [ Password:", temp[1], " ]")

		stmt, err := db.Prepare("select UID, username, userpassword from user where username=? && userpassword=?")
		checkErr(err)

		rows, err := stmt.Query(temp[0], temp[1]) // temp contants username and password which split before
		checkErr(err)

		for rows.Next() {
			var userpassword string
			effect++

			err = rows.Scan(&uid, &username, &userpassword)
			checkErr(err)

			log.Println("MySQL : [ UID:", uid, " ]  [ Username:", username, " ]  [ Password:", userpassword, " ]")
		}

		if effect > 0 {
			log.Println(uid, "login success . . .")
			t := strconv.Itoa(uid)
			websocket.Message.Send(ws, t+"+"+username)
			return
		} else {
			websocket.Message.Send(ws, "0")
			log.Println("login fail . . .")
		}
	}
}

func Register(ws *websocket.Conn) {
	log.Println(" # USER REGISTER #")
	log.Println("client :", ws.Request().RemoteAddr)
	var reply string

	db, err := sql.Open("mysql", "root:mrp520@/game") // connect database
	checkErr(err)
	log.Println("mysql connect success . . .")
	defer func() {
		err = db.Close()
		checkErr(err)
		log.Println("close database . . .")
	}()

	for {
		// get register imformations
		err = websocket.Message.Receive(ws, &reply)
		checkErr(err)

		temp := strings.Split(reply, "+")

		log.Println("Receive register message : [ Username:", temp[0], " ]  [ email:", temp[1], " ]  [ Password:", temp[2], " ]")

		// apply register to database
		stmt, err := db.Prepare("INSERT user SET username=?, email=?, userpassword=?")
		checkErr(err)

		_, err = stmt.Exec(temp[0], temp[1], temp[2])
		checkErr(err)

		if err != nil {
			websocket.Message.Send(ws, "REGISTER_FAIL")
		} else {
			websocket.Message.Send(ws, "REGISTER_SUCCESS")
			return
		}
	}
}

func checkErr(err error) {
	if err != nil {
		log.Println("Error:", err.Error())
	}
}
