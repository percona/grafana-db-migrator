package sqlite

import (
	"database/sql"
	"github.com/percona/grafana-db-migrator/pkg/common"

	_ "modernc.org/sqlite"
)

func GetFolders(dbFile string) (*common.Tree, map[int]*common.Folder, error) {
	db, err := sql.Open("sqlite", dbFile)
	if err != nil {
		return nil, nil, err
	}
	return common.GetTree(db)
}
