package dimen

import (
	"testing"

	"github.com/npillmayer/schuko/tracing/gotestingadapter"
)

func TestParseDimen(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "tyse.core")
	defer teardown()
	//
	d, _, err := ParsePercent("12px")
	if err != nil {
		t.Errorf("(1) %s", err.Error())
	} else if d != 12*PX {
		t.Errorf("(1) expected d to be 12px (%d), is %d", 12*PX, d)
	}
	//
	d, _, err = ParsePercent("0")
	if err != nil {
		t.Errorf("(2) %s", err.Error())
	} else if d != 0 {
		t.Errorf("(2) expected d to be 0, is %d", d)
	}
	//
	_, ispcnt, err := ParsePercent("20%")
	if err != nil {
		t.Errorf("(3) %s", err.Error())
	} else if ispcnt != true {
		t.Errorf("(3) expected percentage-marker to be true, is %v", ispcnt)
	}
}
