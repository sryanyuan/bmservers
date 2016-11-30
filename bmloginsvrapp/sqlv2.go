package main

import (
	"database/sql"
	"strconv"

	"os"

	"github.com/axgle/mahonia"
	"github.com/cihub/seelog"
)

func initDatabaseUserV2(path string) *sql.DB {
	if !pathExist(path) {
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
		CREATE TABLE IF NOT EXISTS useraccount (uid integer primary key, account varchar(20) UNIQUE, password varchar(20), name0 varchar(20), name1 varchar(20), name2 varchar(20), online bool, mail VARCHAR(64))
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
			CREATE TABLE IF NOT EXISTS player_rank (id integer primary key, uid integer, name varchar(21), job integer, level integer, expr integer, power integer)
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

	//	check web_user
	sqlexpr = `
		CREATE TABLE IF NOT EXISTS web_user (uid integer primary key, user_name VARCHAR(20) UNIQUE, password VARCHAR(64), permission INTEGER, mail VARCHAR(64))
	`
	_, err = db.Exec(sqlexpr)
	if err != nil {
		seelog.Error("Failed to create new table,err:", err)
		db.Close()
		return nil
	}

	//	check role_name
	sqlexpr = `
		CREATE TABLE IF NOT EXISTS online_player (id integer primary key, uid integer, server_id integer, lid integer)
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

func dbGetUserAccountInfoByName(db *sql.DB, name string, info *ExportUserAccountInfo) error {
	row := db.QueryRow("SELECT uid, account, password, mail FROM useraccount WHERE name0 = ? or name1 = ? or name2 = ?",
		name, name, name)

	return row.Scan(&info.Uid, &info.Account, &info.Password, &info.Mail)
}

func dbIsUserRankExistsV2(db *sql.DB, name string) bool {
	rows, err := db.Query("SELECT COUNT(*) AS CNT FROM player_rank WHERE name = ?",
		name)
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

func dbRemoveUserRankInfoV2(db *sql.DB, name string) bool {
	if !dbIsUserRankExistsV2(db, name) {
		return true
	}

	_, err := db.Exec("DELETE FROM player_rank WHERE name = ?",
		name)
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

	if !dbIsUserRankExistsV2(db, info.Name) {
		//	new record
		_, err := db.Exec("INSERT INTO player_rank VALUES(null, ?, ?, ?, ?, ?, ?)",
			info.Uid, info.Name, info.Job, info.Level, info.Expr, info.Power)
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
		expr += " WHERE name='" + info.Name + "'"

		_, err := db.Exec(expr)
		if err != nil {
			seelog.Errorf("Error on executing expression[%s] Error[%s]",
				expr, err.Error())
			return false
		}

		return true
	}
}

func dbGetUserRankInfoOrderByPowerV2(db *sql.DB, limit int, job int) []UserRankInfo {
	var ret []UserRankInfo = nil

	expr := "SELECT uid, name, job, level, expr, power FROM player_rank "
	if job >= 0 &&
		job <= 2 {
		expr += " WHERE job = " + strconv.Itoa(job)
	}

	expr += " ORDER BY power DESC, level DESC LIMIT " + strconv.Itoa(limit)
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

func dbGetUserRankInfoOrderByLevelV2(db *sql.DB, limit int, job int) []UserRankInfo {
	var ret []UserRankInfo = nil

	expr := "SELECT uid, name, job, level, expr, power FROM player_rank "
	if job >= 0 &&
		job <= 2 {
		expr += " WHERE job = " + strconv.Itoa(job)
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

func dbRemoveAllOnlinePlayer(db *sql.DB) error {
	_, err := db.Exec("DELETE FROM online_player")
	return err
}

func dbRemoveOnlinePlayerByServerId(db *sql.DB, serverId int) error {
	_, err := db.Exec("DELETE FROM online_player WHERE server_id = ?", serverId)
	return err
}

func dbRemoveOnlinePlayerByUID(db *sql.DB, uid uint32, serverId int, lid int32) error {
	_, err := db.Exec("DELETE FROM online_player WHERE uid = ? AND server_id = ? AND lid = ?", uid, serverId, lid)
	return err
}

func dbAddOnlinePlayer(db *sql.DB, uid uint32, serverId int, lid int32) error {
	_, err := db.Exec("INSERT INTO online_player values (null, ?, ?, ?)", uid, serverId, lid)
	return err
}

func dbIsPlayerOnline(db *sql.DB, uid uint32, serverId int) bool {
	row := db.QueryRow("SELECT COUNT(*) FROM online_player WHERE uid = ? AND server_id = ?", uid, serverId)
	var count int
	err := row.Scan(&count)
	if err != nil {
		if err == sql.ErrNoRows {
			return false
		}
	}

	return true
}

func dbGetPlayerOnlineServerId(db *sql.DB, uid uint32) int {
	row := db.QueryRow("SELECT server_id FROM online_player WHERE uid = ?", uid)
	var serverId int
	err := row.Scan(&serverId)
	if err != nil {
		return 0
	}
	return serverId
}
