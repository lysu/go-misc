package main

import (
	"database/sql"
	"github.com/lysu/beanstalk"
	"time"
)

func main() {
	var conn, err = beanstalk.Dial("tcp", "127.0.0.1:11300")
	if err != nil {
		panic(err)
	}
	// 1. Send data into queue first but in 'infine' delay
	id, err := conn.Put([]byte("msg"), 1, 100000*time.Hour, 1*time.Minute)
	if err != nil {
		panic(err)
	}
	// 2. if pre-send bk success
	db, err := sql.Open("", "")
	if err != nil {
		panic(err)
	}
	// 3. start db transaction
	tx, err := db.Begin()
	if err != nil {
		panic(err)
	}
	// 4. do biz log
	result := dbop(tx)
	if result {
		// 5.1. commit if biz success
		err = tx.Commit()
		if err == nil {
			// 5.2. kick beanstalk job
			err := conn.KickJob(id)
			if err != nil {
				// 5.3. write file log and wait re-tick for task-idempotent
				WriteLogAndRetryInInterval(id, "wait-kick")
			}
		} else {
			// ...
		}
	} else {
		// 6.1. rollback if biz failure
		err = tx.Rollback()
		if err == nil {
			// 6.2. delete beanstalk job
			err := conn.Delete(id)
			if err != nil {
				// 6.3. write file log and wait re-delete-idempotent
				WriteLogAndRetryInInterval(id, "wait-delete")
			}
		} else {
			// ...
		}
	}

}

func WriteLogAndRetryInInterval(id uint64, typ string) error {
	return nil
}

func dbop(db *sql.Tx) bool {
	return true
}
