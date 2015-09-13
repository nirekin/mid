package main

/*
Install of go-sql-driver


go get github.com/go-sql-driver/mysql
go get github.com/mxk/go-sqlite/sqlite3   --> mingv-w64	github.com/mxk/go-sqlite/sqlite3

DOC https://gobyexample.com/time-formatting-parsing
t := fmt.Sprintf("%d-%02d-%02d %02d:%02d:%02d", fromTime.Year(), fromTime.Month(), fromTime.Day(), fromTime.Hour(), fromTime.Minute(), fromTime.Second())

TODO :

- Encode special characters in something like UTF8...
- Test volume speed
- Select criteria para MYSQL

- Ping for erp
- JS validation in all forms
- Entry hierarchy mother/child
- Log for extraction --> SyncEvent
- Log for error
- Synchronizer by entry o for all entries ?

- Export config json / XML
- Consume config json  / XML

- Master slave
	- register slaves
	- push changes
	- find gap
	- fix gap

- Add Mutex while synchronizing entries...  ( if case of, for example, a frequency lower than the processing time )

*/

import (
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"html/template"
	"net/http"
	"runtime"
	"strconv"
	"strings"
)

type ExtractedContent struct {
	ErpEntryId int
	ErpPk      string
	Content    string
}

type LoadedContentLine struct {
	Name         string
	ErpPk        string
	Content      string
	CreationDate string
	LastUpdate   string
}

type RequestAddContent struct {
	FieldId     int
	DecoratorId int
	PDecorator  *PredefinedDecorator
}

type ExtractedContentMap map[string]*ExtractedContent

const (

	// MYSQL
	MYSQL_TYPE       = 1
	MYSQL_TYPE_SPLIT = "__/$/__"
	MYSQL_TYPE_EMPTY = "__/#/__"
	ACCESS_TYPE      = 2

	db_url = "root:admin@tcp(localhost:3306)/mid_db"
)

var dbC *sql.DB
var cptGen int
var decorators DecoratorMap

type rFInt func(r *http.Request) (int, error)
type rFString func(r *http.Request) string

var iFId = readFormInt("Id")
var iFField = readFormInt("FieldId")
var iFErp = readFormInt("ErpId")
var iFEntry = readFormInt("EntryId")
var iFDec = readFormInt("DecId")
var iFDec2 = readFormInt("DecoratorId")
var iFBlock = readFormInt("BlockSize")
var iFLimit = readFormInt("Limit")

var sFSourceName = readFormString("SourceName")
var sFName = readFormString("Name")
var sFValue = readFormString("Value")
var sFFieldName = readFormString("FieldName")
var sFEntry = readFormString("EntryId")
var sFTextContent = readFormString("TestContent")
var sFId = readFormString("Id")
var sFJSonName = readFormString("JsonName")
var sFField = readFormString("FieldId")
var sFLikeOnPk = readFormString("LikeOnErpPk")
var sFLikeOnContent = readFormString("LikeOnContent")

var jsonHtmlTmpl = template.Must(template.New("jsonHtml").Parse(`
	<pre>{{.}}</pre>
`))

var xmlHtmlTmpl = template.Must(template.New("xmlHtml").Parse(`
	<pre>{{.}}</pre>
`))

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	defer dbC.Close()
	defer fmt.Printf("stopped \n")

	go StartSync()

	// ERP ---
	http.HandleFunc("/admin/", adminHandler)
	http.HandleFunc("/configJSon/", configJSonHandler)
	http.HandleFunc("/configXml/", configXmlHandler)
	http.HandleFunc("/importConfigJSon/", importConfigJSonHandler)

	http.HandleFunc("/inspect/", inspectHandler)

	http.HandleFunc("/erpsources/", erpsourcesHandler)
	http.HandleFunc("/addMySQL/", addMySQLHandler)
	http.HandleFunc("/updateMySQL/", updateMySQLHandler)

	http.HandleFunc("/addAccess/", addAccessHandler)
	http.HandleFunc("/createMySQL/", createMySQLHandler)
	http.HandleFunc("/erpListTables/", erpListTablesHandler)
	http.HandleFunc("/deleteErp/", deleteErpHandler)
	http.HandleFunc("/editErp/", editErpHandler)

	// SOURCES ---
	http.HandleFunc("/erpentries/", erpentriesHandler)
	http.HandleFunc("/createErpEntry/", createErpEntryHandler)
	http.HandleFunc("/editErpEntry/", editErpEntryHandler)
	http.HandleFunc("/saveErpEntry/", saveErpEntryHandler)
	http.HandleFunc("/updateErpEntry/", updateErpEntryHandler)

	http.HandleFunc("/deleteErpEntry/", deleteErpEntryHandler)
	http.HandleFunc("/syncErpEntry/", syncErpEntryHandler)
	http.HandleFunc("/pingAsyncErpEntry/", pingAsyncErpEntryHandler)
	http.HandleFunc("/pingAsyncTestErpEntry/", pingAsyncTestErpEntryHandler)

	// FIELDS ---
	http.HandleFunc("/erpListFields/", erpListFieldsHandler)
	http.HandleFunc("/createSyncField/", createSyncFieldHandler)
	http.HandleFunc("/deleteSyncField/", deleteSyncFielddHandler)
	http.HandleFunc("/editSyncField/", editSyncFieldHandler)
	http.HandleFunc("/updateSyncField/", updateSyncFieldHandler)

	// DECORATORS
	http.HandleFunc("/addDecorator/", addDecoratorHandler)
	http.HandleFunc("/requestAddDecorator/", requestAddDecoratorHandler)
	http.HandleFunc("/requestAddDecoratoParam/", requestAddDecoratorParamHandler)
	http.HandleFunc("/deleteDecorator/", deleteDecoratorHandler)
	http.HandleFunc("/deleteDecoratorInAdd/", deleteDecoratorInAddHandler)

	http.Handle("/", http.FileServer(http.Dir("./resources")))
	http.Handle("/resources/", http.StripPrefix("/resources/", http.FileServer(http.Dir("resources"))))

	fmt.Println("ListenAndServe")
	http.ListenAndServe(":8090", nil)
}

func init() {
	defer fmt.Printf("Init DONE\n")
	dbC, _ = sql.Open("mysql", db_url)
	initDb(dbC)
	initDecorators()
}

func erpsourcesHandler(w http.ResponseWriter, r *http.Request) {
	t, _ := template.ParseFiles("./template/erpsources.html")
	all, err := getErps()
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	t.Execute(w, all)
}

func erpentriesHandler(w http.ResponseWriter, r *http.Request) {
	t, _ := template.ParseFiles("./template/erpentries.html")
	all, err := getErpEntries()
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	t.Execute(w, all)
}

var loadErpUrl = loadIdInt(readIntUrl, Erp{})

func erpListTablesHandler(w http.ResponseWriter, r *http.Request) {
	t, _ := template.ParseFiles("./template/erpListTables.html")
	//i, _ := readIntUrl(r)
	//erp := &Erp{DBEntity: DBEntity{Id: i}}
	//err := erp.loadDb()
	erp, err := loadErpUrl(r)
	fmt.Printf("Erp %v\n", erp)
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	//err = erp.lazyLoadTables()
	//if err != nil {
	//	fmt.Printf("%v\n", err)
	//}
	t.Execute(w, erp)
}

func erpListFieldsHandler(w http.ResponseWriter, r *http.Request) {
	t, _ := template.ParseFiles("./template/erpListFields.html")
	i, _ := readIntUrl(r)
	ent := &ErpEntry{DBEntity: DBEntity{Id: i}}
	err := ent.loadDb()
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	err = ent.lazyLoadRFields()
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	t.Execute(w, ent)
}

func createErpEntryHandler(w http.ResponseWriter, r *http.Request) {
	t, _ := template.ParseFiles("./template/createErpEntry.html")
	o := &ErpEntry{}
	i, _ := iFId(r)
	o.ErpId = i
	o.SourceName = sFSourceName(r)
	t.Execute(w, o)
}

func saveErpEntryHandler(w http.ResponseWriter, r *http.Request) {
	o := &ErpEntry{}
	o.ErpId, _ = iFErp(r)
	o.SourceName = sFSourceName(r)
	o.Name = sFName(r)
	err := o.saveDb()
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	o.createImportationTableName()
	http.Redirect(w, r, "/erpentries", http.StatusFound)
}

func updateErpEntryHandler(w http.ResponseWriter, r *http.Request) {
	i, _ := iFErp(r)
	ent := &ErpEntry{DBEntity: DBEntity{Id: i}}
	err := ent.loadDb()
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	ent.Name = sFName(r)
	bs, _ := iFBlock(r)
	ent.BlockSize = bs
	err = ent.updateDb()
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	http.Redirect(w, r, "/erpentries", http.StatusFound)
}

func pingAsyncErpEntryHandler(w http.ResponseWriter, r *http.Request) {
	i, _ := readIntUrl(r)
	ent := &ErpEntry{DBEntity: DBEntity{Id: i}}
	err := ent.loadDb()
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	lines, _ := ent.ping(20)
	for _, val := range lines {
		w.Write([]byte(val + "<BR>"))
	}
}

func pingAsyncTestErpEntryHandler(w http.ResponseWriter, r *http.Request) {
	idf, _ := iFField(r)
	f := &SyncField{DBEntity: DBEntity{Id: idf}}
	err := f.loadDb()
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	testContent := sFTextContent(r)

	mapD := map[string]string{}
	fN, val := f.decorate(testContent)
	mapD[fN] = val
	outJson, _ := json.Marshal(mapD)
	w.Write([]byte(testContent + "<BR>"))
	w.Write([]byte(string(outJson) + "<BR>"))
}

func syncErpEntryHandler(w http.ResponseWriter, r *http.Request) {
	i, _ := readIntUrl(r)
	ent := &ErpEntry{DBEntity: DBEntity{Id: i}}
	err := ent.loadDb()
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	err = synchronize(*ent)
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	http.Redirect(w, r, "/erpentries", http.StatusFound)
}

func deleteErpEntryHandler(w http.ResponseWriter, r *http.Request) {
	i, _ := readIntUrl(r)
	ent := &ErpEntry{DBEntity: DBEntity{Id: i}}
	err := ent.loadDb() // TODO check if the load is required here
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	err = ent.deleteDb()
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	http.Redirect(w, r, "/erpentries", http.StatusFound)
}

func deleteSyncFielddHandler(w http.ResponseWriter, r *http.Request) {
	idf, _ := iFField(r)
	f := &SyncField{DBEntity: DBEntity{Id: idf}}
	err := f.loadDb() // TODO check id the load is required here
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	err = f.deleteDb()
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	http.Redirect(w, r, "/editErpEntry/"+sFEntry(r), http.StatusFound)
}

func editSyncFieldHandler(w http.ResponseWriter, r *http.Request) {
	t, _ := template.ParseFiles("./template/editSyncField.html")
	i, _ := readIntUrl(r)
	f := &SyncField{DBEntity: DBEntity{Id: i}}
	err := f.loadDb()
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	t.Execute(w, f)
}

func updateSyncFieldHandler(w http.ResponseWriter, r *http.Request) {
	fieldId := sFId(r)
	jsonName := sFJSonName(r)
	i, _ := strconv.Atoi(fieldId)
	f := &SyncField{DBEntity: DBEntity{Id: i}}
	err := f.loadDb()
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	f.ErpPk = eqstring(r.FormValue("ErpPk"), "on")
	f.JsonName = jsonName
	err = f.updateDb()
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	http.Redirect(w, r, "/editSyncField/"+fieldId, http.StatusFound)
}

func deleteDecoratorHandler(w http.ResponseWriter, r *http.Request) {
	t, _ := template.ParseFiles("./template/editSyncField.html")
	idf, _ := iFField(r)
	idd, _ := iFDec(r)
	d := &Decorator{DBEntity: DBEntity{Id: idd}}
	err := d.deleteDb()
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	f := &SyncField{DBEntity: DBEntity{Id: idf}}
	f.reOrderDecorators()
	err = f.loadDb()
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	t.Execute(w, f)
}

func deleteDecoratorInAddHandler(w http.ResponseWriter, r *http.Request) {
	fieldId := sFField(r)
	idd, _ := iFDec(r)
	d := &Decorator{DBEntity: DBEntity{Id: idd}}
	err := d.deleteDb()
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	idf, _ := iFField(r)
	f := &SyncField{DBEntity: DBEntity{Id: idf}}
	f.reOrderDecorators()
	err = f.loadDb()
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	http.Redirect(w, r, "/addDecorator/"+fieldId, http.StatusFound)
}

func deleteErpHandler(w http.ResponseWriter, r *http.Request) {
	i, _ := strconv.Atoi(r.URL.Path[len("/deleteErp/"):])
	erp := &Erp{DBEntity: DBEntity{Id: i}}
	err := erp.loadDb() // Check if the load is required here
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	err = erp.deleteDb()
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	http.Redirect(w, r, "/erpsources", http.StatusFound)
}

func editErpHandler(w http.ResponseWriter, r *http.Request) {
	i, _ := readIntUrl(r)
	erp := &Erp{DBEntity: DBEntity{Id: i}}
	err := erp.loadDb()
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	if erp.TypeInt == MYSQL_TYPE {
		t, _ := template.ParseFiles("./template/erpEditMySql.html")
		t.Execute(w, erp)
	} else {
		t, _ := template.ParseFiles("./template/erpEditTODO.html")
		t.Execute(w, erp)
	}
}

func editErpEntryHandler(w http.ResponseWriter, r *http.Request) {
	t, _ := template.ParseFiles("./template/editErpEntry.html")
	i, _ := readIntUrl(r)
	ent := &ErpEntry{DBEntity: DBEntity{Id: i}}
	err := ent.loadDb()
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	t.Execute(w, ent)
}

func createSyncFieldHandler(w http.ResponseWriter, r *http.Request) {
	o := &SyncField{}
	o.ErpEntryId, _ = iFId(r)
	o.FieldName = sFFieldName(r)
	err := o.saveDb()
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	http.Redirect(w, r, "/erpentries", http.StatusFound)
}

func addMySQLHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/createMySQL.html", http.StatusFound)
}

func createMySQLHandler(w http.ResponseWriter, r *http.Request) {
	o := &Erp{}
	o.TypeInt = MYSQL_TYPE
	o.Type = "MySql"
	o.Name = sFName(r)
	o.Value = sFValue(r)
	err := o.saveDb()
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	http.Redirect(w, r, "/erpsources", http.StatusFound)
}

func updateMySQLHandler(w http.ResponseWriter, r *http.Request) {
	i, _ := iFId(r)
	erp := &Erp{DBEntity: DBEntity{Id: i}}
	err := erp.loadDb()
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	erp.Name = sFName(r)
	erp.Value = sFName(r)
	err = erp.updateDb()
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	http.Redirect(w, r, "/erpsources", http.StatusFound)
}

func addAccessHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/createAccess.html", http.StatusFound)
}

func adminHandler(w http.ResponseWriter, r *http.Request) {
	t, _ := template.ParseFiles("./template/admin.html")
	t.Execute(w, nil)
}

func configJSonHandler(w http.ResponseWriter, r *http.Request) {
	c := &CentralConfig{}
	s := c.toJson()
	jsonHtmlTmpl.Execute(w, template.JS(s))
}

func configXmlHandler(w http.ResponseWriter, r *http.Request) {
	c := &CentralConfig{}
	s := c.toXml()
	xmlHtmlTmpl.Execute(w, template.JS(s))
}

func importConfigJSonHandler(w http.ResponseWriter, r *http.Request) {
	t, _ := template.ParseFiles("./template/importConfigJson.html")
	t.Execute(w, nil)
}

func addDecoratorHandler(w http.ResponseWriter, r *http.Request) {
	t, _ := template.ParseFiles("./template/addDecorator.html")
	i, _ := readIntUrl(r)
	f := &SyncField{DBEntity: DBEntity{Id: i}}
	err := f.loadDb()
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	t.Execute(w, f)
}

func requestAddDecoratorHandler(w http.ResponseWriter, r *http.Request) {
	decI, _ := iFDec2(r)
	fieldId := sFField(r)
	idf, _ := iFField(r)

	pDec := decorators[decI]
	if pDec.Params == nil {
		t, _ := template.ParseFiles("./template/addDecorator.html")
		d := &Decorator{}
		d.DecoratorId = decI
		d.Params = ""
		d.SyncFieldId = idf

		f := &SyncField{DBEntity: DBEntity{Id: idf}}
		err := f.loadDb()
		if err != nil {
			fmt.Printf("%v\n", err)
		}

		d.SortingOrder = len(f.Decorators) + 1
		err = d.saveDb()
		if err != nil {
			fmt.Printf("%v\n", err)
		}
		err = f.loadDbDecorators()
		if err != nil {
			fmt.Printf("%v\n", err)
		}
		t.Execute(w, f)
	} else {
		t, _ := template.ParseFiles(pDec.Template)
		rac := &RequestAddContent{}
		rac.FieldId = idf
		rac.DecoratorId = decI
		rac.PDecorator = decorators[decI]
		t.Execute(w, rac)
	}
	http.Redirect(w, r, "/editSyncField/"+fieldId, http.StatusNotFound)
}

func requestAddDecoratorParamHandler(w http.ResponseWriter, r *http.Request) {
	t, _ := template.ParseFiles("./template/addDecorator.html")
	decI, _ := iFDec2(r)
	idf, _ := iFField(r)

	pDec := decorators[decI]
	params := pDec.Params

	var content string
	for i := 0; i < len(params); i++ {
		sp := r.FormValue(params[i].Name)
		content += fmt.Sprintf("\"%s\":\"%s\",", params[i].Name, sp)
	}
	content = fmt.Sprintf("{%s}", content[:len(content)-1])
	d := &Decorator{DecoratorId: decI, SyncFieldId: idf}
	d.Params = content
	f := &SyncField{DBEntity: DBEntity{Id: idf}}
	err := f.loadDb()
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	d.SortingOrder = len(f.Decorators) + 1
	err = d.saveDb()
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	err = f.loadDbDecorators()
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	t.Execute(w, f)
}

func inspectHandler(w http.ResponseWriter, r *http.Request) {
	if i, err := readIntUrl(r); err != nil {
		t, _ := template.ParseFiles("./template/inspectList.html")
		ens, _ := getErpEntries()
		t.Execute(w, ens)
	} else {
		t, _ := template.ParseFiles("./template/inspection.html")
		en := &ErpEntry{DBEntity: DBEntity{Id: i}}
		err := en.loadDb()
		if err != nil {
			fmt.Printf("%v\n", err)
		}

		ie := &InspectedEntry{}
		ie.Limit = 10

		ie.Entry = en
		st, err := dbC.Prepare("SELECT TABLE_NAME, TABLE_ROWS, AVG_ROW_LENGTH, DATA_LENGTH, DATA_FREE, AUTO_INCREMENT, CREATE_TIME, TABLE_COLLATION FROM information_schema.tables WHERE TABLE_SCHEMA = 'mid_db' and TABLE_NAME='" + en.getImportationTable() + "'")
		checkErr(err)
		rows, err := st.Query()
		checkErr(err)
		for rows.Next() {
			err := rows.Scan(&ie.ImportationTableName, &ie.ImportedRows, &ie.AGVRowsLength, &ie.DataLength, &ie.DataFree, &ie.AutoIncrement, &ie.CreateTime, &ie.TableCollation)
			fmt.Printf("Imported Row %v\n", ie.ImportedRows)
			if err != nil {
				fmt.Printf("err 04 %v\n", err)
				return
			}
		}

		if limit, err := iFLimit(r); err != nil {
			ie.Limit = 10
		} else {
			ie.Limit = limit
		}

		likeOnErpPk := sFLikeOnPk(r)
		ie.LikeOnErpPk = likeOnErpPk
		likeOnContent := sFLikeOnContent(r)
		ie.LikeOnContent = likeOnContent
		ie.LoadedContentLines, _ = en.getLoadedContent(likeOnErpPk, likeOnContent, ie.Limit)
		t.Execute(w, ie)
	}
}

func initDb(db *sql.DB) {
	defer fmt.Printf("Init DB DONE! \n")
	err := initDbErp(db)
	checkErr(err)
	err = initDbEntry(db)
	checkErr(err)
	err = initDbSyncField(db)
	checkErr(err)
	err = initDbDecorator(db)
	checkErr(err)
	err = initDbSyncEvent(db)
	checkErr(err)
}

func readFormInt(n string) func(r *http.Request) (int, error) {
	return func(r *http.Request) (int, error) {
		if i, err := strconv.Atoi(r.FormValue(n)); err == nil {
			return i, nil
		} else {
			fmt.Printf("err %v\n", err)
			return 0, err
		}
	}
}

func readFormString(n string) func(r *http.Request) string {
	return func(r *http.Request) string {
		return r.FormValue(n)
	}
}

func readIntUrl(r *http.Request) (int, error) {
	fmt.Printf("readIntUrl %v\n", r.URL.Path)
	s := strings.Split(r.URL.Path, "/")
	if i, err := strconv.Atoi(s[len(s)-1]); err == nil {
		fmt.Printf("readIntUrl return %v\n", i)
		return i, nil
	} else {
		fmt.Printf("readIntUrl err %v\n", err)
		return 0, err
	}
}

func loadIdInt(f rFInt, l Loader) func(r *http.Request) (Loader, error) {
	return func(r *http.Request) (Loader, error) {
		fmt.Printf("before calling f %v\n", r)
		i, err := f(r)

		if err != nil {
			fmt.Printf("called1 %v\n", err)
			return nil, err
		} else {
			fmt.Printf("called2 %v\n", i)
			lp := l
			fmt.Printf("lp1 %v\n", lp)
			lp.setId(i)
			fmt.Printf("lp2 %v\n", lp.getId())
			lp.loadDb()
			return l, nil
		}
		/**
		if i, err := f(r); err != nil {
			lp := l
			fmt.Printf("setId %v\n", i)
			lp.setId(i)

			lp.loadDb()
			return l, nil
		} else {
			fmt.Printf("err %v\n", err)
			return nil, err
		}
		*/
	}
}
