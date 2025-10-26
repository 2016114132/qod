package data

import (
	"context"
	"database/sql"
	"slices"
	"time"

	"github.com/lib/pq"
)

// We will have the permissions in a slice which we will be able to search
type Permissions []string

// Is the permission code found for the Permissions slice
func (p Permissions) Include(code string) bool {
	return slices.Contains(p, code)
}

// Setup our model
type PermissionModel struct {
	DB *sql.DB
}

// What are all the permissions associated with the user
func (p PermissionModel) GetAllForUser(userID int64) (Permissions, error) {
	query := `
               SELECT permissions.code
               FROM permissions 
               INNER JOIN users_permissions ON 
               users_permissions.permission_id = permissions.id

INNER JOIN users ON users_permissions.user_id = users.id
             WHERE users.id = $1
          `
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rows, err := p.DB.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	// Ensure to release resources after use
	defer rows.Close()
	// Store the permissions for the user in our slice
	var permissions Permissions
	for rows.Next() {
		var permission string

		err := rows.Scan(&permission)
		if err != nil {
			return nil, err
		}
		permissions = append(permissions, permission)
	} // end of for loop

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return permissions, nil

}

// Add permissions for the user. Notice that this function accepts
// multiple permissions (...string)
func (p PermissionModel) AddForUser(userID int64, codes ...string) error {
	query := `
        INSERT INTO users_permissions
        SELECT $1, permissions.id FROM permissions 
        WHERE permissions.code = ANY($2)
       `
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// slices need to be converted to arrays to work in PostgreSQL
	_, err := p.DB.ExecContext(ctx, query, userID, pq.Array(codes))

	return err
}
