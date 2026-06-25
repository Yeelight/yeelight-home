package credential

import "testing"

func TestSystemStoreBuildsStableAccountName(t *testing.T) {
	store := NewSystemStore("yeelight-home")

	if got := store.accountName("default"); got != "yeelight-home:default" {
		t.Fatalf("accountName = %q", got)
	}
}
