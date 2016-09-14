package main

import (
	"database/sql"
)

/*
WebUser
*/
type WebUser struct {
	Uid        uint32
	UserName   string
	Password   string
	Permission uint32
}

func dbWebUserGet(db *sql.DB, username string) (*WebUser, error) {
	var user WebUser
	row := db.QueryRow("SELECT uid, password, permission FROM web_user WHERE user_name = ?", username)
	err := row.Scan(&user.Uid, &user.Password, &user.Permission)
	if nil != err {
		if err == sql.ErrNoRows {
			user.Permission = kPermission_Guest
			return &user, nil
		}

		return nil, err
	}

	user.UserName = username
	return &user, nil
}

func dbWebUserGetByUid(db *sql.DB, uid uint32) (*WebUser, error) {
	var user WebUser
	user.Uid = uid
	row := db.QueryRow("SELECT user_name, password, permission FROM web_user WHERE uid = ?", uid)
	err := row.Scan(&user.UserName, &user.Password, &user.Permission)
	if nil != err {
		if err == sql.ErrNoRows {
			user.Permission = kPermission_Guest
			return &user, nil
		}

		return nil, err
	}

	return &user, nil
}
