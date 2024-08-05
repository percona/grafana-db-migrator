package common

import (
	"database/sql"
)

type Folder struct {
	ID             int
	Slug           string
	ParentFolderID int
	Dashboards     map[string]Dashboard
}

type Tree struct {
	SubFolders map[string]*Tree
	*Folder
}

func GetTree(db *sql.DB) (*Tree, map[int]*Folder, error) {
	folders, err := GetFolders(db)
	if err != nil {
		return nil, nil, err
	}
	tree := GenerateTree(0, folders)
	return tree, folders, nil
}

func GenerateTree(id int, folders map[int]*Folder) *Tree {
	t := &Tree{SubFolders: make(map[string]*Tree), Folder: &Folder{ID: id}}
	for i, folder := range folders {
		if folder.ParentFolderID == id {
			subTree := GenerateTree(folder.ID, folders)
			subTree.Folder = folder
			t.SubFolders[folder.Slug] = subTree
			delete(folders, i)
		}
	}
	return t
}

type Dashboard struct {
	ID       int
	Slug     string
	FolderID int
}

func GetFolders(conn *sql.DB) (map[int]*Folder, error) {
	folders := make(map[int]*Folder)
	rows, err := conn.Query("SELECT id,slug, folder_id FROM dashboard WHERE is_folder=1")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var id int
	var folderSlug string
	var parentFolder int
	for rows.Next() {
		err = rows.Scan(&id, &folderSlug, &parentFolder)
		if err != nil {
			return nil, err
		}
		folders[id] = &Folder{
			ID:             id,
			Slug:           folderSlug,
			ParentFolderID: parentFolder,
		}
	}
	return folders, nil
}
