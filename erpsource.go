package main

import ()

type ErpSource struct {
	ErpId int
	Name  string
	Used  int
}

func (o *ErpSource) loadUsed() error {
	st, err := dbC.Prepare(COUNT_ENTRY_USED)
	if err != nil {
		return err
	} else {
		defer st.Close()
	}
	rows, err := st.Query(o.ErpId, o.Name)
	if err != nil {
		return err
	}
	for rows.Next() {
		err = rows.Scan(&o.Used)
	}
	return nil
}
