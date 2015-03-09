package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
	"strings"
	"strconv"
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


const(
	// ENTRY
	COUNT_ENTRY_ALL     = "SELECT COUNT(*) FROM admin_erp_entry"
	COUNT_ENTRY_USED    = "SELECT COUNT(*) FROM admin_erp_entry WHERE erpId=? AND sourceName=?"
	COUNT_ENTRY_BY_ERP  = "SELECT COUNT(*) FROM admin_erp_entry WHERE erpId=?"
	SELECT_ENTRY_ALL    = "SELECT id, creationDate, erpId, sourceName, name FROM admin_erp_entry"
	SELECT_ENTRY_BY_ID  = "SELECT id, creationDate, erpId, sourceName, name FROM admin_erp_entry WHERE id=?"
	SELECT_ENTRY_BY_ERP = "SELECT id, creationDate, erpId, sourceName, name FROM admin_erp_entry WHERE erpId=?"
	INSERT_ENTRY        = "INSERT admin_erp_entry SET creationDate=?, erpId=?, sourceName=?, name=?"
	UPDATE_ENTRY_BY_ID  = "UPDATE admin_erp_entry SET creationDate=?, erpId=?, sourceName=?, name=? WHERE id=?"
	DELETE_ENTRY        = "DELETE FROM admin_erp_entry WHERE id=?"
	
	DATA_TABLE_NAME = "data_erp_entry_content_"
)

func (o *ErpEntry) loadDb() {
	fmt.Printf("loadErpEntry\n")
	st, _ := dbC.Prepare(SELECT_ENTRY_BY_ID)
	defer st.Close()
	rows, err := st.Query(o.Id)
	if err != nil {
		fmt.Printf("err 03\n")
	}
	for rows.Next() {
		o.loadFromDbRow(rows)
	}
}

func (o *ErpEntry) saveDb() {
	fmt.Printf("ErpEntry saveDB\n")
	st, err := dbC.Prepare(INSERT_ENTRY)
	defer st.Close()
	checkErr(err)
	res, err := st.Exec(time.Now(), o.ErpId, o.SourceName, o.Name)
	id, err := res.LastInsertId()
	o.Id = int(id)
	checkErr(err)
}

func (o *ErpEntry) updateDb() {
	fmt.Printf("ErpEntry updateDB\n")
	st, err := dbC.Prepare(UPDATE_ENTRY_BY_ID)
	defer st.Close()
	checkErr(err)
	_, err = st.Exec(o.CreationDate, o.ErpId, o.SourceName, o.Name, o.Id)
	checkErr(err)
}

func (o *ErpEntry) deleteDb() {
	fmt.Printf("ErpEntry deleteDB\n")
	st, err := dbC.Prepare(DELETE_ENTRY)
	defer st.Close()
	checkErr(err)
	_, err = st.Exec(o.Id)
	checkErr(err)
	o.loadDbSyncFields()
	for _, val := range o.SyncFields {
		val.deleteDb()
	}

	st, err = dbC.Prepare(fmt.Sprintf("DROP TABLE IF EXISTS `mid_db`.`%s`", o.getImportationTableName()))
	checkErr(err)
	_, err = st.Exec()
	checkErr(err)
}

func (o *ErpEntry) loadFromDbRow(rows *sql.Rows) error {
	err := rows.Scan(&o.Id, &o.CreationDate, &o.ErpId, &o.SourceName, &o.Name)
	if err != nil {
		fmt.Printf("err 04\n")
		return err
	}
	o.loadDbSyncFields()
	o.checkImportationTableName()
	return nil
}

func (o *ErpEntry) lazyLoadRFields() error {
	fmt.Printf("ErpEntry lazyLoadRFields\n")
	erp := &Erp{Id: o.ErpId}
	erp.loadDb()

	if erp.TypeInt == MYSQL_TYPE {
		desiredSchema := getMySqlSchema(erp.Value)
		dbCErp, _ := sql.Open("mysql", erp.Value)
		defer dbCErp.Close()

		st, err := dbCErp.Prepare(COUNT_TABLE_MYSQL)
		defer st.Close()
		checkErr(err)
		rows, err := st.Query(desiredSchema, o.SourceName)
		checkErr(err)

		var cpt int
		for rows.Next() {
			_ = rows.Scan(&cpt)
		}

		result := make([]ErpRField, cpt)

		st, err = dbCErp.Prepare(SELECT_TABLE_MYSQL)
		checkErr(err)
		rows, err = st.Query(desiredSchema, o.SourceName)
		checkErr(err)
		i := 0
		var name string
		for rows.Next() {
			e := &ErpRField{}
			e.ErpEntryId = o.Id
			err := rows.Scan(&name)
			if err != nil {
				fmt.Printf("err 04 %v\n", err)
				return err
			}
			e.Name = name
			e.loadUsed()
			result[i] = *e
			i++
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
func (o *ErpEntry) loadDbSyncFields() {
	st, _ := dbC.Prepare(COUNT_FIELD_BY_ENTRY)
	defer st.Close()
	rows, err := st.Query(o.Id)
	checkErr(err)
	var cpt int
	for rows.Next() {
		_ = rows.Scan(&cpt)
	}
	result := make([]SyncField, cpt)
	st, err = dbC.Prepare(SELECT_FIELD_BY_ENTRY)
	checkErr(err)

	rows, err = st.Query(o.Id)
	checkErr(err)
	i := 0
	for rows.Next() {
		o := &SyncField{}
		o.loadFromDbRow(rows)
		result[i] = *o
		i++
	}
	o.SyncFields = result
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
		fmt.Printf("extract sentence %s\n", s)
		return s
	}
	return ""
}

func (o *ErpEntry) sync() error {
	extractSentence := o.getExtractSentence()
	if extractSentence == "" {
		return nil
	}
	fmt.Printf("ErpEntry sync\n")
	erp := &Erp{Id: o.ErpId}
	erp.loadDb()
	cpt := 0

	if erp.TypeInt == MYSQL_TYPE {
		dbCErp, _ := sql.Open("mysql", erp.Value)
		defer dbCErp.Close()
		st, err := dbCErp.Prepare(extractSentence)
		defer st.Close()
		checkErr(err)
		rows, err := st.Query()
		checkErr(err)
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
	fmt.Printf("ErpEntry ping\n")
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
		dbCErp, _ := sql.Open("mysql", erp.Value)
		defer dbCErp.Close()
		st, err := dbCErp.Prepare(extractSentence)
		defer st.Close()
		checkErr(err)
		rows, err := st.Query()
		checkErr(err)
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

func (o *ErpEntry) checkImportationTableName() {
	// CREATE THE DATA TABLE TO STORE THE IMPORTED CONTENT
	sql := "CREATE TABLE IF NOT EXISTS `mid_db`.`" + o.getImportationTableName() + "` ( `id` int(11) NOT NULL AUTO_INCREMENT,  `active` tinyint(1) NOT NULL,`content` text,  `creationDate` bigint(20) unsigned DEFAULT NULL, `erpPk` varchar(255) DEFAULT NULL,`lastUpdate` bigint(20) unsigned DEFAULT NULL,`name` varchar(255) DEFAULT NULL,`processedFromERP` tinyint(1) NOT NULL,PRIMARY KEY (`id`)) ENGINE=InnoDB AUTO_INCREMENT=13222 DEFAULT CHARSET=latin1;"
	st, err := dbC.Prepare(sql)
	defer st.Close()
	checkErr(err)
	_, err = st.Exec()
	checkErr(err)
}

func (o *ErpEntry) getLoadedContent(likeOnErpPk, likeOnContent string, limit int) []*LoadedContentLine {
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
	checkErr(err)
	rows, err := st.Query()
	checkErr(err)
	i := 0
	for rows.Next() {
		if i == limit {
			break
		}
		lcl := &LoadedContentLine{}
		err = rows.Scan(&lcl.ErpPk, &lcl.CreationDate, &lcl.LastUpdate, &lcl.Name, &lcl.Content)
		if err != nil {
			fmt.Printf("loaded err %v\n", err)
		}
		result[i] = lcl
		i++
	}
	return result
}

func insertOrUpdate(entry *ErpEntry, ec ExtractedContentMap) {

	timeMSStart := time.Now().UnixNano() / int64(time.Millisecond)

	fmt.Printf("ExtractedContent  insertOrUpdate\n")
	tableName := fmt.Sprintf("`mid_db`.`%s`", entry.getImportationTableName())
	var nbExistringRows int

	stMark, _ := dbC.Prepare("UPDATE " + tableName + " SET processedFromERP=0")
	defer stMark.Close()
	_, err := stMark.Exec()

	st1, _ := dbC.Prepare("SELECT COUNT(*) FROM " + tableName)
	defer st1.Close()
	rows, err := st1.Query()
	checkErr(err)
	for rows.Next() {
		_ = rows.Scan(&nbExistringRows)
	}
	existingPKs := make([]string, nbExistringRows)

	st2, _ := dbC.Prepare("SELECT erpPk FROM " + tableName)
	defer st2.Close()
	rows, err = st2.Query()
	checkErr(err)
	i := 0
	var val string
	for rows.Next() {
		_ = rows.Scan(&val)
		existingPKs[i] = val
		i++
	}

	stIns, err := dbC.Prepare("INSERT " + tableName + " SET active=1, content=?, creationDate=?, erpPk=?, lastUpdate=?, name=?, processedFromERP=1")
	defer stIns.Close()
	checkErr(err)

	stUpdate, err := dbC.Prepare("UPDATE " + tableName + " SET content=?, lastUpdate=?, processedFromERP=1 WHERE erpPk=?")
	defer stUpdate.Close()
	checkErr(err)

	stUpdateProcessed, err := dbC.Prepare("UPDATE " + tableName + " SET processedFromERP=1 WHERE erpPk=?")
	defer stUpdateProcessed.Close()
	checkErr(err)

	var keys []string
	for k := range ec {
		keys = append(keys, k)
	}

	if len(existingPKs) == 0 {
		fmt.Printf("firstInsert into  %v\n", tableName)
		for _, k := range keys {
			// TODO Opimize this
			//INSERT INTO temp_data_broker (id,name) VALUES(36,'Santiago (copia)'),(34,'Zumaya') ... bulk sample
			c := ec[k]
			n := time.Now().UnixNano() / int64(time.Millisecond)
			_, err := stIns.Exec(c.Content, n, c.ErpPk, n, entry.Name)
			checkErr(err)
		}
	} else {
		var cptE int
		for _, k := range keys {
			c := ec[k]
			rows, err := dbC.Query("SELECT COUNT(*) FROM " + tableName + " WHERE erpPk='" + c.ErpPk + "'")
			checkErr(err)
			for rows.Next() {
				_ = rows.Scan(&cptE)
			}
			if cptE == 1 {
				s := "SELECT COUNT(*) FROM " + tableName + " WHERE erpPk='" + c.ErpPk + "' AND content='" + c.Content + "'"
				rows, err := dbC.Query(s)
				if err != nil {
					fmt.Printf("err 101 %v\n", err)
					fmt.Printf("sql %v\n", s)
				}
				for rows.Next() {
					_ = rows.Scan(&cptE)
				}
				if cptE == 0 {

					_, err := stUpdate.Exec(c.Content, time.Now().UnixNano()/int64(time.Millisecond), c.ErpPk)
					checkErr(err)
					fmt.Printf("updateD %v\n", c.ErpPk)
				} else {
					_, err := stUpdateProcessed.Exec(c.ErpPk)
					checkErr(err)
				}
			} else {
				n := time.Now().UnixNano() / int64(time.Millisecond)
				_, err := stIns.Exec(c.Content, n, c.ErpPk, n, entry.Name)
				checkErr(err)
				fmt.Printf("not existing insert into %v\n", c.ErpPk)
			}
		}
	}

	stDelete, _ := dbC.Prepare("DELETE FROM " + tableName + " WHERE processedFromERP=0")
	defer stDelete.Close()
	_, err = stDelete.Exec()

	timeMSStop := time.Now().UnixNano() / int64(time.Millisecond)
	fmt.Printf("sync (%s) Time : %d - %d = %d\n", entry.Name, timeMSStop, timeMSStart, (timeMSStop - timeMSStart))
	fmt.Printf("Entries :%v\n", len(keys))
}

func getErpEntries() []ErpEntry {
	fmt.Printf("getErpEntries\n")
	st, err := dbC.Prepare(COUNT_ENTRY_ALL)
	defer st.Close()
	checkErr(err)

	rows, err := st.Query()
	checkErr(err)

	var cpt int
	for rows.Next() {
		_ = rows.Scan(&cpt)
	}
	result := make([]ErpEntry, cpt)

	st, err = dbC.Prepare(SELECT_ENTRY_ALL)
	checkErr(err)

	rows, err = st.Query()
	checkErr(err)
	i := 0
	for rows.Next() {
		o := &ErpEntry{}
		o.loadFromDbRow(rows)
		result[i] = *o
		i++
	}
	return result
}

func initDbEntry(db *sql.DB) {
	defer fmt.Printf("Init DB DONE! \n")

	// TABLE FOR ERP ENTRY
	sql := "CREATE TABLE IF NOT EXISTS `mid_db`.`admin_erp_entry` (`id` int(10) unsigned NOT NULL AUTO_INCREMENT,`erpId` int(10) unsigned NOT NULL DEFAULT '0',  `creationDate` datetime NOT NULL DEFAULT '0000-00-00 00:00:00',`sourceName` varchar(255) NOT NULL DEFAULT '',`name` varchar(255) NOT NULL DEFAULT '',PRIMARY KEY (`id`)) ENGINE=InnoDB DEFAULT CHARSET=latin1;"
	st, err := db.Prepare(sql)
	checkErr(err)
	_, err = st.Exec()
	checkErr(err)

}