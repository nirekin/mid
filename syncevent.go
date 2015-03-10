package main

import (
	"database/sql"
)

// SyncEvent is the processed content of one extraction from an ERP
type SyncEvent struct {
	Id         int
	ErpEntryId int
	SyncDate   string
	Imported   int64
	Updated    int64
	Deleted    int64
	PTime      int64
	NBEntries  int64
}

const (
	// EVENTS
	EVENT_SELECT_FIELDS = "SELECT id, erpEntryId, syncDate, imported, updated, deleted, pTime, nbEnries "
	EVENT_INSERT_UPDATE = "erpEntryId=?, syncDate=?, imported=?, updated=?, deleted=?, pTime=?, nbEnries=?"

	SELECT_EVENT_ALL      = EVENT_SELECT_FIELDS + " FROM admin_sync_events"
	SELECT_EVENT_BY_ID    = EVENT_SELECT_FIELDS + " FROM admin_sync_events WHERE id=?"
	SELECT_EVENT_BY_ENTRY = EVENT_SELECT_FIELDS + " FROM admin_sync_events WHERE erpEntryId=?"
	INSERT_EVENT          = "INSERT admin_sync_events SET " + EVENT_INSERT_UPDATE
	UPDATE_EVENT_BY_ID    = "UPDATE admin_sync_events SET " + EVENT_INSERT_UPDATE + " WHERE id=?"
	DELETE_EVENT_BY_ID    = "DELETE FROM admin_sync_events WHERE id=?"
)

// saveDb saves the SyncEvent into the db
func (o *SyncEvent) saveDb() error {
	st, err := dbC.Prepare(INSERT_EVENT)
	if err != nil {
		return err
	} else {
		defer st.Close()
	}

	res, err := st.Exec(o.ErpEntryId, getNowMillisecond(), o.Imported, o.Updated, o.Deleted, o.PTime, o.NBEntries)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	o.Id = int(id)
	if err != nil {
		return err
	}
	return nil
}

// loadDb loads the SyncEvent from the db,
// the id must be defined to identify the content to load.
func (o *SyncEvent) loadDb() error {
	st, err := dbC.Prepare(SELECT_EVENT_BY_ID)
	if err != nil {
		return err
	} else {
		defer st.Close()
	}

	rows, err := st.Query(o.Id)
	if err != nil {
		return err
	}

	for rows.Next() {
		o.loadFromDbRow(rows)
	}
	return nil
}

// deleteDb deletes the SyncEvent from the db,
// the id must be defined to identify the content to delete.
func (o *SyncEvent) deleteDb() error {
	st, err := dbC.Prepare(DELETE_EVENT_BY_ID)
	if err != nil {
		return err
	} else {
		defer st.Close()
	}
	_, err = st.Exec(o.Id)
	if err != nil {
		return err
	}
	return nil
}

// loadFromDbRow loads a SyncEvent from one sql.Rows
func (o *SyncEvent) loadFromDbRow(rows *sql.Rows) error {
	err := rows.Scan(&o.Id, &o.ErpEntryId, &o.SyncDate, &o.Imported, &o.Updated, &o.Deleted, &o.PTime, &o.NBEntries)
	if err != nil {
		return err
	}
	return nil
}

// initDbSyncEvent creates the table to store SyncEvents
func initDbSyncEvent(db *sql.DB) error {
	sql := "CREATE TABLE IF NOT EXISTS `mid_db`.`admin_sync_events` (`id` int(10) unsigned NOT NULL AUTO_INCREMENT, `erpEntryId` int(10) unsigned NOT NULL DEFAULT '0', `syncDate` bigint(20) unsigned DEFAULT NULL, `imported` bigint(20) unsigned NOT NULL DEFAULT '0', `updated` bigint(20) unsigned NOT NULL DEFAULT '0', `deleted` bigint(20) unsigned NOT NULL DEFAULT '0', `pTime` bigint(20) unsigned NOT NULL DEFAULT '0', `nbEnries` bigint(20) unsigned NOT NULL DEFAULT '0', PRIMARY KEY (`id`)) ENGINE=InnoDB DEFAULT CHARSET=latin1;"
	st, err := db.Prepare(sql)
	if err != nil {
		return err
	} else {
		defer st.Close()
	}

	_, err = st.Exec()
	if err != nil {
		return err
	}
	return nil
}

func addEvent(entry *ErpEntry, imported int64, updated int64, deleted int64, pTime int64, nbEntries int64) error {
	o := &SyncEvent{ErpEntryId: entry.Id, Imported: imported, Updated: updated, Deleted: deleted, PTime: pTime, NBEntries: nbEntries}
	return o.saveDb()
}
