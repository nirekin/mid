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

const (
	// ERP
	ERP_SELECT_FIELDS = "SELECT id, creationDate, typeInt, type, name, value "
	ERP_INSERT_UPDATE = " creationdate=?, typeInt=?, type=?, name=?, value=?"

	SELECT_ERP_ALL   = ERP_SELECT_FIELDS + "FROM admin_erp"
	SELECT_ERP_BY_ID = ERP_SELECT_FIELDS + "FROM admin_erp WHERE id=?"
	INSERT_ERP       = "INSERT admin_erp SET " + ERP_INSERT_UPDATE
	UPDATE_ERP_BY_ID = "UPDATE admin_erp SET " + ERP_INSERT_UPDATE + " WHERE id=?"
	DELETE_ERP_BY_ID = "DELETE FROM admin_erp WHERE id=?"

	SELECT_ERP_MYSQL = "SELECT TABLE_NAME FROM information_schema.tables WHERE TABLE_SCHEMA = ?"
)

func (o *Erp) loadDb() error {
	st, err := dbC.Prepare(SELECT_ERP_BY_ID)
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

func (o *Erp) saveDb() error {
	st, err := dbC.Prepare(INSERT_ERP)
	if err != nil {
		return err
	} else {
		defer st.Close()
	}
	_, err = st.Exec(time.Now(), o.TypeInt, o.Type, o.Name, o.Value)
	if err != nil {
		return err
	}
	return nil
}

func (o *Erp) updateDb() error {
	st, err := dbC.Prepare(UPDATE_ERP_BY_ID)
	if err != nil {
		return err
	} else {
		defer st.Close()
	}
	_, err = st.Exec(o.CreationDate, o.TypeInt, o.Type, o.Name, o.Value, o.Id)
	if err != nil {
		return err
	}
	return nil
}

func (o *Erp) deleteDb() error {
	l, _ := getErpEntries()
	for _, val := range l {
		if val.ErpId == o.Id {
			val.deleteDb() // TODO Optimize this
		}
	}
	st, err := dbC.Prepare(DELETE_ERP_BY_ID)
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

func (p *Erp) HasSources() bool {
	return len(p.Sources) > 0
}

func (p *Erp) lazyLoadTables() error {
	if p.TypeInt == MYSQL_TYPE {
		desiredSchema := getMySqlSchema(p.Value)
		dbCErp, err := sql.Open("mysql", p.Value)
		if err != nil {
			return err
		}
		defer dbCErp.Close()

		var tResult [10]ErpSource
		result := tResult[0:0]

		st, err := dbC.Prepare(SELECT_ERP_MYSQL)
		if err != nil {
			return err
		}
		rows, err := st.Query(desiredSchema)
		if err != nil {
			return err
		}

		nameLoaded := false
		for rows.Next() {
			nameLoaded = true
			e := ErpSource{}
			e.ErpId = p.Id
			err := rows.Scan(&e.Name)
			if err != nil {
				return err
			}
			e.loadUsed()
			result = append(result, e)
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
	return nil
}

func (p *Erp) loadFromDbRow(rows *sql.Rows) error {
	err := rows.Scan(&p.Id, &p.CreationDate, &p.TypeInt, &p.Type, &p.Name, &p.Value)
	if err != nil {
		return err
	}
	return nil
}

func (p *Erp) loadErpEntries() error {
	var tResult [10]ErpEntry
	result := tResult[0:0]

	st, err := dbC.Prepare(SELECT_ENTRY_BY_ERP)
	if err != nil {
		return err
	}

	rows, err := st.Query(p.Id)
	if err != nil {
		return err
	}
	for rows.Next() {
		o := ErpEntry{}
		o.loadFromDbRow(rows)
		result = append(result, o)
	}
	p.Entries = result
	return nil
}

func getErps() ([]Erp, error) {
	var tResult [10]Erp
	result := tResult[0:0]

	st, err := dbC.Prepare(SELECT_ERP_ALL)
	if err != nil {
		return nil, err
	}

	rows, err := st.Query()
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		o := Erp{}
		o.loadFromDbRow(rows)
		result = append(result, o)
	}
	return result, nil
}

func initDbErp(db *sql.DB) error {
	sql := "CREATE TABLE IF NOT EXISTS `mid_db`.`admin_erp` (`id` INTEGER UNSIGNED NOT NULL AUTO_INCREMENT,`creationDate` DATETIME NOT NULL DEFAULT 0,`typeInt` INTEGER UNSIGNED NOT NULL DEFAULT 0,`type` VARCHAR(45) NOT NULL DEFAULT '',`name` VARCHAR(45) NOT NULL DEFAULT '',`value` longtext, PRIMARY KEY(`id`))ENGINE = InnoDB;"
	st, err := db.Prepare(sql)
	defer st.Close()
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
