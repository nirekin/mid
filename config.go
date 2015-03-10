package main

import (
	"encoding/json"
	//"fmt"
)

type CentralConfig struct {
	Erps        []Erp
	JSonContent string
}

func (o *CentralConfig) toJson() {
	o.Erps, _ = getErps()

	vErps := make([]*VisibleErp, len(o.Erps))
	for er := 0; er < len(o.Erps); er++ {
		erp := o.Erps[er]

		ve := &VisibleErp{}
		ve.CreationDate = erp.CreationDate
		ve.Name = erp.Name
		ve.Type = erp.Type
		ve.TypeInt = erp.TypeInt
		ve.Value = erp.Value

		erp.loadErpEntries()
		ve.Entries = make([]*VisibleErpEntry, len(erp.Entries))
		for en := 0; en < len(erp.Entries); en++ {
			ent := erp.Entries[en]

			vn := &VisibleErpEntry{}
			vn.CreationDate = ent.CreationDate
			vn.SourceName = ent.SourceName
			vn.Name = ent.Name
			ve.Entries[en] = vn

			ent.loadDbSyncFields()
			vn.Fields = make([]*VisibleSyncField, len(ent.SyncFields))
			for f := 0; f < len(ent.SyncFields); f++ {
				fi := ent.SyncFields[f]

				vf := &VisibleSyncField{}
				vf.CreationDate = fi.CreationDate
				vf.ErpPk = fi.ErpPk
				vf.FieldName = fi.FieldName
				vf.JsonName = fi.JsonName
				vn.Fields[f] = vf

				fi.loadDbDecorators()
				vf.Decoratos = make([]*VisibleDecorator, len(fi.Decorators))
				for d := 0; d < len(fi.Decorators); d++ {
					de := fi.Decorators[d]
					vd := &VisibleDecorator{}
					vd.Description = de.Description
					vd.Name = de.Name
					vd.Params = de.Params
					vd.SortingOrder = de.SortingOrder
					vf.Decoratos[d] = vd
				}
			}
		}
		vErps[er] = ve
	}

	b, _ := json.Marshal(vErps)
	json := "{\"erps\":"
	json += string(b)
	json += "}"
	o.JSonContent = json
}
