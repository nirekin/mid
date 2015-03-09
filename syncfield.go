package main

import (
	"database/sql"
	"time"
	"fmt"
)

type VisibleSyncField struct {
	CreationDate string
	FieldName    string
	JsonName     string
	ErpPk        bool
	Decoratos    []*VisibleDecorator
}

type SyncField struct {
	CreationDate         string
	FieldName            string
	NbDecorator          int
	JsonName             string
	ErpPk                bool
	Id                   int
	ErpEntryId           int
	Decorators           []Decorator
	PredefinedDecorators []*PredefinedDecorator
	TestContent          string
	TestResponse         string
}

const(
	// FIELD
	COUNT_FIELD_BY_ENTRY  = "SELECT COUNT(*) FROM admin_sync_field WHERE erpEntryId=?"
	COUNT_FIELD_USED      = "SELECT COUNT(*) FROM admin_sync_field WHERE erpEntryId=? AND fieldName=?"
	SELECT_FIELD_BY_ID    = "SELECT id, erpEntryId, creationDate, fieldName, erpPk, jsonName FROM admin_sync_field WHERE id=?"
	SELECT_FIELD_BY_ENTRY = "SELECT id, erpEntryId, creationDate, fieldName, erpPk, jsonName FROM admin_sync_field WHERE erpEntryId=?"
	INSERT_FIELD          = "INSERT admin_sync_field SET creationdate=?, erpEntryId=?, fieldName=?, erpPk=?, jsonName=?"
	UPDATE_FIELD_BY_ID    = "UPDATE admin_sync_field SET creationdate=?, erpEntryId=?, fieldName=?, erpPk=?, jsonName=? WHERE id=?"
	DELETE_FIELD_BY_ID    = "DELETE FROM admin_sync_field WHERE id=?"
	DELETE_SYNC_BY_ENTRY  = "Delete from admin_sync_field WHERE erpEntryId=?"
)

func (o *SyncField) loadDb() {
	fmt.Printf("loadSyncField\n")
	st, _ := dbC.Prepare(SELECT_FIELD_BY_ID)
	defer st.Close()
	rows, err := st.Query(o.Id)
	if err != nil {
		fmt.Printf("err 03\n")
	}

	for rows.Next() {
		o.loadFromDbRow(rows)
	}
}

func (o *SyncField) saveDb() {
	fmt.Printf("SyncField saveDb\n")
	st, err := dbC.Prepare(INSERT_FIELD)
	defer st.Close()
	checkErr(err)
	res, err := st.Exec(time.Now(), o.ErpEntryId, o.FieldName, o.ErpPk, o.FieldName)
	id, err := res.LastInsertId()
	o.Id = int(id)
	checkErr(err)
}

func (o *SyncField) updateDb() {
	fmt.Printf("SyncField updateDb\n")
	st, err := dbC.Prepare(UPDATE_FIELD_BY_ID)
	defer st.Close()
	checkErr(err)
	res, err := st.Exec(o.CreationDate, o.ErpEntryId, o.FieldName, o.ErpPk, o.JsonName, o.Id)
	id, err := res.LastInsertId()
	o.Id = int(id)
	checkErr(err)
}

func (o *SyncField) deleteDb() {
	st, err := dbC.Prepare(DELETE_FIELD_BY_ID)
	defer st.Close()
	checkErr(err)
	_, err = st.Exec(o.Id)
	checkErr(err)
	deleteDecoratorByField(o.Id)
}


func (o *SyncField) loadFromDbRow(rows *sql.Rows) error {
	err := rows.Scan(&o.Id, &o.ErpEntryId, &o.CreationDate, &o.FieldName, &o.ErpPk, &o.JsonName)
	if err != nil {
		fmt.Printf("err 04\n")
		return err
	}
	o.PredefinedDecorators = getPredefinedDecorator()
	o.loadDbDecorators()
	return nil
}

func (o *SyncField) decorate(s string) (string, string) {
	for _, val := range o.Decorators {
		s = val.decorate(s)
	}

	return o.JsonName, encodeUTF(s)
}

func (o *SyncField) loadDbDecorators() {
	st, _ := dbC.Prepare(COUNT_DECORATOR_BY_FIELD)
	defer st.Close()
	rows, err := st.Query(o.Id)
	checkErr(err)

	var cpt int
	for rows.Next() {
		_ = rows.Scan(&cpt)
	}

	result := make([]Decorator, cpt)

	st, err = dbC.Prepare(SELECT_DECORATOR_BY_FIELD)
	checkErr(err)
	rows, err = st.Query(o.Id)
	checkErr(err)
	i := 0
	for rows.Next() {
		o := &Decorator{}
		o.loadFromDbRow(rows)
		o.Name = decorators[o.DecoratorId].Name
		o.Description = decorators[o.DecoratorId].Description
		result[i] = *o
		i++
	}
	o.Decorators = result
	o.NbDecorator = len(o.Decorators)
}

func (o *SyncField) reOrderDecorators() {
	o.loadDbDecorators()
	cpt := 1
	for _, val := range o.Decorators {
		val.SortingOrder = cpt
		val.updateDb()
		cpt++
	}
}

func initDbSyncField(db *sql.DB) {
	defer fmt.Printf("Init DB DONE! \n")

	// TABLE FOR SYNC FIELD
	//sql = "CREATE TABLE IF NOT EXISTS `mid_db`.`admin_sync_field` (`id` int(10) unsigned NOT NULL AUTO_INCREMENT,`erpEntryId` int(10) unsigned NOT NULL DEFAULT '0',  `creationDate` datetime NOT NULL DEFAULT '0000-00-00 00:00:00',`fieldName` varchar(255) NOT NULL DEFAULT '',PRIMARY KEY (`id`)) ENGINE=InnoDB DEFAULT CHARSET=latin1;"
	sql := "CREATE TABLE IF NOT EXISTS `mid_db`.`admin_sync_field` (`id` int(10) unsigned NOT NULL AUTO_INCREMENT,`erpEntryId` int(10) unsigned NOT NULL DEFAULT '0',  `creationDate` datetime NOT NULL DEFAULT '0000-00-00 00:00:00',`fieldName` varchar(255) NOT NULL DEFAULT '',`jsonName` varchar(255) NOT NULL DEFAULT '',`erpPk` int(10) unsigned DEFAULT '0',PRIMARY KEY (`id`)) ENGINE=InnoDB DEFAULT CHARSET=latin1;"
	st, err := db.Prepare(sql)
	checkErr(err)
	_, err = st.Exec()
	checkErr(err)

	
}