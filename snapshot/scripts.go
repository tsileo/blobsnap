package snapshot

import (
	"fmt"
	"github.com/tsileo/blobstash/client/script"
)

// Args: {"host": "nuc-server"}
// Result: { "snapshots": [ { "hostname": "nuc-server", "path": "/home/thomas/work" } ] }
var HostSnapSetScript = `
local snapshots, _ = blobstash.DB.Smembers("blobsnap:host:" .. blobstash.Args.host)
local res = {}
for i = 1, #snapshots do
    local snap, _ = blobstash.DB.GetHash("blobsnap:snapset:" .. snapshots[i])
    snap.hash = snapshots[i]
    table.insert(res, snap)
end
return {snapshots = res}`


// Args: {"host": "nuc-server"}
// Result: { "snapshots": [{ "hash": "520a787ca2ae1283219a33f95536ed05ccf07a20e72bbee6f0e4801eb9cdd84b",
// "mode": "2147484141", "mtime": "2014-08-13T20:56:09+02:00", "name": "docs",
// "ref": "b2578355e7c199f9a22795845da54d768b3a5c008cf6fcce5a9250b6ccd85581", "size": "8958", "type": "dir" } ] }
var HostLatestScript = `
local snapshots, _ = blobstash.DB.Smembers("blobsnap:host:" .. blobstash.Args.host)
local res = {}
for i = 1, #snapshots do
    local last, _ = blobstash.DB.Llast("blobsnap:snapset:" .. snapshots[i] .. ":history")
    local meta, _ = blobstash.DB.GetHash(last)
    meta.hash = last
    table.insert(res, meta)
end
return {snapshots = res}`

var SnapshotsScript = `
local snapshots, _ = blobstash.DB.LiterWithIndex("blobsnap:snapset:" .. blobstash.Args.snapset .. ":history")
return {snapshots = snapshots}
`
func Snapshots(hash string) (map[string]interface{}, error) {
	res := map[string]interface{}{}
	if err := script.RunScript("", SnapshotsScript, "{\"snapset\":\""+hash+"\"}", &res); err != nil {
		return res, fmt.Errorf("failed to run script: %v", err)
	}
	return res, nil
}
func HostSnapSet(host string) (map[string]interface{}, error) {
	res := map[string]interface{}{}
	if err := script.RunScript("", HostSnapSetScript, "{\"host\":\""+host+"\"}", &res); err != nil {
		return res, fmt.Errorf("failed to run script: %v", err)
	}
	return res, nil
}

func HostLatest(host string) (map[string]interface{}, error) {
	res := map[string]interface{}{}
	if err := script.RunScript("", HostLatestScript, "{\"host\":\""+host+"\"}", &res); err != nil {
		return res, fmt.Errorf("failed to run script: %v", err)
	}
	return res, nil
}
