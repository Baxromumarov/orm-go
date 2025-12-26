package main

import (
	"context"

	orm "github.com/baxromumarov/orm-go"
)

func main() {
	db, err := orm.Connect(&orm.Config{
		Dsn: "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable",
		Ctx: context.Background(),
	})
	if err != nil {
		panic(err)
	}
	defer db.Close()

	type User struct {
		ID      int64  `orm:"id"`
		Name    string `orm:"name"`
		Balance int    `orm:"balance"`
	}

	user := User{Name: "Uzb", Balance: 1000, ID: 12}

	// Update
	db.Model(&user).
		Update(context.Background())

	//err = db.
	//	Model(&user).
	//	Insert(context.Background()).
	//	AutoTableName().
	//	Returning(
	//		"id",
	//		"name",
	//		"balance",
	//	).
	//	Exec()
	//if err != nil {
	//	fmt.Println(">>> Error:", err)
	//	return
	//}
	//
	//fmt.Println(">>> Result:", user)

}
