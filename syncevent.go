package main

import (
	"database/sql"
	"fmt"
)

type SyncEvent struct {
	Id         int
	ErpEntryId int
	SyncDate   string
	Imported   int
	updated    int
	deleted    int
}

const(
	// EVENTS
	COUNT_EVENT_ALL       = "SELECT COUNT(*) FROM admin_erp_sync_events"
	COUNT_EVENT_BY_ENTRY  = "SELECT COUNT(*) FROM admin_erp_sync_events WHERE erpEntryId=?"
	SELECT_EVENT_ALL      = "SELECT id, erpEntry, syncDate, imported, updated, deleted FROM admin_erp_sync_events"
	SELECT_EVENT_BY_ID    = "SELECT id, erpEntry, syncDate, imported, updated, deleted FROM admin_erp_sync_events WHERE id=?"
	SELECT_EVENT_BY_ENTRY = "SELECT id, erpEntry, syncDate, imported, updated, deleted FROM admin_erp_sync_events WHERE erpEntryId=?"
	INSERT_EVENT          = "INSERT admin_erp_sync_events SET erpEntry=?, syncDate=?, imported=?, updated=?, deleted=?"
	UPDATE_EVENT_BY_ID    = "UPDATE admin_erp_sync_events SET erpEntry=?, syncDate=?, imported=?, updated=?, deleted=? WHERE id=?"
	DELETE_EVENT          = "DELETE FROM admin_erp_sync_events WHERE id=?"
)

func initDbSyncEvent(db *sql.DB) {
	defer fmt.Printf("Init DB DONE! \n")

	// TABLE FOR SYNC JOUNRNAL
	sql := "CREATE TABLE IF NOT EXISTS `mid_db`.`admin_sync_events` (`id` int(10) unsigned NOT NULL AUTO_INCREMENT, `erpEntryId` int(10) unsigned NOT NULL DEFAULT '0', `syncDate` bigint(20) unsigned DEFAULT NULL, `imported` int(10) unsigned NOT NULL DEFAULT '0', `updated` int(10) unsigned NOT NULL DEFAULT '0', `deleted` int(10) unsigned NOT NULL DEFAULT '0', PRIMARY KEY (`id`)) ENGINE=InnoDB DEFAULT CHARSET=latin1;"
	st, err := db.Prepare(sql)
	checkErr(err)
	_, err = st.Exec()
	checkErr(err)
}