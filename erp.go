package main

import (
	"database/sql"
	"fmt"
	"time"
)

type VisibleErp struct {
	CreationDate string
	TypeInt      int
	Type         string
	Name         string
	Value        string
	Entries      []*VisibleErpEntry
}


type Erp struct {
	CreationDate string
	TypeInt      int
	Type         string
	Name         string
	Value        string
	Id           int
	Sources      []ErpSource
	Entries      []ErpEntry
}

const(
	// ERP
	COUNT_ERP_ALL    = "SELECT COUNT(*) FROM admin_erp"
	SELECT_ERP_ALL   = "SELECT id, creationDate, typeInt, type, name, value FROM admin_erp"
	SELECT_ERP_BY_ID = "SELECT id, creationDate, typeInt, type, name, value FROM admin_erp WHERE id=?"
	INSERT_ERP       = "INSERT admin_erp SET creationdate=?, typeInt=?, type=?, name=?, value=?"
	UPDATE_ERP_BY_ID = "UPDATE admin_erp SET creationdate=?, typeInt=?, type=?, name=?, value=? WHERE id=?"
	DELETE_ERP_BY_ID = "DELETE FROM admin_erp WHERE id=?"
)

func (o *Erp) loadDb() {
	st, _ := dbC.Prepare(SELECT_ERP_BY_ID)
	defer st.Close()
	rows, err := st.Query(o.Id)
	if err != nil {
		fmt.Printf("err 03\n")
	}
	for rows.Next() {
		o.loadFromDbRow(rows)
	}
}

func (o *Erp) saveDb() {
	st, _ := dbC.Prepare(INSERT_ERP)
	defer st.Close()
	_, err := st.Exec(time.Now(), o.TypeInt, o.Type, o.Name, o.Value)
	checkErr(err)
}

func (o *Erp) updateDb() {
	st, _ := dbC.Prepare(UPDATE_ERP_BY_ID)
	defer st.Close()
	_, err := st.Exec(o.CreationDate, o.TypeInt, o.Type, o.Name, o.Value, o.Id)
	checkErr(err)
}

func (o *Erp) deleteDb() {
	l := getErpEntries()
	for _, val := range l {
		if val.ErpId == o.Id {
			val.deleteDb() // TODO Optimize this
		}
	}
	st, err := dbC.Prepare(DELETE_ERP_BY_ID)
	defer st.Close()
	checkErr(err)
	_, err = st.Exec(o.Id)
	checkErr(err)
}

func (p *Erp) HasSources() bool {
	return len(p.Sources) > 0
}

func (p *Erp) lazyLoadTables() error {
	fmt.Printf("Erp lazyLoadTables\n")
	if p.TypeInt == MYSQL_TYPE {
		desiredSchema := getMySqlSchema(p.Value)
		fmt.Printf("desiredSchema %v\n", desiredSchema)
		dbCErp, _ := sql.Open("mysql", p.Value)
		//dbCErp.SetMaxIdleConns(500)
		//dbCErp.SetMaxOpenConns(400)
		defer dbCErp.Close()
		st, err := dbCErp.Prepare(COUNT_ERP_MYSQL)
		defer st.Close()
		checkErr(err)
		rows, err := st.Query(desiredSchema)
		checkErr(err)

		var cpt int
		for rows.Next() {
			_ = rows.Scan(&cpt)
		}

		result := make([]ErpSource, cpt)
		st, err = dbC.Prepare(SELECT_ERP_MYSQL)
		checkErr(err)
		rows, err = st.Query(desiredSchema)
		checkErr(err)
		i := 0
		nameLoaded := false
		for rows.Next() {
			fmt.Printf("row.Next %v\n", i)
			nameLoaded = true
			e := &ErpSource{}
			e.ErpId = p.Id
			err := rows.Scan(&e.Name)
			if err != nil {
				fmt.Printf("err 04 %v\n", err)
				return err
			}
			e.loadUsed()
			result[i] = *e
			i++
		}
		if nameLoaded {
			fmt.Printf("loaded %v\n", nameLoaded)
			p.Sources = result
		} else {
			fmt.Printf("not loaded %v\n", nameLoaded)
			p.Sources = make([]ErpSource, 0)
		}
		return nil
	} else {
		result := make([]ErpSource, 1)
		result[0].Name = "ERP Type not implemented yet"
		p.Sources = result
		return nil
	}
}

func (p *Erp) loadFromDbRow(rows *sql.Rows) error {
	err := rows.Scan(&p.Id, &p.CreationDate, &p.TypeInt, &p.Type, &p.Name, &p.Value)
	if err != nil {
		fmt.Printf("err 04\n")
		return err
	}
	return nil
}

func (p *Erp) loadErpEntries() {
	fmt.Printf("Erp loadErpEntries\n")
	st, err := dbC.Prepare(COUNT_ENTRY_BY_ERP)
	defer st.Close()
	checkErr(err)

	rows, err := st.Query(p.Id)
	checkErr(err)

	var cpt int
	for rows.Next() {
		_ = rows.Scan(&cpt)
	}

	result := make([]ErpEntry, cpt)

	st, err = dbC.Prepare(SELECT_ENTRY_BY_ERP)
	checkErr(err)

	rows, err = st.Query(p.Id)
	checkErr(err)
	i := 0
	for rows.Next() {
		o := &ErpEntry{}
		o.loadFromDbRow(rows)
		result[i] = *o
		i++
	}
	p.Entries = result
}

func getErps() []Erp {
	fmt.Printf("getErps\n")
	st, err := dbC.Prepare(COUNT_ERP_ALL)
	defer st.Close()
	checkErr(err)

	rows, err := st.Query()
	checkErr(err)

	var cpt int
	for rows.Next() {
		_ = rows.Scan(&cpt)
	}
	result := make([]Erp, cpt)

	st, err = dbC.Prepare(SELECT_ERP_ALL)
	checkErr(err)

	rows, err = st.Query()
	checkErr(err)
	i := 0
	for rows.Next() {
		o := &Erp{}
		o.loadFromDbRow(rows)
		result[i] = *o
		i++
	}
	return result
}

func initDbErp(db *sql.DB) {

	// TABLE FOR ERP
	sql := "CREATE TABLE IF NOT EXISTS `mid_db`.`admin_erp` (`id` INTEGER UNSIGNED NOT NULL AUTO_INCREMENT,`creationDate` DATETIME NOT NULL DEFAULT 0,`typeInt` INTEGER UNSIGNED NOT NULL DEFAULT 0,`type` VARCHAR(45) NOT NULL DEFAULT '',`name` VARCHAR(45) NOT NULL DEFAULT '',`value` longtext, PRIMARY KEY(`id`))ENGINE = InnoDB;"
	st, err := db.Prepare(sql)
	defer st.Close()
	checkErr(err)
	_, err = st.Exec()
	checkErr(err)
}
