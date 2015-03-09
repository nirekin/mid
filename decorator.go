package main

import (
	"database/sql"
	"strings"
	"fmt"
	"encoding/json"
	"strconv"
	"sort"
)

type VisibleDecorator struct {
	Name         string
	Description  string
	SortingOrder int
	Params       string
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
	Name         string
	Description  string
	SortingOrder int
	Params       string
	Id           int
	DecoratorId  int
	SyncFieldId  int
}

type DecoratorMap map[int]*PredefinedDecorator

const(

	// DECORATOR
	COUNT_DECORATOR_BY_FIELD  = "SELECT COUNT(*) FROM admin_sync_field_decorator WHERE syncFieldId=?"
	SELECT_DECORATOR_BY_FIELD = "SELECT id, decoratorId, syncFieldId, sortingOrder, params FROM admin_sync_field_decorator WHERE syncFieldId=? ORDER BY sortingOrder"
	INSERT_DECORATOR          = "INSERT admin_sync_field_decorator SET decoratorId=?, syncFieldId=?, sortingOrder=?, params=?"
	UPDATE_DECORATOR_BY_ID    = "UPDATE admin_sync_field_decorator SET decoratorId=?, syncFieldId=?, sortingOrder=?, params=? WHERE id=?"
	DELETE_DECORATOR_BY_ID    = "DELETE FROM admin_sync_field_decorator WHERE id=?"
	DELETE_DECORATOR_BY_FIELD = "DELETE FROM admin_sync_field_decorator WHERE syncFieldId=?"
)

func (o *Decorator) loadFromDbRow(rows *sql.Rows) error {
	err := rows.Scan(&o.Id, &o.DecoratorId, &o.SyncFieldId, &o.SortingOrder, &o.Params)
	if err != nil {
		fmt.Printf("err 04\n")
		return err
	}
	return nil
}

func (o *Decorator) getParamValue(name string) string {
	fmt.Printf("Decorator getParamValue\n")
	var f interface{}
	_ = json.Unmarshal([]byte(o.Params), &f)
	m := f.(map[string]interface{})
	return fmt.Sprintf("%v", m[name])
}

func (o *Decorator) saveDb() {
	fmt.Printf("Decorator saveDb\n")
	st, _ := dbC.Prepare(INSERT_DECORATOR)
	defer st.Close()
	_, err := st.Exec(o.DecoratorId, o.SyncFieldId, o.SortingOrder, o.Params)
	checkErr(err)
}

func (o *Decorator) updateDb() {
	fmt.Printf("Decorator updateDB\n")
	st, _ := dbC.Prepare(UPDATE_DECORATOR_BY_ID)
	defer st.Close()
	_, err := st.Exec(o.DecoratorId, o.SyncFieldId, o.SortingOrder, o.Params, o.Id)
	checkErr(err)
}

func (o *Decorator) deleteDb() {
	fmt.Printf("Decorator deleteDb\n")
	deleteDecoratorById(o.Id)
}

func deleteDecoratorByField(id int) {
	fmt.Printf("deleteDecoratorByFild\n")
	st, err := dbC.Prepare(DELETE_DECORATOR_BY_FIELD)
	defer st.Close()
	checkErr(err)
	_, err = st.Exec(id)
	checkErr(err)
}

func deleteDecoratorById(id int) {
	fmt.Printf("Decorator deleteDb\n")
	st, err := dbC.Prepare(DELETE_DECORATOR_BY_ID)
	defer st.Close()
	checkErr(err)
	_, err = st.Exec(id)
	checkErr(err)
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
		fmt.Printf("1500 len %v\n", ln)
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
	cpt := len(decorators)
	result := make([]*PredefinedDecorator, cpt)

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
