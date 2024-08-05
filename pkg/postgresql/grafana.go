package postgresql

import (
	"fmt"
	"github.com/percona/grafana-db-migrator/pkg/common"
)

func (db *DB) FixFolderID(sqliteFolders *common.Tree) error {
	return db.fixFolders(sqliteFolders)
}

func (db *DB) fixFolders(sqliteFolders *common.Tree) error {
	folders, err := db.getFolders(sqliteFolders.ID)
	if err != nil {
		return err
	}
	for slug, sqliteFolder := range sqliteFolders.SubFolders {
		if pgFolder, ok := folders[slug]; ok {
			if pgFolder.ID != sqliteFolder.ID {
				res, err := db.conn.Exec("UPDATE dashboard SET id = $1 WHERE id = $2;", sqliteFolder.ID, pgFolder.ID)
				if err != nil {
					return err
				}
				db.log.Infof("💡 Replace folder id for %v to %v", pgFolder.ID, sqliteFolder.ID)
				count, err := res.RowsAffected()
				if err != nil {
					return err
				}
				db.log.Infof("💡 %v rows was fixed", count)
			}
			err = db.fixFolders(sqliteFolder)
			if err != nil {
				return err
			}
		} else {
			db.log.Warnf("⚠️couldn't find copy of folder %s in PG", sqliteFolder.Slug)
		}
	}
	return nil
}

func (db *DB) getFolders(id int) (map[string]*common.Folder, error) {
	folders := make(map[string]*common.Folder)
	rows, err := db.conn.Query("select id, slug, folder_id from dashboard where folder_id = $1 and is_folder=TRUE", id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var folderSlug string
		var parentFolder int
		err = rows.Scan(&id, &folderSlug, &parentFolder)
		if err != nil {
			return nil, err
		}
		folders[folderSlug] = &common.Folder{
			ID:             id,
			Slug:           folderSlug,
			ParentFolderID: parentFolder,
		}
	}
	return folders, nil
}

func (db *DB) ChangeHEXToText() error {
	for _, change := range HexDataChanges {
		db.log.Infof("💡 Replace hex values for %v in %v", change.ColumnName, change.Table)
		stmt := "UPDATE " + change.Table + " SET " + change.ColumnName + " = convert_from(" + change.ColumnName + "::bytea, 'UTF8') WHERE starts_with(" + change.ColumnName + ", '\\x')"
		db.log.Debugln("Executing: ", stmt)
		res, err := db.conn.Exec(stmt)
		if err != nil {
			return fmt.Errorf("couldn't update table %s: %q", change.Table, err)
		}
		count, _ := res.RowsAffected()
		db.log.Infof("💡 %v rows was fixed %s", count, change.Table)
	}
	return nil
}

func (db *DB) FixHomeDashboard() error {
	_, err := db.conn.Exec("UPDATE preferences SET home_dashboard_id=0 WHERE org_id=1;")
	if err != nil {
		return err
	}
	return nil
}

func (db *DB) ChangeCharToText() error {
	// TODO This may break grafana migrations in the future. We'll need to find better solution
	_, err := db.conn.Exec("ALTER TABLE tag ALTER COLUMN key TYPE text;")
	if err != nil {
		return err
	}
	_, err = db.conn.Exec("ALTER TABLE tag ALTER COLUMN value TYPE text;")
	if err != nil {
		return err
	}
	return nil
}

// HexChange documents a table that needs to be changed
// and specificly which Columns need to be changed.
type HexChange struct {
	Table string
	// Name of the column where value is stored
	ColumnName string
}

var HexDataChanges = []HexChange{
	{
		Table:      "library_element",
		ColumnName: "model",
	},
	{
		Table:      "data_source",
		ColumnName: "json_data",
	},
	{
		Table:      "preferences",
		ColumnName: "json_data",
	},
}
