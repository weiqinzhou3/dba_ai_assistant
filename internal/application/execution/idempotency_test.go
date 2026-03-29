package execution

import "testing"

func TestBuildIdempotencyKeyFormatsActionAssetAndParam(t *testing.T) {
	got := BuildIdempotencyKey("mysql.database.create", "asset-mysql-prod-01", "order_db")
	want := "mysql.database.create:asset-mysql-prod-01:order_db"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}
