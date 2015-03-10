package main

import ()

type ErpRField struct {
	ErpEntryId int
	Name       string
	Used       int
}

func (o *ErpRField) loadUsed() error {
	st, err := dbC.Prepare(COUNT_FIELD_USED)
	if err != nil {
		return err
	} else {
		defer st.Close()
	}
	rows, err := st.Query(o.ErpEntryId, o.Name)
	if err != nil {
		return err
	}
	for rows.Next() {
		err = rows.Scan(&o.Used)
	}
	return nil
}
