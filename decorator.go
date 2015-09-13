package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

type VisibleDecorator struct {
	Name         string `xml:"name"`
	Description  string `xml:"description"`
	SortingOrder int    `xml:"sortingOrder"`
	Params       string `xml:"params"`
}

type PredefinedDecoratorParam struct {
	Name        string
	Type        string
	Description string
}

type PredefinedDecorator struct {
	Id          int
	Name        string
	Description string
	Params      []PredefinedDecoratorParam
	Template    string
	FDecorate   func(pd *PredefinedDecorator, d *Decorator, s string) string
}

type Decorator struct {
	DBEntity
	Name         string
	Description  string
	SortingOrder int
	Params       string
	DecoratorId  int
	SyncFieldId  int
}

type DecoratorMap map[int]*PredefinedDecorator

const (
	DECORATOR_TABLE_NAME = "admin_sync_field_decorator"

	// DECORATOR
	DECORATOR_SELECT_FIELDS = "SELECT id, decoratorId, syncFieldId, sortingOrder, params "
	DECORATOR_INSERT_UPDATE = " decoratorId=?, syncFieldId=?, sortingOrder=?, params=?"

	SELECT_DECORATOR_BY_FIELD = DECORATOR_SELECT_FIELDS + " FROM " + DECORATOR_TABLE_NAME + " WHERE syncFieldId=? ORDER BY sortingOrder"
	INSERT_DECORATOR          = "INSERT " + DECORATOR_TABLE_NAME + " SET " + DECORATOR_INSERT_UPDATE
	UPDATE_DECORATOR_BY_ID    = "UPDATE " + DECORATOR_TABLE_NAME + " SET " + DECORATOR_INSERT_UPDATE + " WHERE id=?"
)

func (o *Decorator) loadFromDbRow(rows *sql.Rows) error {
	err := rows.Scan(&o.Id, &o.DecoratorId, &o.SyncFieldId, &o.SortingOrder, &o.Params)
	if err != nil {
		return err
	}
	return nil
}

func (o *Decorator) getParamValue(name string) string {
	var f interface{}
	_ = json.Unmarshal([]byte(o.Params), &f)
	m := f.(map[string]interface{})
	return fmt.Sprintf("%v", m[name])
}

func (o *Decorator) saveDb() error {
	st, err := dbC.Prepare(INSERT_DECORATOR)
	if err != nil {
		return err
	} else {
		defer st.Close()
	}
	_, err = st.Exec(o.DecoratorId, o.SyncFieldId, o.SortingOrder, o.Params)
	if err != nil {
		return err
	}
	return nil
}

func (o *Decorator) updateDb() error {
	st, err := dbC.Prepare(UPDATE_DECORATOR_BY_ID)
	if err != nil {
		return err
	} else {
		defer st.Close()
	}
	_, err = st.Exec(o.DecoratorId, o.SyncFieldId, o.SortingOrder, o.Params, o.Id)
	if err != nil {
		return err
	}
	return nil
}

func (o *Decorator) deleteDb() error {
	err := delete(o)
	if err != nil {
		return err
	}
	return nil
}

func (self *PredefinedDecorator) Decorate(d Decorator, s string) string {
	return self.FDecorate(self, &d, s)
}

func initDecorators() {

	decorators = make(map[int]*PredefinedDecorator)

	// * * * *
	d := &PredefinedDecorator{Id: 1000, Name: "Length", Description: "Get The string length"}
	d.FDecorate = func(pd *PredefinedDecorator, d *Decorator, s string) string {
		return strconv.Itoa(len(s))
	}
	decorators.add(d)

	// * * * *
	d = &PredefinedDecorator{Id: 1001, Name: "Upper Case", Description: "Convert to upper case"}
	d.FDecorate = func(pd *PredefinedDecorator, d *Decorator, s string) string {
		return strings.ToUpper(s)
	}
	decorators.add(d)

	// * * * *
	d = &PredefinedDecorator{Id: 1002, Name: "Lower Case", Description: "Convert to lower case"}
	d.FDecorate = func(pd *PredefinedDecorator, d *Decorator, s string) string {
		return strings.ToLower(s)
	}
	decorators.add(d)

	// * * * *
	d = &PredefinedDecorator{Id: 1003, Name: "Reverse", Description: "Reverse the string content"}
	d.FDecorate = func(pd *PredefinedDecorator, d *Decorator, s string) string {
		runes := []rune(s)
		for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
			runes[i], runes[j] = runes[j], runes[i]
		}
		str := string(runes)
		return str
	}
	decorators.add(d)

	// * * * *
	d = &PredefinedDecorator{Id: 1013, Name: "Replace substring", Description: "Replace all instances of a substring"}
	d.Params = make([]PredefinedDecoratorParam, 2)
	d.Params[0].Name = "oldstring"
	d.Params[0].Type = "string"
	d.Params[0].Description = "The substring to replace"

	d.Params[1].Name = "newstring"
	d.Params[1].Type = "string"
	d.Params[1].Description = "The string to use as replacement"

	d.FDecorate = func(pd *PredefinedDecorator, d *Decorator, s string) string {
		oldstring := d.getParamValue("oldstring")
		newstring := d.getParamValue("newstring")
		return strings.Replace(s, oldstring, newstring, -1)
	}
	decorators.add(d)

	// * * * *
	d = &PredefinedDecorator{Id: 1014, Name: "Replace \"1\"", Description: "If == \"1\" then \"true\", otherwise \"false\""}
	d.FDecorate = func(pd *PredefinedDecorator, d *Decorator, s string) string {
		if "1" == s {
			return "true"
		} else {
			return "false"
		}
	}
	decorators.add(d)

	// * * * *
	d = &PredefinedDecorator{Id: 1015, Name: "Replace \"0\"", Description: "If \"0\" then \"false\", otherwise \"true\""}
	d.FDecorate = func(pd *PredefinedDecorator, d *Decorator, s string) string {
		if "0" == s {
			return "false"
		} else {
			return "true"
		}
	}
	decorators.add(d)

	// * * * *
	d = &PredefinedDecorator{Id: 1500, Name: "Trim X left", Description: "Remove X caracters from the left", Template: "./template/listParams.html"}
	d.Params = make([]PredefinedDecoratorParam, 1)
	d.Params[0].Name = "len"
	d.Params[0].Type = "int"
	d.Params[0].Description = "The number of caracter to remove"

	d.FDecorate = func(pd *PredefinedDecorator, d *Decorator, s string) string {
		ln, _ := strconv.Atoi(d.getParamValue("len"))
		if len(s) > ln-1 {
			return s[ln:len(s)]
		} else if len(s) <= ln {
			return ""
		} else {
			return s
		}
	}
	decorators.add(d)

	// * * * *
	d = &PredefinedDecorator{Id: 1600, Name: "Replace BY OR", Description: "Replace the string by one or another constant", Template: "./template/listParams.html"}
	d.Params = make([]PredefinedDecoratorParam, 3)
	d.Params[0].Name = "teststr"
	d.Params[0].Type = "string"
	d.Params[0].Description = "the string to test"

	d.Params[1].Name = "testok"
	d.Params[1].Type = "string"
	d.Params[1].Description = "the string to use if OK"

	d.Params[2].Name = "testko"
	d.Params[2].Type = "string"
	d.Params[2].Description = "te string to use if KO"

	d.FDecorate = func(pd *PredefinedDecorator, d *Decorator, s string) string {
		teststr := d.getParamValue("teststr")
		testok := d.getParamValue("testok")
		testko := d.getParamValue("testko")
		if teststr == s {
			return testok
		} else {
			return testko
		}
	}
	decorators.add(d)

	// * * * *
	d = &PredefinedDecorator{Id: 1601, Name: "Replace if true", Description: "Replace the string only if its equals to a constant", Template: "./template/listParams.html"}
	d.Params = make([]PredefinedDecoratorParam, 2)
	d.Params[0].Name = "teststr"
	d.Params[0].Type = "string"
	d.Params[0].Description = "the string to test"

	d.Params[1].Name = "testok"
	d.Params[1].Type = "string"
	d.Params[1].Description = "the string to use if OK"

	d.FDecorate = func(pd *PredefinedDecorator, d *Decorator, s string) string {
		teststr := d.getParamValue("teststr")
		testok := d.getParamValue("testok")

		if teststr == s {
			return testok
		} else {
			return s
		}
	}
	decorators.add(d)

	// * * * *
	d = &PredefinedDecorator{Id: 1602, Name: "Replace if false", Description: "Replace the string only if its not equals to a constant", Template: "./template/listParams.html"}
	d.Params = make([]PredefinedDecoratorParam, 2)
	d.Params[0].Name = "teststr"
	d.Params[0].Type = "string"
	d.Params[0].Description = "the string to test"

	d.Params[1].Name = "testko"
	d.Params[1].Type = "string"
	d.Params[1].Description = "the string to use if KO"

	d.FDecorate = func(pd *PredefinedDecorator, d *Decorator, s string) string {
		teststr := d.getParamValue("teststr")
		testko := d.getParamValue("testko")

		if teststr == s {
			return s
		} else {
			return testko
		}
	}
	decorators.add(d)

	// * * * *
	d = &PredefinedDecorator{Id: 1610, Name: "Add Before", Description: "Add a string before", Template: "./template/listParams.html"}
	d.Params = make([]PredefinedDecoratorParam, 1)
	d.Params[0].Name = "strtoadd"
	d.Params[0].Type = "string"
	d.Params[0].Description = "The string to add"
	d.FDecorate = func(pd *PredefinedDecorator, d *Decorator, s string) string {
		strtoadd := d.getParamValue("strtoadd")
		return strtoadd + s
	}
	decorators.add(d)

	// * * * *
	d = &PredefinedDecorator{Id: 1611, Name: "Add After", Description: "Add a string after", Template: "./template/listParams.html"}
	d.Params = make([]PredefinedDecoratorParam, 1)
	d.Params[0].Name = "strtoadd"
	d.Params[0].Type = "string"
	d.Params[0].Description = "The string to add"
	d.FDecorate = func(pd *PredefinedDecorator, d *Decorator, s string) string {
		strtoadd := d.getParamValue("strtoadd")
		return s + strtoadd
	}
	decorators.add(d)

	// * * * *
	d = &PredefinedDecorator{Id: 1700, Name: "Keep X from left", Description: "Keep X characters from the lest of the string", Template: "./template/listParams.html"}
	d.Params = make([]PredefinedDecoratorParam, 2)
	d.Params[0].Name = "strlen"
	d.Params[0].Type = "int"
	d.Params[0].Description = "The number of characters to keep"

	d.Params[1].Name = "strsuffix"
	d.Params[1].Type = "string"
	d.Params[1].Description = "The suffix to add to truncated string"

	d.FDecorate = func(pd *PredefinedDecorator, d *Decorator, s string) string {
		strlen := d.getParamValue("strlen")
		l, _ := strconv.Atoi(strlen)
		strsuffix := d.getParamValue("strsuffix")

		if len(s) <= l {
			return s
		} else {
			return s[:l] + strsuffix
		}
	}
	decorators.add(d)
}

func (m *DecoratorMap) add(d *PredefinedDecorator) {
	decorators[d.Id] = d
}

func (o *Decorator) decorate(s string) string {
	pDef := decorators[o.DecoratorId]
	return pDef.FDecorate(pDef, o, s)
}

func getPredefinedDecorator() []*PredefinedDecorator {
	result := make([]*PredefinedDecorator, len(decorators))

	var keys []int
	for k := range decorators {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	for i, k := range keys {
		result[i] = decorators[k]
	}
	return result
}

func initDbDecorator(db *sql.DB) error {
	sql := "CREATE TABLE IF NOT EXISTS `mid_db`.`admin_sync_field_decorator` (`id` int(10) unsigned NOT NULL AUTO_INCREMENT, `decoratorId` int(10) unsigned NOT NULL DEFAULT '0', `syncFieldId` int(10) unsigned NOT NULL DEFAULT '0', `sortingOrder` int(10) unsigned NOT NULL DEFAULT '0', `params` varchar(255) NOT NULL DEFAULT '',PRIMARY KEY (`id`)) ENGINE=InnoDB DEFAULT CHARSET=latin1;"
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

func (e Decorator) getTableName() string {
	return DECORATOR_TABLE_NAME
}

func (e Decorator) getSelectFields() string {
	return DECORATOR_SELECT_FIELDS
}
