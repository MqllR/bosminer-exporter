package cgminer

import "testing"

func TestDecode(t *testing.T) {
	r, err := decode([]byte(`{"STATUS":[{"STATUS":"S","Code":202,"Msg":"1 Fan(s)"}],"FANS":[{"RPM":1000}],"id":1}`))
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := Float(r.Data["FANS"][0], "RPM"); v != 1000 {
		t.Fatalf("RPM = %v", v)
	}
	if r.Status[0].Code != 202 {
		t.Fatalf("code = %d", r.Status[0].Code)
	}
}

func TestDecodeError(t *testing.T) {
	if _, err := decode([]byte(`{"STATUS":[{"STATUS":"E","Msg":"Invalid command"}],"id":1}`)); err == nil {
		t.Fatal("expected error")
	}
}
