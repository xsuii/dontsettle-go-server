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

type LoginInfo struct {
	Username   string
	Userpasswd string
}

func Login(ws *websocket.Conn) {
	log.Println(" # USER LOGIN # ")
	log.Println("client :", ws.Request().RemoteAddr)
	//var a string
	var login LoginInfo
	var effect int
	var uid int
	var username string

	db, err := sql.Open("mysql", "root:mrp520@/game") // connect database
	log.Println("open database")
	if err != nil {
		log.Println("Error:", err.Error())
	}
	log.Println("mysql connect success . . .")
	defer func() {
		err = db.Close()
		if err != nil {
			log.Println("Error:", err.Error())
		}
		log.Println("close database . . .")
	}()

	for { // keep until login success
		// get login imformation from client
		err = websocket.JSON.Receive(ws, &login)
		if err != nil {
			log.Println("Error:", err.Error())
		}
		log.Println("Receive login message : [ Username:", login.Username, " ]  [ Password:", login.Userpasswd, " ]")

		stmt, err := db.Prepare("select UID, username, userpassword from user where username=? && userpassword=?")
		if err != nil {
			log.Println("Error:", err.Error())
		}

		rows, err := stmt.Query(login.Username, login.Userpasswd) // temp contants username and password which split before
		if err != nil {
			log.Println("Error:", err.Error())
		}

		for rows.Next() {
			var userpassword string
			effect++

			err = rows.Scan(&uid, &username, &userpassword)
			if err != nil {
				log.Println("Error:", err.Error())
			}

			log.Println("MySQL : [ UID:", uid, " ]  [ Username:", username, " ]  [ Password:", userpassword, " ]")
		}

		if effect > 0 {
			log.Println(uid, "(uid) login success.")
			t := strconv.Itoa(uid)
			websocket.Message.Send(ws, t)
			return
		} else {
			websocket.Message.Send(ws, "0")
			log.Println("login fail . . .")
		}
	}
}

// [later:JSON,logic,]
func Register(ws *websocket.Conn) {
	log.Println(" # USER REGISTER #")
	log.Println("client :", ws.Request().RemoteAddr)
	var reply string

	db, err := sql.Open("mysql", "root:mrp520@/game") // connect database
	if err != nil {
		log.Println("Error:", err.Error())
	}
	log.Println("mysql connect success . . .")
	defer func() {
		err = db.Close()
		if err != nil {
			log.Println("Error:", err.Error())
		}
		log.Println("close database . . .")
	}()

	for {
		// get register imformations
		err = websocket.JSON.Receive(ws, &reply)
		if err != nil {
			log.Println("Error:", err.Error())
		}

		temp := strings.Split(reply, "+")

		log.Println("Receive register message : [ Username:", temp[0], " ]  [ email:", temp[1], " ]  [ Password:", temp[2], " ]")

		// apply register to database
		stmt, err := db.Prepare("INSERT user SET username=?, email=?, userpassword=?")
		if err != nil {
			log.Println("Error:", err.Error())
		}

		_, err = stmt.Exec(temp[0], temp[1], temp[2])
		if err != nil {
			log.Println("Error:", err.Error())
		}

		if err != nil {
			websocket.JSON.Send(ws, "REGISTER_FAIL")
		} else {
			websocket.JSON.Send(ws, "REGISTER_SUCCESS")
			return
		}
	}
}

func checkErr(err error) {
	if err != nil {
		log.Println("Error:", err.Error())
	}
}
