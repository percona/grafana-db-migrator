package postgresql

import (
	"encoding/hex"
	"github.com/sirupsen/logrus"
	"strings"
)

func reverseMap(m map[string]int) map[int]string {
	n := make(map[int]string, len(m))
	for k, v := range m {
		n[v] = k
	}
	return n
}

// dashboardMapping is map[dashboardSlug]folderName
func (db *DB) FixFolderID(dashboardsMapping map[string]string, logger *logrus.Logger) error {
	foldersSlugToID := make(map[string]int)

	// Get folders
	rows, err := db.conn.Query("SELECT id,slug FROM dashboard WHERE is_folder=true;")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var folderId int
		var folderSlug string

		err = rows.Scan(&folderId, &folderSlug)
		if err != nil {
			return err
		}
		foldersSlugToID[folderSlug] = folderId
	}
	err = rows.Err()
	if err != nil {
		return err
	}
	foldersIDToSlug := reverseMap(foldersSlugToID)

	// Get dashboards from postgres
	rows, err = db.conn.Query("SELECT slug, folder_id FROM dashboard WHERE is_folder=false;")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var dashboardFolderID int
		var dashboardSlug string

		err = rows.Scan(&dashboardSlug, &dashboardFolderID)
		if err != nil {
			return err
		}
		currentFolder := foldersIDToSlug[dashboardFolderID]
		targetFolder := dashboardsMapping[dashboardSlug]
		if currentFolder != targetFolder {
			targetFolderId := foldersSlugToID[targetFolder]
			// it means dashboard not exist
			if targetFolderId == 0 {
				continue
			}
			logger.Infof("💡 Replace folder id for %v to %v", dashboardSlug, targetFolderId)
			res, err := db.conn.Exec("UPDATE dashboard SET folder_id = $1 WHERE slug = $2;", targetFolderId, dashboardSlug)
			if err != nil {
				return err
			}
			count, err := res.RowsAffected()
			if err != nil {
				return err
			}
			logger.Infof("💡 %v rows was fixed", count)
		}

	}

	return nil
}

func (db *DB) ChangeHEXToText(logger *logrus.Logger) error {
	for _, change := range HexDataChanges {
		for _, column := range change.Columns {
			err := db.changeHexToTextInTable(change, column, logger)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
func (db *DB) changeHexToTextInTable(change TableChange, column Column, logger *logrus.Logger) error {
	rows, err := db.conn.Query("SELECT id," + column.Name + " FROM " + change.Table)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id int
		var hexColumn string
		err = rows.Scan(&id, &hexColumn)
		if err != nil {
			return err
		}
		newValue, err := hex.DecodeString(strings.TrimPrefix(hexColumn, `\x`))
		if err != nil {
			return err
		}
		res, err := db.conn.Exec("UPDATE "+change.Table+" SET "+column.Name+" = $1 WHERE id = $2;", newValue, id)
		if err != nil {
			return err
		}
		count, err := res.RowsAffected()
		if err != nil {
			return err
		}
		logger.Infof("💡 %v rows was fixed", count)
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

var HexDataChanges = []TableChange{
	{
		Table: "library_element",
		Columns: []Column{
			{
				Name: "model",
			},
		},
	},
	{
		Table: "data_keys",
		Columns: []Column{
			{
				Name: "encrypted_data",
			},
		},
	},
	{
		Table: "data_source",
		Columns: []Column{
			{
				Name: "json_data",
			},
		},
	},
	{
		Table: "preferences",
		Columns: []Column{
			{
				Name: "json_data",
			},
		},
	},
}
