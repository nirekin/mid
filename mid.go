package main

/*
Install of go-sql-driver

Define the GOPATH in system variable
Make sure Git is installed on your machine and in your system's PATH. Then simple run in your system's shell:

go get github.com/go-sql-driver/mysql
go get github.com/mxk/go-sqlite/sqlite3   --> mingv-w64	github.com/mxk/go-sqlite/sqlite3

DOC https://gobyexample.com/time-formatting-parsing
t := fmt.Sprintf("%d-%02d-%02d %02d:%02d:%02d", fromTime.Year(), fromTime.Month(), fromTime.Day(), fromTime.Hour(), fromTime.Minute(), fromTime.Second())

fmt.Printf("blable %d\n" , id)

TODO :

- Encode special characters in something like UTF8...
- Test volume speed
- Select criteria para MYSQL

- Ping for erp
- JS validation in all forms
- Entry hierarchy mother/child
- Log for extraction
- Log for error
- Synchronizer by entry o for all entries ?

- Export config json / XML
- Consume config json  / XML

- Master slave
	- register slaves
	- push changes
	- find gap
	- fix gap
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
	COUNT_ERP_MYSQL    = "SELECT COUNT(*) FROM information_schema.tables WHERE TABLE_SCHEMA = ?"
	COUNT_TABLE_MYSQL  = "SELECT COUNT(*) FROM information_schema.columns WHERE TABLE_SCHEMA = ? AND TABLE_NAME =?"
	SELECT_ERP_MYSQL   = "SELECT TABLE_NAME FROM information_schema.tables WHERE TABLE_SCHEMA = ?"
	SELECT_TABLE_MYSQL = "select COLUMN_NAME from information_schema.columns where TABLE_SCHEMA = ? AND TABLE_NAME =?"

	MYSQL_TYPE       = 1
	MYSQL_TYPE_SPLIT = "__/$/__"
	MYSQL_TYPE_EMPTY = "__/#/__"
	ACCESS_TYPE      = 2

	
	db_url          = "root:admin@tcp(localhost:3306)/mid_db"
)

var dbC *sql.DB
var cptGen int
var decorators DecoratorMap

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	defer dbC.Close()
	defer fmt.Printf("stopped \n")

	// ERP ---
	http.HandleFunc("/admin/", adminHandler)
	http.HandleFunc("/viewConfigJSon/", viewConfigJSonHandler)
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
	http.HandleFunc("/pingErpEntry/", pingErpEntryHandler)
	http.HandleFunc("/pingTestErpEntry/", pingTestErpEntryHandler)

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
	http.ListenAndServe(":8090", nil)
}

func init() {
	defer fmt.Printf("Init DONE\n")
	dbC, _ = sql.Open("mysql", db_url)
	initDb(dbC)
	initDecorators()
}

func erpsourcesHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("erpsourcesHandler\n")
	t, _ := template.ParseFiles("./template/erpsources.html")
	all := getErps()
	t.Execute(w, all)
}

func erpentriesHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("erpentriesHandler\n")
	t, _ := template.ParseFiles("./template/erpentries.html")
	all := getErpEntries()
	t.Execute(w, all)
}

func erpListTablesHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("erpListTablesHandler\n")
	t, _ := template.ParseFiles("./template/erpListTables.html")
	i, _ := strconv.Atoi(r.URL.Path[len("/erpListTables/"):])
	erp := &Erp{Id: i}
	erp.loadDb()
	_ = erp.lazyLoadTables()
	t.Execute(w, erp)
}

func erpListFieldsHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("erpListFieldsHandler\n")
	t, _ := template.ParseFiles("./template/erpListFields.html")
	i, _ := strconv.Atoi(r.URL.Path[len("/erpListTables/"):])
	ent := &ErpEntry{Id: i}
	ent.loadDb()
	_ = ent.lazyLoadRFields()
	t.Execute(w, ent)
}

func createErpEntryHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("createErpEntryHandler\n")
	t, _ := template.ParseFiles("./template/createErpEntry.html")
	o := &ErpEntry{}
	i, _ := strconv.Atoi(r.FormValue("Id"))
	o.ErpId = i
	o.SourceName = r.FormValue("SourceName")
	t.Execute(w, o)
}

func saveErpEntryHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("saveErpEntryHandler\n")
	o := &ErpEntry{}
	o.ErpId, _ = strconv.Atoi(r.FormValue("ErpId"))
	o.SourceName = r.FormValue("SourceName")
	o.Name = r.FormValue("Name")
	o.saveDb()
	o.createImportationTableName()
	http.Redirect(w, r, "/erpentries", http.StatusFound)
}

func updateErpEntryHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("updateErpEntryHandler\n")
	i, _ := strconv.Atoi(r.FormValue("Id"))
	ent := &ErpEntry{Id: i}
	ent.loadDb()
	ent.Name = r.FormValue("Name")
	ent.updateDb()
	http.Redirect(w, r, "/erpentries", http.StatusFound)
}

func pingErpEntryHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("pingErpEntryHandler\n")
	t, _ := template.ParseFiles("./template/editErpEntry.html")
	i, _ := strconv.Atoi(r.FormValue("Id"))
	nbRows, _ := strconv.Atoi(r.FormValue("NbRows"))
	ent := &ErpEntry{Id: i}
	ent.loadDb()
	ent.ping(nbRows)
	t.Execute(w, ent)
}

func pingTestErpEntryHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("pingTestErpEntryHandler\n")
	t, _ := template.ParseFiles("./template/addDecorator.html")
	idf, _ := strconv.Atoi(r.FormValue("FieldId"))
	f := &SyncField{Id: idf}
	f.loadDb()
	f.TestContent = r.FormValue("TestContent")
	mapD := map[string]string{}
	fN, val := f.decorate(f.TestContent)
	mapD[fN] = val
	outJson, _ := json.Marshal(mapD)
	f.TestResponse = string(outJson)
	t.Execute(w, f)
}

func syncErpEntryHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("syncErpEntryHandler\n")
	i, _ := strconv.Atoi(r.URL.Path[len("/syncErpEntry/"):])
	ent := &ErpEntry{Id: i}
	ent.loadDb()
	ent.sync()
	http.Redirect(w, r, "/erpentries", http.StatusFound)
}

func deleteErpEntryHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("deleteErpEntryHandler\n")
	i, _ := strconv.Atoi(r.URL.Path[len("/deleteErpEntry/"):])
	ent := &ErpEntry{Id: i}
	ent.loadDb() // TODO check if the load is required here
	ent.deleteDb()
	http.Redirect(w, r, "/erpentries", http.StatusFound)
}

func deleteSyncFielddHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("deleteSyncFielddHandler\n")
	fieldId := r.FormValue("FieldId")
	entryId := r.FormValue("EntryId")
	idf, _ := strconv.Atoi(fieldId)
	f := &SyncField{Id: idf}
	f.loadDb() // TODO check id the load is required here
	f.deleteDb()
	http.Redirect(w, r, "/editErpEntry/"+entryId, http.StatusFound)
}

func editSyncFieldHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("editSyncFieldHandler\n")
	t, _ := template.ParseFiles("./template/editSyncField.html")
	i, _ := strconv.Atoi(r.URL.Path[len("/editSyncField/"):])
	f := &SyncField{Id: i}
	f.loadDb()
	t.Execute(w, f)
}

func updateSyncFieldHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("updateSyncFieldHandler\n")
	fieldId := r.FormValue("Id")
	jsonName := r.FormValue("JsonName")
	i, _ := strconv.Atoi(fieldId)
	f := &SyncField{Id: i}
	f.loadDb()
	f.ErpPk = eqstring(r.FormValue("ErpPk"), "on")
	f.JsonName = jsonName
	f.updateDb()
	http.Redirect(w, r, "/editSyncField/"+fieldId, http.StatusFound)
}

func deleteDecoratorHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("deleteDecoratorHandler\n")
	t, _ := template.ParseFiles("./template/editSyncField.html")
	idf, _ := strconv.Atoi(r.FormValue("FieldId"))
	idd, _ := strconv.Atoi(r.FormValue("DecId"))
	deleteDecoratorById(idd)
	f := &SyncField{Id: idf}
	f.reOrderDecorators()
	f.loadDb()
	t.Execute(w, f)
}

func deleteDecoratorInAddHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("deleteDecoratorInAddHandler\n")
	fieldId := r.FormValue("FieldId")
	decId := r.FormValue("DecId")
	idd, _ := strconv.Atoi(decId)
	deleteDecoratorById(idd)
	idf, _ := strconv.Atoi(r.FormValue("FieldId"))
	f := &SyncField{Id: idf}
	f.reOrderDecorators()
	f.loadDb()
	http.Redirect(w, r, "/addDecorator/"+fieldId, http.StatusFound)
}

func deleteErpHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("deleteErpHandler\n")
	i, _ := strconv.Atoi(r.URL.Path[len("/deleteErp/"):])
	erp := &Erp{Id: i}
	erp.loadDb() // Check if the load is required here
	erp.deleteDb()
	http.Redirect(w, r, "/erpsources", http.StatusFound)
}

func editErpHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("editErpHandler\n")
	i, _ := strconv.Atoi(r.URL.Path[len("/editErp/"):])
	erp := &Erp{Id: i}
	erp.loadDb()
	if erp.TypeInt == MYSQL_TYPE {
		t, _ := template.ParseFiles("./template/erpEditMySql.html")
		t.Execute(w, erp)
	} else {
		t, _ := template.ParseFiles("./template/erpEditTODO.html")
		t.Execute(w, erp)
	}
}

func editErpEntryHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("editErpEntryHandler\n")
	t, _ := template.ParseFiles("./template/editErpEntry.html")
	i, _ := strconv.Atoi(r.URL.Path[len("/editErpEntry/"):])
	ent := &ErpEntry{Id: i}
	ent.loadDb()
	t.Execute(w, ent)
}

func createSyncFieldHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("createSyncFieldHandler\n")
	o := &SyncField{}
	o.ErpEntryId, _ = strconv.Atoi(r.FormValue("Id"))
	o.FieldName = r.FormValue("FieldName")
	o.saveDb()
	http.Redirect(w, r, "/erpentries", http.StatusFound)
}

func addMySQLHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("addMySQLHandler\n")
	http.Redirect(w, r, "/createMySQL.html", http.StatusFound)
}

func createMySQLHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("createMySQLHandler\n")
	o := &Erp{}
	o.TypeInt = MYSQL_TYPE
	o.Type = "MySql"
	o.Name = r.FormValue("Name")
	o.Value = r.FormValue("Value")
	o.saveDb()
	http.Redirect(w, r, "/erpsources", http.StatusFound)
}

func updateMySQLHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("updateMySQLHandler\n")
	i, _ := strconv.Atoi(r.FormValue("Id"))
	erp := &Erp{Id: i}
	erp.loadDb()
	erp.Name = r.FormValue("Name")
	erp.Value = r.FormValue("Value")
	erp.updateDb()
	http.Redirect(w, r, "/erpsources", http.StatusFound)
}

func addAccessHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("addAccessHandler\n")
	http.Redirect(w, r, "/createAccess.html", http.StatusFound)
}

func adminHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("adminHandler\n")
	t, _ := template.ParseFiles("./template/admin.html")
	t.Execute(w, nil)
}

func viewConfigJSonHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("viewConfigJSonHandler\n")
	t, _ := template.ParseFiles("./template/viewConfigJson.html")
	c := &CentralConfig{}
	c.toJson()
	t.Execute(w, template.JS(c.JSonContent))
}

func importConfigJSonHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("importConfigJSonHandler\n")
	t, _ := template.ParseFiles("./template/importConfigJson.html")
	t.Execute(w, nil)
}

func addDecoratorHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("addDecoratorHandler\n")
	t, _ := template.ParseFiles("./template/addDecorator.html")
	i, _ := strconv.Atoi(r.URL.Path[len("/addDecorator/"):])
	f := &SyncField{Id: i}
	f.loadDb()
	t.Execute(w, f)
}

func requestAddDecoratorHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("requestAddDecoratorHandler\n")
	decI, _ := strconv.Atoi(r.FormValue("DecoratorId"))
	fieldId := r.FormValue("FieldId")
	idf, _ := strconv.Atoi(fieldId)

	pDec := decorators[decI]
	if pDec.Params == nil {
		t, _ := template.ParseFiles("./template/addDecorator.html")
		d := &Decorator{}
		d.DecoratorId = decI
		d.Params = ""
		d.SyncFieldId = idf

		f := &SyncField{Id: idf}
		f.loadDb()

		d.SortingOrder = len(f.Decorators) + 1
		d.saveDb()
		f.loadDbDecorators()
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
	fmt.Printf("requestAddDecoratorParamHandler\n")
	t, _ := template.ParseFiles("./template/addDecorator.html")
	decI, _ := strconv.Atoi(r.FormValue("DecoratorId"))
	idf, _ := strconv.Atoi(r.FormValue("FieldId"))

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
	f := &SyncField{Id: idf}
	f.loadDb()
	d.SortingOrder = len(f.Decorators) + 1
	d.saveDb()
	f.loadDbDecorators()

	t.Execute(w, f)
}

func inspectHandler(w http.ResponseWriter, r *http.Request) {
	i, err := strconv.Atoi(r.URL.Path[len("/inspect/"):])
	if err != nil {
		t, _ := template.ParseFiles("./template/inspectList.html")
		t.Execute(w, getErpEntries())
	} else {
		t, _ := template.ParseFiles("./template/inspection.html")
		en := &ErpEntry{Id: i}
		en.loadDb()

		ie := &InspectedEntry{}
		ie.Limit = 10

		ie.Entry = en
		st, err := dbC.Prepare("SELECT TABLE_NAME, TABLE_ROWS, AVG_ROW_LENGTH, DATA_LENGTH, DATA_FREE, AUTO_INCREMENT, CREATE_TIME, TABLE_COLLATION FROM information_schema.tables WHERE TABLE_SCHEMA = 'mid_db' and TABLE_NAME='" + en.getImportationTableName() + "'")
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

		limit, err := strconv.Atoi(r.FormValue("Limit"))
		if err != nil {
			ie.Limit = 10
		} else {
			ie.Limit = limit
		}

		likeOnErpPk := r.FormValue("LikeOnErpPk")
		ie.LikeOnErpPk = likeOnErpPk
		likeOnContent := r.FormValue("LikeOnContent")
		ie.LikeOnContent = likeOnContent
		ie.LoadedContentLines = en.getLoadedContent(likeOnErpPk, likeOnContent, ie.Limit)
		t.Execute(w, ie)
	}
}

func initDb(db *sql.DB) {
	defer fmt.Printf("Init DB DONE! \n")

	// TABLE FOR ERP
	sql := "CREATE TABLE IF NOT EXISTS `mid_db`.`admin_erp` (`id` INTEGER UNSIGNED NOT NULL AUTO_INCREMENT,`creationDate` DATETIME NOT NULL DEFAULT 0,`typeInt` INTEGER UNSIGNED NOT NULL DEFAULT 0,`type` VARCHAR(45) NOT NULL DEFAULT '',`name` VARCHAR(45) NOT NULL DEFAULT '',`value` longtext, PRIMARY KEY(`id`))ENGINE = InnoDB;"
	st, err := db.Prepare(sql)
	defer st.Close()
	checkErr(err)
	_, err = st.Exec()
	checkErr(err)

	// TABLE FOR ERP ENTRY
	sql = "CREATE TABLE IF NOT EXISTS `mid_db`.`admin_erp_entry` (`id` int(10) unsigned NOT NULL AUTO_INCREMENT,`erpId` int(10) unsigned NOT NULL DEFAULT '0',  `creationDate` datetime NOT NULL DEFAULT '0000-00-00 00:00:00',`sourceName` varchar(255) NOT NULL DEFAULT '',`name` varchar(255) NOT NULL DEFAULT '',PRIMARY KEY (`id`)) ENGINE=InnoDB DEFAULT CHARSET=latin1;"
	st, err = db.Prepare(sql)
	checkErr(err)
	_, err = st.Exec()
	checkErr(err)

	// TABLE FOR SYNC FIELD
	//sql = "CREATE TABLE IF NOT EXISTS `mid_db`.`admin_sync_field` (`id` int(10) unsigned NOT NULL AUTO_INCREMENT,`erpEntryId` int(10) unsigned NOT NULL DEFAULT '0',  `creationDate` datetime NOT NULL DEFAULT '0000-00-00 00:00:00',`fieldName` varchar(255) NOT NULL DEFAULT '',PRIMARY KEY (`id`)) ENGINE=InnoDB DEFAULT CHARSET=latin1;"
	sql = "CREATE TABLE IF NOT EXISTS `mid_db`.`admin_sync_field` (`id` int(10) unsigned NOT NULL AUTO_INCREMENT,`erpEntryId` int(10) unsigned NOT NULL DEFAULT '0',  `creationDate` datetime NOT NULL DEFAULT '0000-00-00 00:00:00',`fieldName` varchar(255) NOT NULL DEFAULT '',`jsonName` varchar(255) NOT NULL DEFAULT '',`erpPk` int(10) unsigned DEFAULT '0',PRIMARY KEY (`id`)) ENGINE=InnoDB DEFAULT CHARSET=latin1;"
	st, err = db.Prepare(sql)
	checkErr(err)
	_, err = st.Exec()
	checkErr(err)

	// TABLE FOR SYNC FIELD DECORATOR
	sql = "CREATE TABLE IF NOT EXISTS `mid_db`.`admin_sync_field_decorator` (`id` int(10) unsigned NOT NULL AUTO_INCREMENT, `decoratorId` int(10) unsigned NOT NULL DEFAULT '0', `syncFieldId` int(10) unsigned NOT NULL DEFAULT '0', `sortingOrder` int(10) unsigned NOT NULL DEFAULT '0', `params` varchar(255) NOT NULL DEFAULT '',PRIMARY KEY (`id`)) ENGINE=InnoDB DEFAULT CHARSET=latin1;"
	st, err = db.Prepare(sql)
	checkErr(err)
	_, err = st.Exec()
	checkErr(err)

	// TABLE FOR SYNC JOUNRNAL
	sql = "CREATE TABLE IF NOT EXISTS `mid_db`.`admin_sync_events` (`id` int(10) unsigned NOT NULL AUTO_INCREMENT, `erpEntryId` int(10) unsigned NOT NULL DEFAULT '0', `syncDate` bigint(20) unsigned DEFAULT NULL, `imported` int(10) unsigned NOT NULL DEFAULT '0', `updated` int(10) unsigned NOT NULL DEFAULT '0', `deleted` int(10) unsigned NOT NULL DEFAULT '0', PRIMARY KEY (`id`)) ENGINE=InnoDB DEFAULT CHARSET=latin1;"
	st, err = db.Prepare(sql)
	checkErr(err)
	_, err = st.Exec()
	checkErr(err)
}