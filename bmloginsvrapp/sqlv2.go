package main

import (
	"database/sql"
	"strconv"

	"os"

	"github.com/axgle/mahonia"
	"github.com/cihub/seelog"
)

func initDatabaseUserV2(path string) *sql.DB {
	if !PathExist(path) {
		file, err := os.Create(path)
		if err != nil {
			seelog.Error("Can't create db file.", err)
			return nil
		} else {
			file.Close()
		}
	}

	//	Connect db
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		seelog.Error("Can't open db file.", err)
		return nil
	}

	sqlexpr := `
		CREATE TABLE IF NOT EXISTS useraccount (uid integer primary key, account varchar(20), password varchar(20), name0 varchar(20), name1 varchar(20), name2 varchar(20), online bool)
		`
	_, err = db.Exec(sqlexpr)
	if err != nil {
		seelog.Error("Create new table failed.Error:", err)
		db.Close()
		return nil
	}

	//	check user donate table
	sqlexpr = `
		CREATE TABLE IF NOT EXISTS userdonate (uid integer primary key, donate integer, lastdonatetime integer, expiretime integer)
		`
	_, err = db.Exec(sqlexpr)
	if err != nil {
		seelog.Error("Create new table failed.Error:", err)
		db.Close()
		return nil
	}

	//	check user donate history table
	sqlexpr = `
			CREATE TABLE IF NOT EXISTS userdonatehistory (id integer primary key, uid integer, donate integer, donatetime integer, donateorderid varchar(50))
			`
	_, err = db.Exec(sqlexpr)
	if err != nil {
		seelog.Error("Create new table failed.Error:", err)
		db.Close()
		return nil
	}

	//	check systemgift table
	sqlexpr = `
			CREATE TABLE IF NOT EXISTS systemgift (id integer primary key, uid integer, giftid integer, giftsum integer, givetime integer, expiretime integer)
			`
	_, err = db.Exec(sqlexpr)
	if err != nil {
		seelog.Error("Create new table failed.Error:", err)
		db.Close()
		return nil
	}

	//	check user donate consume
	sqlexpr = `
			CREATE TABLE IF NOT EXISTS userdonateconsume (id integer primary key, uid integer, name varchar(21), itemid integer, cost integer, buytime integer)
			`
	_, err = db.Exec(sqlexpr)
	if err != nil {
		seelog.Error("Failed to create new table,err:", err)
		db.Close()
		return nil
	}

	//	check player rank
	sqlexpr = `
			CREATE TABLE IF NOT EXISTS player_rank (id integer primary key, uid integer, server_id integer, name varchar(21), job integer, level integer, expr integer, power integer)
			`
	_, err = db.Exec(sqlexpr)
	if err != nil {
		seelog.Error("Failed to create new table,err:", err)
		db.Close()
		return nil
	}

	//	check admin_account
	sqlexpr = `
			CREATE TABLE IF NOT EXISTS admin_account (id integer primary key, account varchar(20), level integer)
			`
	_, err = db.Exec(sqlexpr)
	if err != nil {
		seelog.Error("Failed to create new table,err:", err)
		db.Close()
		return nil
	}

	//	check donate_request
	sqlexpr = `
			CREATE TABLE IF NOT EXISTS donate_request (id integer primary key, name varchar(20), uid integer, orderid varchar(50), money integer, note varchar(40), result integer)
			`
	_, err = db.Exec(sqlexpr)
	if err != nil {
		seelog.Error("Failed to create new table,err:", err)
		db.Close()
		return nil
	}

	//	check role_name
	sqlexpr = `
		CREATE TABLE IF NOT EXISTS role_name (id integer primary key, uid integer, server_id integer, name0 varchar(20), name1 varchar(20), name2 varchar(20))
	`
	_, err = db.Exec(sqlexpr)
	if err != nil {
		seelog.Error("Failed to create new table,err:", err)
		db.Close()
		return nil
	}

	return db
}

func dbUserNameExistV2(db *sql.DB, serverId int, name string) bool {
	row := db.QueryRow("SELECT COUNT(*) FROM role_name WHERE server_id = ? AND (name0 = ? or name1 = ? or name2 = ?)")
	var count int
	err := row.Scan(&count)
	if nil != err {
		seelog.Error("SQL : ", err)
		return true
	}

	if count == 0 {
		return false
	}
	return true
}

func dbRemoveUserNameV2(db *sql.DB, uid uint32, serverId int, name string) bool {
	exists := dbUserNameExistV2(db, serverId, name)
	if exists {
		return false
	}

	ret, err := db.Exec("UPDATE role_name SET name0 = ? WHERE server_id = ? AND name0 = ? AND uid = ?",
		name,
		serverId,
		"",
		uid)
	if nil != err {
		seelog.Error("SQL :", err)
		return false
	}
	affected, _ := ret.RowsAffected()
	if affected == 1 {
		return true
	}

	ret, err = db.Exec("UPDATE role_name SET name1 = ? WHERE server_id = ? AND name1 = ? AND uid = ?",
		name,
		serverId,
		"",
		uid)
	if nil != err {
		seelog.Error("SQL :", err)
		return false
	}
	affected, _ = ret.RowsAffected()
	if affected == 1 {
		return true
	}

	ret, err = db.Exec("UPDATE role_name SET name2 = ? WHERE server_id = ? AND name2 = ? AND uid = ?",
		name,
		serverId,
		"",
		uid)
	if nil != err {
		seelog.Error("SQL :", err)
		return false
	}
	affected, _ = ret.RowsAffected()
	if affected == 1 {
		return true
	}

	return false
}

func dbAddUserNameV2(db *sql.DB, uid uint32, serverId int, name string) bool {
	exists := dbUserNameExistV2(db, serverId, name)
	if exists {
		return false
	}

	//	server account exists ?
	row := db.QueryRow("SELECT COUNT(*) from role_from WHERE server_id = ? AND uid = ?", serverId, uid)
	var count int
	if err := row.Scan(&count); nil != err {
		seelog.Error("SQL :", err)
		return false
	}
	if 0 == count {
		//	need insert record
		_, err := db.Exec("INSERT INTO role_name values (null, ?, ?, ?, ?, ?)",
			uid, serverId, "", "", "")
		if nil != err {
			seelog.Error("SQL :", err)
			return false
		}
	}

	ret, err := db.Exec("UPDATE role_name SET name0 = ? WHERE server_id = ? AND name0 = ? AND uid = ?",
		name,
		serverId,
		"",
		uid)
	if nil != err {
		seelog.Error("SQL :", err)
		return false
	}
	affected, _ := ret.RowsAffected()
	if affected == 1 {
		return true
	}

	ret, err = db.Exec("UPDATE role_name SET name1 = ? WHERE server_id = ? AND name1 = ? AND uid = ?",
		name,
		serverId,
		"",
		uid)
	if nil != err {
		seelog.Error("SQL :", err)
		return false
	}
	affected, _ = ret.RowsAffected()
	if affected == 1 {
		return true
	}

	ret, err = db.Exec("UPDATE role_name SET name2 = ? WHERE server_id = ? AND name2 = ? AND uid = ?",
		name,
		serverId,
		"",
		uid)
	if nil != err {
		seelog.Error("SQL :", err)
		return false
	}
	affected, _ = ret.RowsAffected()
	if affected == 1 {
		return true
	}

	return false
}

func dbIsUserRankExistsV2(db *sql.DB, serverId int, name string) bool {
	rows, err := db.Query("SELECT COUNT(*) AS CNT FROM player_rank WHERE name = ? AND server_id = ?",
		name, serverId)
	if err != nil {
		seelog.Errorf("Error on selecting uid,error[%s]", err.Error())
		return true
	}

	defer rows.Close()
	if rows.Next() {
		count := 0
		rows.Scan(&count)

		if count == 0 {
			return false
		}
		return true
	}
	return false
}

func dbRemoveUserRankInfoV2(db *sql.DB, serverId int, name string) bool {
	if !dbIsUserRankExistsV2(db, serverId, name) {
		return true
	}

	_, err := db.Exec("DELETE FROM player_rank WHERE name = ? AND server_id = ?",
		name, serverId)
	if err != nil {
		seelog.Errorf("Error on executing expression Error[%s]",
			err.Error())
		return false
	}

	return true
}

func dbUpdateUserRankInfoV2(db *sql.DB, info *UserRankInfo) bool {
	if nil == db {
		return false
	}

	if !dbIsUserRankExistsV2(db, info.ServerId, info.Name) {
		//	new record
		_, err := db.Exec("INSERT INTO player_rank VALUES(null, ?, ?, ?, ?, ?, ?, ?)",
			info.Uid, info.ServerId, info.Name, info.Job, info.Level, info.Expr, info.Power)
		if err != nil {
			seelog.Error("db exec failed.", " err:", err)
			return false
		}

		return true
	} else {
		//	update record
		expr := "UPDATE player_rank SET "
		exprInitialLength := len(expr)

		if 0 == info.Level &&
			0 == info.Expr &&
			0 == info.Power {
			return true
		}

		if 0 != info.Level {
			expr += " level = " + strconv.Itoa(info.Level)
		}
		if 0 != info.Expr {
			if len(expr) != exprInitialLength {
				expr += " , "
			}
			expr += " expr = " + strconv.Itoa(info.Expr)
		}
		if 0 != info.Power {
			if len(expr) != exprInitialLength {
				expr += " , "
			}
			expr += " power = " + strconv.Itoa(info.Power)
		}
		expr += " WHERE name='" + info.Name + "'" + " AND server_id = " + strconv.Itoa(info.ServerId)

		_, err := db.Exec(expr)
		if err != nil {
			seelog.Errorf("Error on executing expression[%s] Error[%s]",
				expr, err.Error())
			return false
		}

		return true
	}
}

func dbGetUserRankInfoOrderByPowerV2(db *sql.DB, serverId int, limit int, job int) []UserRankInfo {
	var ret []UserRankInfo = nil

	expr := "SELECT uid, name, job, level, expr, power FROM player_rank "
	if job >= 0 &&
		job <= 2 {
		expr += " WHERE job = " + strconv.Itoa(job) + " AND server_id = " + strconv.Itoa(serverId)
	}

	expr += " ORDER BY power DEC, level DESC LIMIT " + strconv.Itoa(limit)
	rows, err := db.Query(expr)
	if err != nil {
		seelog.Errorf("Error on executing expression[%s] Error[%s]",
			expr, err.Error())
		return nil
	}

	defer rows.Close()
	ret = make([]UserRankInfo, limit)
	index := 0
	//	Read data
	for rows.Next() {
		rows.Scan(&ret[index].Uid, &ret[index].Name, &ret[index].Job, &ret[index].Level, &ret[index].Expr, &ret[index].Power)
		//	gbk转utf8
		decoder := mahonia.NewDecoder("gb18030")
		u8Str := decoder.ConvertString(ret[index].Name)
		ret[index].Name = u8Str
		index++
	}

	if index == 0 {
		return nil
	}

	return ret[0:index]
}

func dbGetUserRankInfoOrderByLevelV2(db *sql.DB, serverId int, limit int, job int) []UserRankInfo {
	var ret []UserRankInfo = nil

	expr := "SELECT uid, name, job, level, expr, power FROM player_rank "
	if job >= 0 &&
		job <= 2 {
		expr += " WHERE job = " + strconv.Itoa(job) + " AND server_id = " + strconv.Itoa(serverId)
	}

	expr += " ORDER BY level DESC, power DESC LIMIT " + strconv.Itoa(limit)
	rows, err := db.Query(expr)
	if err != nil {
		seelog.Errorf("Error on executing expression[%s] Error[%s]",
			expr, err.Error())
		return nil
	}

	defer rows.Close()
	ret = make([]UserRankInfo, limit)
	index := 0
	//	Read data
	for rows.Next() {
		rows.Scan(&ret[index].Uid, &ret[index].Name, &ret[index].Job, &ret[index].Level, &ret[index].Expr, &ret[index].Power)
		//	gbk转utf8
		decoder := mahonia.NewDecoder("gb18030")
		u8Str := decoder.ConvertString(ret[index].Name)
		ret[index].Name = u8Str
		index++
	}

	if index == 0 {
		return nil
	}

	return ret[:index]
}
