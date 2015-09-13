package main

import (
	"database/sql"
	"fmt"
)

type Loader interface {
	loadDb() error
	setId(i int)
	getId() int
}

type Deleter interface {
	getId() int
	getTableName() string
	getChildren() []Deleter
	getSelectFields() string
}

type DBEntity struct {
	Id       int
	Children []Deleter
}

const (
	DELETE_ENTITY = "DELETE FROM %v WHERE id=%d"
	SELECT_ALL    = "%s FROM %v"
	SELECT_ID     = "%s FROM %v WHERE id=%d"
)

func (e DBEntity) getId() int {
	return e.Id
}

func (e DBEntity) setId(i int) {
	fmt.Printf("setId1 %v\n", i)
	e.Id = i
	fmt.Printf("setId2 %v\n", e.Id)
}

func (e DBEntity) getChildren() []Deleter {
	return e.Children
}

func delete(d Deleter) error {
	s := fmt.Sprintf(DELETE_ENTITY, d.getTableName(), d.getId())

	if d.getChildren() != nil {
		for _, val := range d.getChildren() {
			if err := delete(val); err != nil {
				return err
			}
		}
	}

	if st, err := dbC.Prepare(s); err == nil {
		defer st.Close()
		if _, err = st.Exec(); err != nil {
			return err
		}
		return nil
	} else {
		return err
	}
}

func selectAll(d Deleter) (*sql.Rows, error) {
	s := fmt.Sprintf(SELECT_ALL, d.getSelectFields(), d.getTableName())
	if st, err := dbC.Prepare(s); err == nil {
		if rows, err := st.Query(); err == nil {
			return rows, nil
		} else {
			return nil, err
		}
	} else {
		return nil, err
	}
}

func selectById(d Deleter) (*sql.Rows, error) {
	s := fmt.Sprintf(SELECT_ID, d.getSelectFields(), d.getTableName(), d.getId())
	if st, err := dbC.Prepare(s); err == nil {
		if rows, err := st.Query(); err == nil {
			return rows, nil
		} else {
			return nil, err
		}
	} else {
		return nil, err
	}
}
