package main

import (
	
)

type ErpSource struct {
	ErpId int
	Name  string
	Used  int
}

func (o *ErpSource) loadUsed() {
	st, err := dbC.Prepare(COUNT_ENTRY_USED)
	defer st.Close()
	checkErr(err)
	rows, err := st.Query(o.ErpId, o.Name)
	checkErr(err)
	for rows.Next() {
		err = rows.Scan(&o.Used)
	}
}