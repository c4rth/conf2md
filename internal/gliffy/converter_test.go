package gliffy

import (
	"encoding/xml"
	"testing"
)

func TestConvertGliffyV2UsesDefaultPage(t *testing.T) {
	input := []byte(`{
		"version":"2.0",
		"defaultPage":"selected",
		"pages":[
			{"id":"other","scene":{"background":"#111111","objects":[]}},
			{"id":"selected","scene":{
				"background":"#222222",
				"gridOn":true,
				"drawingGuidesOn":true,
				"objects":[{
					"id":"node",
					"width":20,
					"height":10,
					"graphic":{"type":"Shape","Shape":{"strokeColor":"#000000","fillColor":"#FFFFFF"}}
				}]
			}}
		]
	}`)

	result, err := Convert(input)
	if err != nil {
		t.Fatal(err)
	}

	var model MxGraphModel
	if err := xml.Unmarshal(result.XML, &model); err != nil {
		t.Fatal(err)
	}
	if model.Background != "#222222" || model.Grid != "1" || model.Guides != "1" {
		t.Errorf("unexpected converted scene settings: %+v", model)
	}
	if len(model.Root.Cells) != 3 {
		t.Errorf("unexpected converted cells: %+v", model.Root.Cells)
		return
	}
	if cell := model.Root.Cells[2]; cell.ID != "2" || cell.Parent != "1" || cell.Vertex != "1" {
		t.Errorf("unexpected converted object cell: %+v", cell)
	}
}