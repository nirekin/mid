package main

import (
	"database/sql"
	"time"
)

type VisibleSyncField struct {
	CreationDate string              `xml:"creationDate"`
	FieldName    string              `xml:"fieldName"`
	JsonName     string              `xml:"jsonName"`
	ErpPk        bool                `xml:"erpPk"`
	Decoratos    []*VisibleDecorator `xml:"decorators>decorator"`
}

type SyncField struct {
	DBEntity
	CreationDate         string
	FieldName            string
	JsonName             string
	ErpPk                bool
	ErpEntryId           int
	Decorators           []Decorator
	PredefinedDecorators []*PredefinedDecorator
}

const (
	// FIELD
	FIELD_TABLE_NAME    = "admin_sync_field"
	FIELD_SELECT_FIELDS = "SELECT id, erpEntryId, creationDate, fieldName, erpPk, jsonName "

	FIELD_INSERT_UPDATE = " creationdate=?, erpEntryId=?, fieldName=?, erpPk=?, jsonName=? "

	COUNT_FIELD_USED      = "SELECT COUNT(*) FROM " + FIELD_TABLE_NAME + " WHERE erpEntryId=? AND fieldName=?"
	SELECT_FIELD_BY_ENTRY = FIELD_SELECT_FIELDS + "FROM " + FIELD_TABLE_NAME + " WHERE erpEntryId=?"
	INSERT_FIELD          = "INSERT " + FIELD_TABLE_NAME + " SET " + FIELD_INSERT_UPDATE
	UPDATE_FIELD_BY_ID    = "UPDATE " + FIELD_TABLE_NAME + " SET " + FIELD_INSERT_UPDATE + " WHERE id=?"
)

func (o *SyncField) loadDb() error {
	if rows, err := selectById(o); err == nil {
		for rows.Next() {
			o.loadFromDbRow(rows)
		}
		return nil
	} else {
		return err
	}
}

func (o *SyncField) saveDb() error {
	st, err := dbC.Prepare(INSERT_FIELD)
	if err != nil {
		return err
	} else {
		defer st.Close()
	}

	res, err := st.Exec(time.Now(), o.ErpEntryId, o.FieldName, o.ErpPk, o.FieldName)
	if err != nil {
		return err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	o.Id = int(id)
	return nil
}

func (o *SyncField) updateDb() error {
	st, err := dbC.Prepare(UPDATE_FIELD_BY_ID)
	if err != nil {
		return err
	} else {
		defer st.Close()
	}
	_, err = st.Exec(o.CreationDate, o.ErpEntryId, o.FieldName, o.ErpPk, o.JsonName, o.Id)
	if err != nil {
		return err
	}
	return nil
}

func (o *SyncField) loadChildDecorator() {
	o.loadDbDecorators()
	children := make([]Deleter, len(o.Decorators))
	for i, val := range o.Decorators {
		children[i] = val
	}
	o.Children = children
}

func (o *SyncField) deleteDb() error {
	o.loadChildDecorator()
	err := delete(o)
	if err != nil {
		return err
	}
	return nil
}

func (o *SyncField) loadFromDbRow(rows *sql.Rows) error {
	err := rows.Scan(&o.Id, &o.ErpEntryId, &o.CreationDate, &o.FieldName, &o.ErpPk, &o.JsonName)
	if err != nil {
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

func (o *SyncField) loadDbDecorators() error {
	var tResult [10]Decorator
	result := tResult[0:0]

	st, err := dbC.Prepare(SELECT_DECORATOR_BY_FIELD)
	if err != nil {
		return err
	}
	rows, err := st.Query(o.Id)
	if err != nil {
		return err
	}
	for rows.Next() {
		o := Decorator{}
		o.loadFromDbRow(rows)
		o.Name = decorators[o.DecoratorId].Name
		o.Description = decorators[o.DecoratorId].Description
		result = append(result, o)
	}
	o.Decorators = result
	return nil
}

func (o *SyncField) NbDecorator() int {
	if o.Decorators == nil {
		return 0
	} else {
		return len(o.Decorators)
	}
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

func initDbSyncField(db *sql.DB) error {
	sql := "CREATE TABLE IF NOT EXISTS `mid_db`.`admin_sync_field` (`id` int(10) unsigned NOT NULL AUTO_INCREMENT,`erpEntryId` int(10) unsigned NOT NULL DEFAULT '0',  `creationDate` datetime NOT NULL DEFAULT '0000-00-00 00:00:00',`fieldName` varchar(255) NOT NULL DEFAULT '',`jsonName` varchar(255) NOT NULL DEFAULT '',`erpPk` int(10) unsigned DEFAULT '0',PRIMARY KEY (`id`)) ENGINE=InnoDB DEFAULT CHARSET=latin1;"
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

func (e SyncField) getTableName() string {
	return FIELD_TABLE_NAME
}

func (e SyncField) getSelectFields() string {
	return FIELD_SELECT_FIELDS
}
