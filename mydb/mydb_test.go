package mydb

import (
	"testing"
)

// test
var (
	DriverName     = "mysql"
	DataSourceName = "root:mrp520@/game"
)

func TestOpenDatabase(t *testing.T) {
	db := InitMydb(DriverName, DataSourceName)
	err := db.Open()
	if err != nil {
		t.Error("error open database", err.Error())
	}
}

func TestSelect(t *testing.T) {
	db := InitMydb(DriverName, DataSourceName)
	err := db.Open()
	if err != nil {
		t.Error("error open database", err.Error())
	}
	_, err = db.Select("username,email", "user")
	if err != nil {
		t.Error("error select operation", err.Error())
	}
}
