package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type partialEvent struct {
	Imported int64
	Updated  int64
}

func StartSync() {
	c := make(chan int)

	ticker := time.NewTicker(30 * time.Minute)
	go func() {
		i := 0
		for _ = range ticker.C {
			l, _ := getErpEntries()
			for _, val := range l {
				go synchronize(val)
				i++
			}
			c <- i
		}
	}()
	for {
		fmt.Printf("Sync Done %v at %v\n", <-c, time.Now())
	}
}

func synchronize(o ErpEntry) error {
	extractSentence := o.getExtractSentence()
	if extractSentence == "" {
		return nil
	}
	erp := &Erp{DBEntity: DBEntity{Id: o.ErpId}}
	erp.loadDb()

	var tKeys [10]string
	keys := tKeys[0:0]

	if erp.TypeInt == MYSQL_TYPE {
		dbCErp, err := sql.Open("mysql", erp.Value)
		if err != nil {
			return err
		} else {
			defer dbCErp.Close()
		}
		st, err := dbCErp.Prepare(extractSentence)
		if err != nil {
			return err
		} else {
			defer st.Close()
		}

		rows, err := st.Query()
		if err != nil {
			return err
		}
		var content string
		syncFs := o.SyncFields
		ecMap := make(map[string]*ExtractedContent)
		for rows.Next() {
			if err := rows.Scan(&content); err == nil {
				var pkContent string
				extractedString := strings.Split(content, MYSQL_TYPE_SPLIT)
				lExtractedString := len(extractedString)
				mapD := map[string]string{}
				for i := 0; i < lExtractedString; i++ {
					str := strings.Replace(extractedString[i], MYSQL_TYPE_EMPTY, "", -1)
					if syncFs[i].ErpPk {
						pkContent += str
					}
					fN, val := syncFs[i].decorate(str)
					mapD[fN] = val
				}
				outJson, _ := json.Marshal(mapD)
				ecMap[pkContent] = &ExtractedContent{ErpEntryId: o.Id, ErpPk: pkContent, Content: string(outJson)}
				keys = append(keys, pkContent)
			}
		}

		syncPrepare(o)
		blockSize := o.BlockSize
		if blockSize == 0 {
			blockSize = 100
		}

		chanLen := 0

		var chanEvent chan partialEvent
		var imported, updated, deleted int64 = 0, 0, 0
		iB, mod := 0, 0

		timeMSStart := getNowMillisecond()

		lenKeys := len(keys)
		if lenKeys <= blockSize {
			chanLen = 1
		} else {
			iB = lenKeys / blockSize
			mod = lenKeys % blockSize
			if mod > 0 {
				chanLen = iB + 1
			} else {
				chanLen = iB
			}
		}

		chanEvent = make(chan partialEvent, chanLen)
		if chanLen == 1 {
			go insertOrUpdate(chanEvent, o, ecMap, keys)
		} else {
			for i := 0; i < iB; i++ {
				go insertOrUpdate(chanEvent, o, ecMap, keys[i*blockSize:(i+1)*blockSize])
			}
			if mod > 0 {
				go insertOrUpdate(chanEvent, o, ecMap, keys[iB*blockSize:lenKeys])
			}
		}

		for i := 0; i < chanLen; i++ {
			s := <-chanEvent
			imported = imported + s.Imported
			updated = updated + s.Updated
		}
		deleted, _ = syncClean(o)
		timeMSStop := getNowMillisecond()
		_ = addEvent(o, imported, updated, deleted, timeMSStop-timeMSStart, int64(len(keys)))
		return nil
	} else {
		return nil
	}
}

func syncPrepare(e ErpEntry) error {
	if st, err := dbC.Prepare("UPDATE " + e.getImportationTableSchema() + " SET processedFromERP=0"); err == nil {
		defer st.Close()
		if _, err := st.Exec(); err != nil {
			return err
		}
		return nil
	} else {
		return err
	}
}

func syncClean(e ErpEntry) (int64, error) {
	if st, err := dbC.Prepare("DELETE FROM " + e.getImportationTableSchema() + " WHERE processedFromERP=0"); err == nil {
		defer st.Close()
		if res, err := st.Exec(); err == nil {
			if deleted, err := res.RowsAffected(); err == nil {
				return deleted, nil
			} else {
				return 0, err
			}
		} else {
			return 0, err
		}
	} else {
		return 0, err
	}
}

func insertOrUpdate(ch chan partialEvent, entry ErpEntry, ec ExtractedContentMap, keys []string) error {
	var inserted, updated int64 = 0, 0

	stIns, err := dbC.Prepare("INSERT " + entry.getImportationTableSchema() + " SET active=1, content=?, creationDate=?, erpPk=?, lastUpdate=?, name=?, processedFromERP=1")
	if err == nil {
		defer stIns.Close()
	} else {
		return err
	}

	stUpdate, err := dbC.Prepare("UPDATE " + entry.getImportationTableSchema() + " SET content=?, lastUpdate=?, processedFromERP=1 WHERE erpPk=?")
	if err == nil {
		defer stUpdate.Close()
	} else {
		return err
	}

	stUpdateProcessed, err := dbC.Prepare("UPDATE " + entry.getImportationTableSchema() + " SET processedFromERP=1 WHERE erpPk=?")
	if err == nil {
		defer stUpdateProcessed.Close()
	} else {
		return err
	}

	var cptE int
	for _, k := range keys {
		c := ec[k]
		if rows, err := dbC.Query("SELECT COUNT(*) FROM " + entry.getImportationTable() + " WHERE erpPk='" + c.ErpPk + "'"); err != nil {
			return err
		} else {
			for rows.Next() {
				_ = rows.Scan(&cptE)
			}
		}
		if cptE == 1 {
			s := "SELECT COUNT(*) FROM " + entry.getImportationTable() + " WHERE erpPk='" + c.ErpPk + "' AND content='" + c.Content + "'"
			if rows, err := dbC.Query(s); err != nil {
				return err
			} else {
				for rows.Next() {
					_ = rows.Scan(&cptE)
				}
			}
			if cptE == 0 {
				if _, err := stUpdate.Exec(c.Content, getNowMillisecond(), c.ErpPk); err != nil {
					return err
				}
				updated++
			} else if _, err := stUpdateProcessed.Exec(c.ErpPk); err != nil {
				return err
			}
		} else {
			n := getNowMillisecond()
			if _, err := stIns.Exec(c.Content, n, c.ErpPk, n, entry.Name); err != nil {
				return err
			}
			inserted++
		}
	}
	ch <- partialEvent{inserted, updated}
	return nil
}
