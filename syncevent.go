package main

import (
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