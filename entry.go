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
	CreationDate string
	SourceName   string
	Name         string
	Fields       []*VisibleSyncField
}

type ErpEntry struct {
	CreationDate string
	SourceName   string
	Name         string
	Id           int
	ErpId        int
	Fields       []ErpRField
	SyncFields   []SyncField
	PingLines    []PingLine
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
	ENTRY_SELECT_FIELDS = "SELECT id, creationDate, erpId, sourceName, name "
	ENTRY_INSERT_UPDATE = " creationDate=?, erpId=?, sourceName=?, name=?"

	COUNT_ENTRY_USED    = "SELECT COUNT(*) FROM admin_erp_entry WHERE erpId=? AND sourceName=?"
	SELECT_ENTRY_ALL    = ENTRY_SELECT_FIELDS + "FROM admin_erp_entry"
	SELECT_ENTRY_BY_ID  = ENTRY_SELECT_FIELDS + "FROM admin_erp_entry WHERE id=?"
	SELECT_ENTRY_BY_ERP = ENTRY_SELECT_FIELDS + "FROM admin_erp_entry WHERE erpId=?"
	INSERT_ENTRY        = "INSERT admin_erp_entry SET " + ENTRY_INSERT_UPDATE
	UPDATE_ENTRY_BY_ID  = "UPDATE admin_erp_entry SET " + ENTRY_INSERT_UPDATE + " WHERE id=?"
	DELETE_ENTRY        = "DELETE FROM admin_erp_entry WHERE id=?"

	SELECT_TABLE_MYSQL = "select COLUMN_NAME from information_schema.columns where TABLE_SCHEMA = ? AND TABLE_NAME =?"

	DATA_TABLE_NAME = "data_erp_entry_content_"
)

func (o *ErpEntry) loadDb() error {
	st, err := dbC.Prepare(SELECT_ENTRY_BY_ID)
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

func (o *ErpEntry) saveDb() error {
	st, err := dbC.Prepare(INSERT_ENTRY)
	if err != nil {
		return err
	} else {
		defer st.Close()
	}
	res, err := st.Exec(time.Now(), o.ErpId, o.SourceName, o.Name)
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

func (o *ErpEntry) updateDb() error {
	st, err := dbC.Prepare(UPDATE_ENTRY_BY_ID)
	if err != nil {
		return err
	} else {
		defer st.Close()
	}
	_, err = st.Exec(o.CreationDate, o.ErpId, o.SourceName, o.Name, o.Id)
	if err != nil {
		return err
	}
	return nil
}

func (o *ErpEntry) deleteDb() error {
	st, err := dbC.Prepare(DELETE_ENTRY)
	if err != nil {
		return err
	} else {
		defer st.Close()
	}
	_, err = st.Exec(o.Id)
	if err != nil {
		return err
	}
	o.loadDbSyncFields()
	for _, val := range o.SyncFields {
		val.deleteDb()
	}

	st, err = dbC.Prepare(fmt.Sprintf("DROP TABLE IF EXISTS `mid_db`.`%s`", o.getImportationTableName()))
	if err != nil {
		return err
	}
	_, err = st.Exec()
	if err != nil {
		return err
	}
	return nil
}

func (o *ErpEntry) loadFromDbRow(rows *sql.Rows) error {
	err := rows.Scan(&o.Id, &o.CreationDate, &o.ErpId, &o.SourceName, &o.Name)
	if err != nil {
		return err
	}
	o.loadDbSyncFields()
	o.checkImportationTableName()
	return nil
}

func (o *ErpEntry) lazyLoadRFields() error {
	erp := &Erp{Id: o.ErpId}
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

	st, err := dbC.Prepare(SELECT_FIELD_BY_ENTRY)
	if err != nil {
		return err
	}

	rows, err := st.Query(o.Id)
	if err != nil {
		return err
	}
	for rows.Next() {
		o := SyncField{}
		o.loadFromDbRow(rows)
		result = append(result, o)
	}
	o.SyncFields = result
	return nil
}

func (o *ErpEntry) getExtractSentence() string {
	erp := &Erp{Id: o.ErpId}
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

func (o *ErpEntry) sync() error {
	extractSentence := o.getExtractSentence()
	if extractSentence == "" {
		return nil
	}

	erp := &Erp{Id: o.ErpId}
	erp.loadDb()
	cpt := 0

	if erp.TypeInt == MYSQL_TYPE {
		dbCErp, err := sql.Open("mysql", erp.Value)
		if err != nil {
			return err
		} else {
			defer dbCErp.Close()
		}
		st, err := dbCErp.Prepare(extractSentence)
		if err != nil {
			return err
		} else {
			defer st.Close()
		}

		rows, err := st.Query()
		if err != nil {
			return err
		}
		var content string
		l := o.SyncFields
		ecMap := make(map[string]*ExtractedContent)
		for rows.Next() {
			err := rows.Scan(&content)
			if err == nil {
				var pkContent string
				ps := strings.Split(content, MYSQL_TYPE_SPLIT)
				lp := len(ps)
				mapD := map[string]string{}
				for i := 0; i < lp; i++ {
					str := strings.Replace(ps[i], MYSQL_TYPE_EMPTY, "", -1)
					if l[i].ErpPk {
						pkContent += str
					}
					fN, val := l[i].decorate(str)
					mapD[fN] = val
				}
				outJson, _ := json.Marshal(mapD)
				ecMap[pkContent] = &ExtractedContent{ErpEntryId: o.Id, ErpPk: pkContent, Content: string(outJson)}
			}
			cpt++
		}
		go insertOrUpdate(o, ecMap)

	}
	return nil
}

func (o *ErpEntry) ping(nbRows int) error {
	l := o.SyncFields
	extractSentence := o.getExtractSentence()
	if extractSentence == "" {
		o.PingLines = make([]PingLine, 0)
		return nil
	}

	if nbRows <= 1 {
		nbRows = 1
	}
	erp := &Erp{Id: o.ErpId}
	erp.loadDb()

	if erp.TypeInt == MYSQL_TYPE {
		dbCErp, err := sql.Open("mysql", erp.Value)
		if err != nil {
			return err
		} else {
			defer dbCErp.Close()
		}
		st, err := dbCErp.Prepare(extractSentence)
		if err != nil {
			return err
		} else {
			defer st.Close()
		}
		rows, err := st.Query()
		if err != nil {
			return err
		}
		result := make([]PingLine, nbRows)
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
				result[cpt] = PingLine{string(outJson)}
			}
			if cpt == nbRows-1 {
				break
			}
			cpt++
		}
		o.PingLines = result
	}
	return nil
}

func (o *ErpEntry) getImportationTableName() string {
	return DATA_TABLE_NAME + strconv.Itoa(o.Id)
}

func (o *ErpEntry) createImportationTableName() {
	o.checkImportationTableName()
}

func (o *ErpEntry) checkImportationTableName() error {
	// CREATE THE DATA TABLE TO STORE THE IMPORTED CONTENT
	sql := "CREATE TABLE IF NOT EXISTS `mid_db`.`" + o.getImportationTableName() + "` ( `id` int(11) NOT NULL AUTO_INCREMENT,  `active` tinyint(1) NOT NULL,`content` text,  `creationDate` bigint(20) unsigned DEFAULT NULL, `erpPk` varchar(255) DEFAULT NULL,`lastUpdate` bigint(20) unsigned DEFAULT NULL,`name` varchar(255) DEFAULT NULL,`processedFromERP` tinyint(1) NOT NULL,PRIMARY KEY (`id`)) ENGINE=InnoDB AUTO_INCREMENT=13222 DEFAULT CHARSET=latin1;"
	st, err := dbC.Prepare(sql)
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

func (o *ErpEntry) getLoadedContent(likeOnErpPk, likeOnContent string, limit int) ([]*LoadedContentLine, error) {
	if limit <= 0 {
		limit = 10
	} else if limit > 500 {
		limit = 500
	}
	likeOnErpPk = encodeUTF(likeOnErpPk)
	likeOnContent = encodeUTF(likeOnContent)
	result := make([]*LoadedContentLine, limit)

	selectString := "SELECT erpPk, creationDate, lastUpdate, name, content FROM " + o.getImportationTableName()
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
	} else {

	}

	st, err := dbC.Prepare(selectString + whereString + " LIMIT " + strconv.Itoa(limit))
	if err != nil {
		return nil, err
	} else {
		defer st.Close()
	}
	rows, err := st.Query()
	if err != nil {
		return nil, err
	}
	i := 0
	for rows.Next() {
		if i == limit {
			break
		}
		lcl := &LoadedContentLine{}
		err = rows.Scan(&lcl.ErpPk, &lcl.CreationDate, &lcl.LastUpdate, &lcl.Name, &lcl.Content)
		if err != nil {
			return nil, err
		}
		result[i] = lcl
		i++
	}
	return result, nil
}

func insertOrUpdate(entry *ErpEntry, ec ExtractedContentMap) error {
	var inserted int64 = 0
	var updated int64 = 0

	timeMSStart := getNowMillisecond()
	tableName := fmt.Sprintf("`mid_db`.`%s`", entry.getImportationTableName())
	var nbExistringRows int

	stMark, err := dbC.Prepare("UPDATE " + tableName + " SET processedFromERP=0")
	if err != nil {
		return err
	} else {
		defer stMark.Close()
	}

	_, err = stMark.Exec()
	if err != nil {
		return err
	}

	st1, _ := dbC.Prepare("SELECT COUNT(*) FROM " + tableName)
	if err != nil {
		return err
	} else {
		defer st1.Close()
	}
	rows, err := st1.Query()
	if err != nil {
		return err
	}
	for rows.Next() {
		_ = rows.Scan(&nbExistringRows)
	}
	existingPKs := make([]string, nbExistringRows)

	st2, _ := dbC.Prepare("SELECT erpPk FROM " + tableName)
	if err != nil {
		return err
	} else {
		defer st2.Close()
	}
	rows, err = st2.Query()
	if err != nil {
		return err
	}
	i := 0
	var val string
	for rows.Next() {
		_ = rows.Scan(&val)
		existingPKs[i] = val
		i++
	}

	stIns, err := dbC.Prepare("INSERT " + tableName + " SET active=1, content=?, creationDate=?, erpPk=?, lastUpdate=?, name=?, processedFromERP=1")
	if err != nil {
		return err
	} else {
		defer stIns.Close()
	}

	stUpdate, err := dbC.Prepare("UPDATE " + tableName + " SET content=?, lastUpdate=?, processedFromERP=1 WHERE erpPk=?")
	if err != nil {
		return err
	} else {
		defer stUpdate.Close()
	}

	stUpdateProcessed, err := dbC.Prepare("UPDATE " + tableName + " SET processedFromERP=1 WHERE erpPk=?")
	if err != nil {
		return err
	} else {
		defer stUpdateProcessed.Close()
	}

	var keys []string
	for k := range ec {
		keys = append(keys, k)
	}

	if len(existingPKs) == 0 {
		for _, k := range keys {
			// TODO Opimize this
			//INSERT INTO temp_data_broker (id,name) VALUES(36,'Santiago (copia)'),(34,'Zumaya') ... bulk sample
			c := ec[k]
			n := getNowMillisecond()
			_, err := stIns.Exec(c.Content, n, c.ErpPk, n, entry.Name)
			inserted++
			if err != nil {
				return err
			}
		}
	} else {
		var cptE int
		for _, k := range keys {
			c := ec[k]
			rows, err := dbC.Query("SELECT COUNT(*) FROM " + tableName + " WHERE erpPk='" + c.ErpPk + "'")
			if err != nil {
				return err
			}
			for rows.Next() {
				_ = rows.Scan(&cptE)
			}
			if cptE == 1 {
				s := "SELECT COUNT(*) FROM " + tableName + " WHERE erpPk='" + c.ErpPk + "' AND content='" + c.Content + "'"
				rows, err := dbC.Query(s)
				if err != nil {
					return err
				}
				for rows.Next() {
					_ = rows.Scan(&cptE)
				}
				if cptE == 0 {

					_, err := stUpdate.Exec(c.Content, getNowMillisecond(), c.ErpPk)
					updated++
					if err != nil {
						return err
					}
				} else {
					_, err := stUpdateProcessed.Exec(c.ErpPk)
					if err != nil {
						return err
					}
				}
			} else {
				n := getNowMillisecond()
				_, err := stIns.Exec(c.Content, n, c.ErpPk, n, entry.Name)
				inserted++
				if err != nil {
					return err
				}
			}
		}
	}

	stDelete, err := dbC.Prepare("DELETE FROM " + tableName + " WHERE processedFromERP=0")
	if err != nil {
		return err
	} else {
		defer stDelete.Close()
	}
	res, err := stDelete.Exec()
	if err != nil {
		return err
	}
	deleted, _ := res.RowsAffected()
	timeMSStop := getNowMillisecond()
	_ = addEvent(entry, inserted, updated, deleted, timeMSStop-timeMSStart, int64(len(keys)))
	return nil
}

func getErpEntries() ([]ErpEntry, error) {
	var tResult [10]ErpEntry
	result := tResult[0:0]

	st, err := dbC.Prepare(SELECT_ENTRY_ALL)
	if err != nil {
		return nil, err
	}

	rows, err := st.Query()
	if err != nil {
		return nil, err
	}

	i := 0
	for rows.Next() {
		o := ErpEntry{}
		o.loadFromDbRow(rows)
		result = append(result, o)
		i++
	}
	return result, nil
}

func initDbEntry(db *sql.DB) error {
	sql := "CREATE TABLE IF NOT EXISTS `mid_db`.`admin_erp_entry` (`id` int(10) unsigned NOT NULL AUTO_INCREMENT,`erpId` int(10) unsigned NOT NULL DEFAULT '0',  `creationDate` datetime NOT NULL DEFAULT '0000-00-00 00:00:00',`sourceName` varchar(255) NOT NULL DEFAULT '',`name` varchar(255) NOT NULL DEFAULT '',PRIMARY KEY (`id`)) ENGINE=InnoDB DEFAULT CHARSET=latin1;"
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
