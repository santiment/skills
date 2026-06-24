package score

import (
	"testing"
)

func TestQueryFlagsSanrStyle(t *testing.T) {
	q := queryFlags{
		style: styleSanr, page: 10, off: 5,
		sort: "createdAt:desc", filter: `"asset":"BTC"`, attrs: `"id","asset"`,
		params: []string{"status=active", "n=3", "flag=true"},
	}
	var got map[string]any
	if err := q.into(&got); err != nil {
		t.Fatal(err)
	}
	if got["take"] != float64(10) || got["skip"] != float64(5) {
		t.Errorf("take/skip mismatch: %v / %v", got["take"], got["skip"])
	}
	if got["sort"] != "createdAt:desc" || got["filter"] != `"asset":"BTC"` || got["attributes"] != `"id","asset"` {
		t.Errorf("string fields mismatch: %+v", got)
	}
	// --param values are JSON-typed when possible.
	if got["status"] != "active" {
		t.Errorf("status: want string active, got %v", got["status"])
	}
	if got["n"] != float64(3) {
		t.Errorf("n: want number 3, got %v (%T)", got["n"], got["n"])
	}
	if got["flag"] != true {
		t.Errorf("flag: want bool true, got %v (%T)", got["flag"], got["flag"])
	}
}

func TestQueryFlagsArenaStyle(t *testing.T) {
	q := queryFlags{style: styleArena, page: 25, off: 50}
	var got map[string]any
	if err := q.into(&got); err != nil {
		t.Fatal(err)
	}
	if got["limit"] != float64(25) || got["offset"] != float64(50) {
		t.Errorf("limit/offset mismatch: %v / %v", got["limit"], got["offset"])
	}
	if _, ok := got["take"]; ok {
		t.Error("arena style must not emit take")
	}
	if _, ok := got["attributes"]; ok {
		t.Error("arena style must not emit attributes")
	}
}

func TestQueryFlagsBadParam(t *testing.T) {
	q := queryFlags{style: styleSanr, params: []string{"missing-equals"}}
	var got map[string]any
	if err := q.into(&got); err == nil {
		t.Error("expected error for --param without '='")
	}
}

func TestParseScalarTypes(t *testing.T) {
	if v := parseScalar("BTC"); v != "BTC" {
		t.Errorf("plain string: got %v", v)
	}
	if v := parseScalar("42"); v != float64(42) {
		t.Errorf("number: got %v (%T)", v, v)
	}
	if v := parseScalar("true"); v != true {
		t.Errorf("bool: got %v (%T)", v, v)
	}
}
