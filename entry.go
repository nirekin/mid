package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type VisibleErpEntry struct {
	CreationDate string              `xml:"creationdate"`
	SourceName   string              `xml:"sourceName"`
	Name         string              `xml:"name"`
	Fields       []*VisibleSyncField `xml:"fields>field"`
}

type ErpEntry struct {
	DBEntity
	CreationDate string
	SourceName   string
	Name         string
	ErpId        int
	BlockSize    int
	Fields       []ErpRField
	SyncFields   []SyncField
}

type InspectedEntry struct {
	Entry                *ErpEntry
	ImportationTableName string
	ImportedRows         int
	AGVRowsLength        int
	DataLength           int
	DataFree             int
	AutoIncrement        int
	CreateTime           string
	TableCollation       string
	Limit                int
	LikeOnErpPk          string
	LikeOnContent        string
	LoadedContentLines   []*LoadedContentLine
}

const (
	// ENTRY
	ENTRY_TABLE_NAME = "admin_erp_entry"

	ENTRY_SELECT_FIELDS = "SELECT id, creationDate, erpId, sourceName, name, blockSize "
	ENTRY_INSERT_UPDATE = " creationDate=?, erpId=?, sourceName=?, name=?, blockSize=?"

	COUNT_ENTRY_USED    = "SELECT COUNT(*) FROM " + ENTRY_TABLE_NAME + " WHERE erpId=? AND sourceName=?"
	SELECT_ENTRY_BY_ERP = ENTRY_SELECT_FIELDS + "FROM " + ENTRY_TABLE_NAME + " WHERE erpId=?"
	INSERT_ENTRY        = "INSERT " + ENTRY_TABLE_NAME + " SET " + ENTRY_INSERT_UPDATE
	UPDATE_ENTRY_BY_ID  = "UPDATE " + ENTRY_TABLE_NAME + " SET " + ENTRY_INSERT_UPDATE + " WHERE id=?"

	SELECT_TABLE_MYSQL = "select COLUMN_NAME from information_schema.columns where TABLE_SCHEMA = ? AND TABLE_NAME =?"

	DATA_TABLE_NAME = "data_erp_entry_content_"
)

func (o *ErpEntry) loadDb() error {

	if rows, err := selectById(o); err == nil {
		for rows.Next() {
			o.loadFromDbRow(rows)
		}
	} else {
		return err
	}
	return nil
}

func (o *ErpEntry) saveDb() error {
	st, err := dbC.Prepare(INSERT_ENTRY)
	if err != nil {
		return err
	}
	defer st.Close()

	if res, err := st.Exec(time.Now(), o.ErpId, o.SourceName, o.Name, o.BlockSize); err != nil {
		return err
	} else {
		if id, err := res.LastInsertId(); err == nil {
			o.Id = int(id)
		} else {
			return err
		}
	}
	return nil
}

func (o *ErpEntry) updateDb() error {
	if st, err := dbC.Prepare(UPDATE_ENTRY_BY_ID); err != nil {
		return err
	} else {
		defer st.Close()
		if _, err = st.Exec(o.CreationDate, o.ErpId, o.SourceName, o.Name, o.BlockSize, o.Id); err != nil {
			return err
		}
	}
	return nil
}

func (o *ErpEntry) loadChildSyncFields() {
	o.loadDbSyncFields()
	childrenS := make([]Deleter, len(o.SyncFields))
	for i, valS := range o.SyncFields {
		childrenS[i] = valS
		valS.loadChildDecorator()
	}
	o.Children = childrenS
}

func (o *ErpEntry) deleteDb() error {
	o.loadChildSyncFields()
	if err := delete(o); err != nil {
		return err
	}
	if st, err := dbC.Prepare(fmt.Sprintf("DROP TABLE IF EXISTS %s", o.getImportationTableSchema())); err != nil {
		return err
	} else {
		if _, err := st.Exec(); err != nil {
			return err
		}
	}
	return nil
}

func (o *ErpEntry) loadFromDbRow(rows *sql.Rows) error {
	if err := rows.Scan(&o.Id, &o.CreationDate, &o.ErpId, &o.SourceName, &o.Name, &o.BlockSize); err != nil {
		return err
	} else {
		err = o.loadDbSyncFields()
		o.checkImportationTableName()
	}
	return nil
}

func (o *ErpEntry) lazyLoadRFields() error {
	erp := &Erp{DBEntity: DBEntity{Id: o.ErpId}}
	erp.loadDb()

	if erp.TypeInt == MYSQL_TYPE {
		desiredSchema := getMySqlSchema(erp.Value)
		dbCErp, err := sql.Open("mysql", erp.Value)
		if err != nil {
			return err
		}

		var tResult [10]ErpRField
		result := tResult[0:0]

		st, err := dbCErp.Prepare(SELECT_TABLE_MYSQL)
		if err != nil {
			return err
		}
		rows, err := st.Query(desiredSchema, o.SourceName)
		if err != nil {
			return err
		}

		var name string
		for rows.Next() {
			e := ErpRField{ErpEntryId: o.Id}
			err := rows.Scan(&name)
			if err != nil {
				return err
			}
			e.Name = name
			e.loadUsed()
			result = append(result, e)
		}
		o.Fields = result
		return nil
	} else {
		result := make([]ErpRField, 1)
		result[0].ErpEntryId = o.Id
		result[0].Used = 0
		result[0].Name = "ERP Type not implemented yet"
		o.Fields = result

	}
	return nil
}
func (o *ErpEntry) loadDbSyncFields() error {
	var tResult [10]SyncField
	result := tResult[0:0]
	if st, err := dbC.Prepare(SELECT_FIELD_BY_ENTRY); err != nil {
		return err
	} else {
		if rows, err := st.Query(o.Id); err != nil {
			return err
		} else {
			for rows.Next() {
				o := SyncField{}
				o.loadFromDbRow(rows)
				result = append(result, o)
			}
		}
	}
	o.SyncFields = result
	return nil
}

func (o *ErpEntry) getExtractSentence() string {
	erp := &Erp{DBEntity: DBEntity{Id: o.ErpId}}
	erp.loadDb()
	if erp.TypeInt == MYSQL_TYPE {
		l := o.SyncFields
		if len(l) == 0 {
			return ""
		}
		fl := fmt.Sprintf("SELECT concat_ws('%s' ", MYSQL_TYPE_SPLIT)
		for _, val := range l {
			fl += ",IF(" + val.FieldName + "<>\"\"," + val.FieldName + ",\"" + MYSQL_TYPE_EMPTY + "\")"
		}
		fl += ")"
		s := fmt.Sprintf("%s FROM %s.%s", fl, getMySqlSchema(erp.Value), o.SourceName)
		return s
	}
	return ""
}

func (o *ErpEntry) ping(nbRows int) ([]string, error) {
	l := o.SyncFields
	extractSentence := o.getExtractSentence()
	if extractSentence == "" {
		return make([]string, 0), nil
	}

	if nbRows <= 1 {
		nbRows = 1
	}
	erp := &Erp{DBEntity: DBEntity{Id: o.ErpId}}
	erp.loadDb()

	result := make([]string, nbRows)

	if erp.TypeInt == MYSQL_TYPE {
		dbCErp, err := sql.Open("mysql", erp.Value)
		if err != nil {
			return nil, err
		} else {
			defer dbCErp.Close()
		}
		st, err := dbCErp.Prepare(extractSentence)
		if err != nil {
			return nil, err
		} else {
			defer st.Close()
		}
		rows, err := st.Query()
		if err != nil {
			return nil, err
		}

		var content string
		cpt := 0
		for rows.Next() {
			err := rows.Scan(&content)
			if err == nil {
				ps := strings.Split(content, MYSQL_TYPE_SPLIT)
				lp := len(ps)
				mapD := map[string]string{}
				for i := 0; i < lp; i++ {
					str := strings.Replace(ps[i], MYSQL_TYPE_EMPTY, "", -1)
					fN, val := l[i].decorate(str)
					mapD[fN] = val
				}
				outJson, _ := json.Marshal(mapD)
				result[cpt] = string(outJson)
			}
			if cpt == nbRows-1 {
				break
			}
			cpt++
		}
	}
	return result, nil
}

func (o *ErpEntry) getImportationTable() string {
	return DATA_TABLE_NAME + strconv.Itoa(o.Id)
}

func (o *ErpEntry) getImportationTableSchema() string {
	return fmt.Sprintf("`mid_db`.`%s`", o.getImportationTable())
}

func (o *ErpEntry) createImportationTableName() {
	o.checkImportationTableName()
}

func (o *ErpEntry) checkImportationTableName() error {
	// CREATE THE DATA TABLE TO STORE THE IMPORTED CONTENT
	sql := "CREATE TABLE IF NOT EXISTS " + o.getImportationTableSchema() +
		" ( `id` int(11) NOT NULL AUTO_INCREMENT,  `active` tinyint(1) NOT NULL,`content` text" +
		",  `creationDate` bigint(20) unsigned DEFAULT NULL, `erpPk` varchar(255) DEFAULT NULL," +
		"`lastUpdate` bigint(20) unsigned DEFAULT NULL,`name` varchar(255) DEFAULT NULL,`processedFromERP`" +
		" tinyint(1) NOT NULL,PRIMARY KEY (`id`)) ENGINE=InnoDB AUTO_INCREMENT=13222 DEFAULT CHARSET=latin1;"
	if st, err := dbC.Prepare(sql); err != nil {
		return err
	} else {
		defer st.Close()
		if _, err = st.Exec(); err != nil {
			return err
		}
	}
	return nil
}

func (o *ErpEntry) getLoadedContent(likeOnErpPk, likeOnContent string, limit int) ([]*LoadedContentLine, error) {
	if limit <= 0 {
		limit = 10
	} else if limit > 500 {
		limit = 500
	}
	likeOnErpPk = encodeUTF(likeOnErpPk)
	likeOnContent = encodeUTF(likeOnContent)
	result := make([]*LoadedContentLine, limit)

	selectString := "SELECT erpPk, creationDate, lastUpdate, name, content FROM " + o.getImportationTable()
	whereString := ""

	if likeOnErpPk != "" || likeOnContent != "" {
		whereString += " WHERE "
		onPk := false
		if likeOnErpPk != "" {
			whereString += " erpPk like '%" + likeOnErpPk + "%'"
			onPk = true
		}
		if likeOnContent != "" {
			if onPk {
				whereString += " AND "
			}
			whereString += " content like '%" + likeOnContent + "%'"
			onPk = true
		}
	}

	if st, err := dbC.Prepare(selectString + whereString + " LIMIT " + strconv.Itoa(limit)); err != nil {
		return nil, err
	} else {
		defer st.Close()
		if rows, err := st.Query(); err != nil {
			return nil, err
		} else {
			i := 0
			for rows.Next() {
				if i == limit {
					break
				}
				lcl := &LoadedContentLine{}
				if err = rows.Scan(&lcl.ErpPk, &lcl.CreationDate, &lcl.LastUpdate, &lcl.Name, &lcl.Content); err != nil {
					return nil, err
				}
				result[i] = lcl
				i++
			}
			return result, nil
		}
	}
}

func getErpEntries() ([]ErpEntry, error) {
	var tResult [10]ErpEntry
	result := tResult[0:0]
	if rows, err := selectAll(&ErpEntry{}); err == nil {
		i := 0
		for rows.Next() {
			o := ErpEntry{}
			o.loadFromDbRow(rows)
			result = append(result, o)
			i++
		}
		return result, nil
	} else {
		return nil, err
	}
}

func initDbEntry(db *sql.DB) error {
	sql := "CREATE TABLE IF NOT EXISTS `mid_db`.`admin_erp_entry` " +
		"(`id` int(10) unsigned NOT NULL AUTO_INCREMENT,`erpId` int(10) unsigned NOT NULL DEFAULT '0'" +
		",  `creationDate` datetime NOT NULL DEFAULT '0000-00-00 00:00:00',`sourceName` varchar(255) NOT NULL DEFAULT ''" +
		",`name` varchar(255) NOT NULL DEFAULT '',`blockSize` int(10) unsigned NOT NULL DEFAULT '0'," +
		"PRIMARY KEY (`id`)) ENGINE=InnoDB DEFAULT CHARSET=latin1;"

	if st, err := db.Prepare(sql); err != nil {
		return err
	} else {
		defer st.Close()
		if _, err = st.Exec(); err != nil {
			return err
		}
	}
	return nil
}

func (e ErpEntry) getTableName() string {
	return ENTRY_TABLE_NAME
}

func (e ErpEntry) getSelectFields() string {
	return ENTRY_SELECT_FIELDS
}
