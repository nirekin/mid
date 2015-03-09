package main

import (
	
)

type ErpRField struct {
	ErpEntryId int
	Name       string
	Used       int
}

func (o *ErpRField) loadUsed() {
	st, err := dbC.Prepare(COUNT_FIELD_USED)
	defer st.Close()
	checkErr(err)
	rows, err := st.Query(o.ErpEntryId, o.Name)
	checkErr(err)
	for rows.Next() {
		err = rows.Scan(&o.Used)
	}
}
