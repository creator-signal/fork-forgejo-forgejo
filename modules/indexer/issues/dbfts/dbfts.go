package dbfts

import (
	"forgejo.org/modules/indexer/issues/db"
	"forgejo.org/modules/indexer/issues/internal"
	"forgejo.org/modules/log"
	"forgejo.org/modules/setting"
)

var (
	registery map[string]FuncNew = map[string]FuncNew{}
	tableName string             = "issue_fts_idx_v1"
)

type FuncNew func() internal.Indexer

func NewIndexer() internal.Indexer {
	t := setting.Database.Type.String()
	if fn, ok := registery[t]; ok {
		return fn()
	}
	log.Warn("FTS is unsupported on %s falling back to using standard DB queries", t)
	return db.NewIndexer()
}
