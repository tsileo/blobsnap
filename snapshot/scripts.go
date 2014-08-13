package snapshot

import (
	"fmt"
	"github.com/tsileo/blobstash/client/script"
)

// Args: {"host": "nuc-server"}
// Result: { "snapshots": [ { "hostname": "nuc-server", "path": "/home/thomas/work" } ] }
var HostLatestScript = `
local snapshots, _ = blobstash.DB.Smembers("blobsnap:host:" .. blobstash.Args.host)
local res = {}
for i = 1, #snapshots do
    local last, _ = blobstash.DB.Llast("blobsnap:snapset:" .. snapshots[i] .. ":history")
    local snap, _ = blobstash.DB.GetHash("blobsnap:snapshot:" .. last)
    local meta, _ = blobstash.DB.GetHash(snap.meta_ref)
    meta.hash = snap.meta_ref
    table.insert(res, meta)
end
return {snapshots = res}`

func HostLatest(host string) (map[string]interface{}, error) {
	res := map[string]interface{}{}
	if err := script.RunScript("", HostLatestScript, "{\"host\":\""+host+"\"}", &res); err != nil {
		return res, fmt.Errorf("failed to run script: %v", err)
	}
	return res, nil
}
