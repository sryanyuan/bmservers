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
	repairDB(*repairPath + "/users.db")
	log.Println("Repair db done")
}

func repairDB(path string) {
	//	Connect db
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		log.Println("Can't open db file.", err)
		return
	}
	defer db.Close()

	//	alter player_rank
	_, err = db.Exec("ALTER TABLE player_rank ADD COLUMN server_id integer NOT NULL DEFAULT 0")
	if nil != err {
		log.Println("SQL:", err)
		return
	}
	//	update server_id
	_, err = db.Exec("UPDATE player_rank SET server_id = 1 WHERE server_id = 0")
	if nil != err {
		log.Println("SQL:", err)
		return
	}
}
