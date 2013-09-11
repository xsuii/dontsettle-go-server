/*
	common database package
	usage 	: 	give a .conf include driverName(e.g. "mysql"), dataSourceName(user:passwd@/dbname),
				than you can use database's common operation(CRUD).
	start	:	2013-9-10
*/

package mydb

import (
	"database/sql"
	_ "github.com/Go-SQL-Driver/MySQL"
	"sync"
)

var dbOperationMutex *sync.Mutex

type DatabaseInterface interface {
}

type DB struct {
	db             *sql.DB
	DriverName     string // mysql...
	DataSourceName string // user:passwd@/dbname (for more detail:see go-sql-driver's wiki in github)
}

func InitMydb(driverName string, dataSourceName string) *DB {
	d := &DB{
		db:             nil,
		DriverName:     driverName,
		DataSourceName: dataSourceName,
	}
	return d
}

func (d *DB) Open() error {
	var err error
	d.db, err = sql.Open(d.DriverName, d.DataSourceName)
	if err != nil {
		return err
	}
	return nil
}

// C(create)
// e.g.
// insert into <tb_user> values (vals, ...)
// insert into <tb_user>(elem, ...) values (vals, ...)
func (d *DB) Insert(tb string, params ...interface{}) (effect int) {
	dbOperationMutex.Lock()
	defer dbOperationMutex.Lock()
	return
}

// R(read)
// e.g.
// select < * | username, userpasswd > from <tb_user>; d.Select("username,userpassword,...","user")
// select < * | username, userpasswd > from <tb_user> where <expression>;
func (d *DB) Select(elems string, tb string) ([][]string, error) {
	//dbOperationMutex.Lock()
	//defer dbOperationMutex.Unlock()
	rows, err := d.db.Query("SELECT " + elems + " FROM " + tb)
	if err != nil {
		return nil, err
	}

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var results [][]string
	rawResult := make([][]byte, len(cols))
	elem := make([]interface{}, len(cols))
	for i, _ := range rawResult {
		elem[i] = &rawResult[i]
	}

	for rows.Next() {
		result := make([]string, len(cols))
		err = rows.Scan(elem...) // [Q:how to use []string as a "..." reciver in func's param]
		if err != nil {
			return nil, err
		}

		for i, raw := range rawResult {
			if raw == nil {
				result[i] = "\\N"
			} else {
				result[i] = string(raw)
			}
		}
		results = append(results, result)
	}
	return results, nil
}

// U(update)
// e.g.
//
func (d *DB) Update(tb string, params ...interface{}) {
	dbOperationMutex.Lock()
	defer dbOperationMutex.Unlock()
}

// D(delete)
// e.g.
// delete from <tb_user> where <expression>
func (d *DB) Delete(tb string, params ...interface{}) {
	dbOperationMutex.Lock()
	defer dbOperationMutex.Unlock()
}

/*

2013-9-11
 * Mutex didn't work, threw a "panic...: memory access error..."
*/
