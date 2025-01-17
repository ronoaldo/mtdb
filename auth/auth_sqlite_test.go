package auth_test

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/minetest-go/mtdb/auth"
	"github.com/minetest-go/mtdb/types"
	"github.com/minetest-go/mtdb/wal"
	"github.com/stretchr/testify/assert"
)

func TestEmptySQliteRepo(t *testing.T) {
	// open db
	db, err := sql.Open("sqlite3", ":memory:")
	assert.NoError(t, err)
	repo := auth.NewAuthRepository(db, types.DATABASE_SQLITE)
	assert.NotNil(t, repo)

	// no such table
	_, err = repo.GetByUsername("test")
	assert.Error(t, err)
}

func TestSQliteRepo(t *testing.T) {
	// init stuff
	dbfile, err := os.CreateTemp(os.TempDir(), "auth.sqlite")
	assert.NoError(t, err)
	assert.NotNil(t, dbfile)
	copyFileContents("testdata/auth.wal.sqlite", dbfile.Name())

	// open db
	db, err := sql.Open("sqlite3", "file:"+dbfile.Name())
	assert.NoError(t, err)
	repo := auth.NewAuthRepository(db, types.DATABASE_SQLITE)
	assert.NotNil(t, repo)

	// existing entry
	entry, err := repo.GetByUsername("test")
	assert.NoError(t, err)
	assert.NotNil(t, entry)
	assert.Equal(t, "test", entry.Name)
	assert.Equal(t, "#1#TxqLUa/uEJvZzPc3A0xwpA#oalXnktlS0bskc7bccsoVTeGwgAwUOyYhhceBu7wAyITkYjCtrzcDg6W5Co5V+oWUSG13y7TIoEfIg6rafaKzAbwRUC9RVGCeYRIUaa0hgEkIe9VkDmpeQ/kfF8zT8p7prOcpyrjWIJR+gmlD8Bf1mrxoPoBLDbvmxkcet327kQ9H4EMlIlv+w3XCufoPGFQ1UrfWiVqqK8dEmt/ldLPfxiK1Rg8MkwswEekymP1jyN9Cpq3w8spVVcjsxsAzI5M7QhSyqMMrIThdgBsUqMBOCULdV+jbRBBiA/ClywtZ8vvBpN9VGqsQuhmQG0h5x3fqPyR2XNdp9Ocm3zHBoJy/w", entry.Password)
	assert.Equal(t, int64(2), *entry.ID)
	assert.Equal(t, 1649603232, entry.LastLogin)

	// non-existing entry
	entry, err = repo.GetByUsername("bogus")
	assert.NoError(t, err)
	assert.Nil(t, entry)

	// create entry
	new_entry := &auth.AuthEntry{
		Name:      "createduser",
		Password:  "blah",
		LastLogin: 456,
	}
	assert.NoError(t, repo.Create(new_entry))
	assert.NotNil(t, new_entry.ID)

	// check newly created entry
	entry, err = repo.GetByUsername("createduser")
	assert.NoError(t, err)
	assert.NotNil(t, entry)
	assert.Equal(t, new_entry.Name, entry.Name)
	assert.Equal(t, new_entry.Password, entry.Password)
	assert.Equal(t, *new_entry.ID, *entry.ID)
	assert.Equal(t, new_entry.LastLogin, entry.LastLogin)

	// change things
	new_entry.Name = "x"
	new_entry.Password = "y"
	new_entry.LastLogin = 123
	assert.NoError(t, repo.Update(new_entry))
	entry, err = repo.GetByUsername("x")
	assert.NoError(t, err)
	assert.NotNil(t, entry)
	assert.Equal(t, new_entry.Name, entry.Name)
	assert.Equal(t, new_entry.Password, entry.Password)
	assert.Equal(t, *new_entry.ID, *entry.ID)
	assert.Equal(t, new_entry.LastLogin, entry.LastLogin)

	// remove new user
	assert.NoError(t, repo.Delete(*new_entry.ID))
	entry, err = repo.GetByUsername("x")
	assert.NoError(t, err)
	assert.Nil(t, entry)

}

func TestSQlitePrivRepo(t *testing.T) {
	// init stuff
	dbfile, err := os.CreateTemp(os.TempDir(), "auth.sqlite")
	assert.NoError(t, err)
	assert.NotNil(t, dbfile)
	copyFileContents("testdata/auth.wal.sqlite", dbfile.Name())

	// open db
	db, err := sql.Open("sqlite3", "file:"+dbfile.Name())
	assert.NoError(t, err)
	repo := auth.NewPrivilegeRepository(db, types.DATABASE_SQLITE)
	assert.NotNil(t, repo)

	// read privs
	list, err := repo.GetByID(2)
	assert.NoError(t, err)
	assert.NotNil(t, list)
	assert.Equal(t, 2, len(list))

	privs := make(map[string]bool)
	for _, e := range list {
		privs[e.Privilege] = true
	}
	assert.True(t, privs["interact"])
	assert.True(t, privs["shout"])

	// create
	assert.NoError(t, repo.Create(&auth.PrivilegeEntry{ID: 2, Privilege: "stuff"}))

	// verify
	list, err = repo.GetByID(2)
	assert.NoError(t, err)
	assert.NotNil(t, list)
	assert.Equal(t, 3, len(list))

	privs = make(map[string]bool)
	for _, e := range list {
		privs[e.Privilege] = true
	}
	assert.True(t, privs["interact"])
	assert.True(t, privs["shout"])
	assert.True(t, privs["stuff"])

	// delete
	assert.NoError(t, repo.Delete(2, "stuff"))

	list, err = repo.GetByID(2)
	assert.NoError(t, err)
	assert.NotNil(t, list)
	assert.Equal(t, 2, len(list))

	privs = make(map[string]bool)
	for _, e := range list {
		privs[e.Privilege] = true
	}
	assert.True(t, privs["interact"])
	assert.True(t, privs["shout"])
}

func TestSqliteAuthRepo(t *testing.T) {
	// open db
	dbfile, err := os.CreateTemp(os.TempDir(), "auth.sqlite")
	assert.NoError(t, err)
	assert.NotNil(t, dbfile)
	db, err := sql.Open("sqlite3", "file:"+dbfile.Name())
	assert.NoError(t, err)
	assert.NoError(t, auth.MigrateAuthDB(db, types.DATABASE_SQLITE))
	assert.NoError(t, wal.EnableWAL(db))

	auth_repo := auth.NewAuthRepository(db, types.DATABASE_SQLITE)
	priv_repo := auth.NewPrivilegeRepository(db, types.DATABASE_SQLITE)

	testAuthRepository(t, auth_repo, priv_repo)
}
