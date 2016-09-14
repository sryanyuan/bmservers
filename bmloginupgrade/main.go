package main

import (
	"database/sql"

	"flag"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	repairPath := flag.String("repair-path", "", "-repair-path <your file path>")
	flag.Parse()

	if len(*repairPath) == 0 {
		log.Println("Invalid repair-path")
		return
	}

	log.Println("Repairing db ...")
	repairDB(*repairPath)
	log.Println("Repair db done")
}

func repairDB(path string) {
	//	Connect db
	dbUser, err := sql.Open("sqlite3", path+"/users.db")
	if err != nil {
		log.Println("Can't open db file.", err)
		return
	}
	defer dbUser.Close()

	//	alter player_rank
	_, err = dbUser.Exec("ALTER TABLE player_rank ADD COLUMN server_id integer NOT NULL DEFAULT 0")
	if nil != err {
		log.Println("SQL:", err)
	}
	//	update server_id
	_, err = dbUser.Exec("UPDATE player_rank SET server_id = 1 WHERE server_id = 0")
	if nil != err {
		log.Println("SQL:", err)
	}
	//	alter useraccount
	_, err = dbUser.Exec("ALTER TABLE useraccount ADD COLUMN mail VARCHAR(64) NOT NULL DEFAULT ''")
	if nil != err {
		log.Println("SQL:", err)
	}
	//	update useraccount mail
	dbReq, err := sql.Open("sqlite3", path+"/req.db")
	if err != nil {
		log.Println(err)
		return
	}
	defer dbReq.Close()

	rows, err := dbReq.Query("SELECT mail, account FROM userregkey")
	if nil != err {
		log.Println(err)
		return
	}
	defer rows.Close()

	var mail string
	var account string
	for rows.Next() {
		err = rows.Scan(&mail, &account)
		if nil != err {
			log.Println(err)
			return
		}

		dbUser.Exec("UPDATE useraccount SET mail = ? WHERE account = ?", mail, account)
	}
}
