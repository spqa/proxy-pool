package pool

import "testing"

func TestProxyHubFetcher_Get(t *testing.T) {
	fetcher := new(ProxyHubFetcher)
	get, err := fetcher.Get()
	if err != nil {
		t.Error(err)
	} else {
		t.Logf("entities %v", get)
		if len(get) == 0 {
			t.Fail()
		}
	}
}
